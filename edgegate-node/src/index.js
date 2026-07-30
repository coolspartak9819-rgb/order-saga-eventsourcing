import fs from 'node:fs';
import http from 'node:http';
import http2 from 'node:http2';
import { timingSafeEqual } from 'node:crypto';
import { resolve } from 'node:path';
import { createClient } from 'redis';
import { loadConfig, validateConfig } from './config.js';
import { Gateway } from './gateway.js';
import { ConfigStore } from './config-store.js';
import { readBody } from './waf.js';

const configPath = resolve(process.env.CONFIG_PATH || 'config.json');
let config = await loadConfig(configPath);
const redis = createClient({ url: config.redisUrl });
redis.on('error', error => console.error(JSON.stringify({ level: 'error', event: 'redis_error', message: error.message })));
await redis.connect();

const gateway = new Gateway(redis);
const configStore = new ConfigStore(redis);
const stored = await configStore.current();
if (stored.config) config = stored.config;
await gateway.applyConfig(config, configPath);
await configStore.subscribe(async (nextConfig, version) => {
  try {
    await gateway.applyConfig(nextConfig, configPath);
    config = nextConfig;
    gateway.metrics.recordConfigReload('success');
    console.log(JSON.stringify({ level: 'info', event: 'distributed_config_applied', version }));
  } catch (error) {
    gateway.metrics.recordConfigReload('failure');
    console.error(JSON.stringify({ level: 'error', event: 'distributed_config_rejected', version, message: error.message }));
  }
});

const handler = (request, response) => {
  if (request.url === '/healthz') return gateway.send(response, 200, 'ok');
  if (request.url === '/readyz') return redis.isReady ? gateway.send(response, 200, 'ready') : gateway.send(response, 503, 'not ready');
  if (request.url === '/metrics') {
    response.writeHead(200, { 'content-type': 'text/plain; version=0.0.4; charset=utf-8' });
    return response.end(gateway.metrics.render(gateway.routes));
  }
  if (request.url.startsWith('/control/config')) {
    void handleControl(request, response).catch(error => {
      console.error(JSON.stringify({ level: 'error', event: 'control_plane_error', message: error.message }));
      if (!response.headersSent) gateway.send(response, 503, 'control plane unavailable');
      else response.destroy(error);
    });
    return;
  }
  gateway.handle(request, response);
};

async function handleControl(request, response) {
  const controlKey = process.env.EDGEGATE_CONTROL_KEY;
  if (!controlKey) return gateway.send(response, 404, 'control plane disabled');
  if (!secureEqual(request.headers['x-control-key'], controlKey)) return gateway.send(response, 401, 'invalid control key');
  if (request.method === 'GET') {
    const current = await configStore.current();
    response.writeHead(200, { 'content-type': 'application/json' });
    return response.end(JSON.stringify({ version: current.version, config: current.config ?? config }));
  }
  if (request.method !== 'PUT') return gateway.send(response, 405, 'method not allowed');
  try {
    const body = JSON.parse((await readBody(request, 1024 * 1024)).toString('utf8'));
    if (!Number.isInteger(body.expectedVersion) || !body.config) return gateway.send(response, 400, 'expectedVersion and config are required');
    validateConfig(body.config);
    const version = await configStore.update(body.config, body.expectedVersion);
    response.writeHead(202, { 'content-type': 'application/json' });
    response.end(JSON.stringify({ version, status: 'accepted' }));
  } catch (error) {
    const statusCode = error.statusCode || 400;
    response.writeHead(statusCode, { 'content-type': 'application/json' });
    response.end(JSON.stringify({ error: error.message, currentVersion: error.currentVersion }));
  }
}

function secureEqual(candidate, expected) {
  if (typeof candidate !== 'string') return false;
  const candidateBuffer = Buffer.from(candidate);
  const expectedBuffer = Buffer.from(expected);
  return candidateBuffer.length === expectedBuffer.length && timingSafeEqual(candidateBuffer, expectedBuffer);
}

let server;
if (process.env.TLS_CERT && process.env.TLS_KEY) {
  server = http2.createSecureServer({
    allowHTTP1: true,
    cert: fs.readFileSync(process.env.TLS_CERT),
    key: fs.readFileSync(process.env.TLS_KEY)
  }, handler);
} else {
  server = http.createServer(handler);
}

server.listen(config.listen.port, config.listen.host, () => {
  console.log(`EdgeGate Node listening on ${config.listen.host}:${config.listen.port}`);
});

fs.watchFile(configPath, { interval: 1000 }, async () => {
  try {
    const nextConfig = await loadConfig(configPath);
    await gateway.applyConfig(nextConfig, configPath);
    config = nextConfig;
    console.log('configuration reloaded');
  } catch (error) {
    console.error(`configuration reload rejected: ${error.message}`);
  }
});

async function shutdown(signal) {
  console.log(`received ${signal}, shutting down`);
  fs.unwatchFile(configPath);
  gateway.close();
  server.close(async () => {
    await configStore.close();
    await redis.quit();
    process.exit(0);
  });
  setTimeout(() => process.exit(1), 10_000).unref();
}

process.on('SIGINT', () => shutdown('SIGINT'));
process.on('SIGTERM', () => shutdown('SIGTERM'));
