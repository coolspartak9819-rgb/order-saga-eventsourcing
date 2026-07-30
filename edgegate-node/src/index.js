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
import { WAFPolicyStore } from './waf-policy-store.js';

const configPath = resolve(process.env.CONFIG_PATH || 'config.json');
let config = await loadConfig(configPath);
const redis = createClient({ url: config.redisUrl });
redis.on('error', error => console.error(JSON.stringify({ level: 'error', event: 'redis_error', message: error.message })));
await redis.connect();

const gateway = new Gateway(redis);
const configStore = new ConfigStore(redis);
const wafPolicyStore = new WAFPolicyStore(redis);
const stored = await configStore.current();
if (stored.config) config = stored.config;
const currentWAF = await wafPolicyStore.current();
gateway.applyWAFPolicy(currentWAF.policy);
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
await wafPolicyStore.subscribe(async (policy, version) => {
  try {
    gateway.applyWAFPolicy(policy);
    console.log(JSON.stringify({ level: 'info', event: 'waf_policy_applied', version, mode: policy.mode }));
  } catch (error) {
    console.error(JSON.stringify({ level: 'error', event: 'waf_policy_rejected', version, message: error.message }));
  }
});

const handler = (request, response) => {
  const pathname = new URL(request.url, 'http://edgegate.local').pathname;
  if (pathname === '/healthz') return gateway.send(response, 200, 'ok');
  if (pathname === '/readyz') return redis.isReady ? gateway.send(response, 200, 'ready') : gateway.send(response, 503, 'not ready');
  if (pathname === '/metrics') {
    response.writeHead(200, { 'content-type': 'text/plain; version=0.0.4; charset=utf-8' });
    return response.end(gateway.metrics.render(gateway.routes));
  }
  if (pathname.startsWith('/control/')) {
    void handleControl(request, response, pathname).catch(error => {
      console.error(JSON.stringify({ level: 'error', event: 'control_plane_error', message: error.message }));
      if (!response.headersSent) gateway.send(response, 503, 'control plane unavailable');
      else response.destroy(error);
    });
    return;
  }
  gateway.handle(request, response);
};

async function handleControl(request, response, pathname) {
  const controlKey = process.env.EDGEGATE_CONTROL_KEY;
  if (!controlKey) return gateway.send(response, 404, 'control plane disabled');
  if (!secureEqual(request.headers['x-control-key'], controlKey)) return gateway.send(response, 401, 'invalid control key');
  if (pathname === '/control/config') return handleConfigControl(request, response);
  if (pathname.startsWith('/control/waf')) return handleWAFControl(request, response, pathname);
  return gateway.send(response, 404, 'control endpoint not found');
}

async function handleConfigControl(request, response) {
  if (request.method === 'GET') {
    const current = await configStore.current();
    return sendJSON(response, 200, { version: current.version, config: current.config ?? config });
  }
  if (request.method !== 'PUT') return gateway.send(response, 405, 'method not allowed');
  try {
    const body = JSON.parse((await readBody(request, 1024 * 1024)).toString('utf8'));
    if (!Number.isInteger(body.expectedVersion) || !body.config) return gateway.send(response, 400, 'expectedVersion and config are required');
    validateConfig(body.config);
    const version = await configStore.update(body.config, body.expectedVersion);
    return sendJSON(response, 202, { version, status: 'accepted' });
  } catch (error) {
    return sendJSON(response, error.statusCode || 400, { error: error.message, currentVersion: error.currentVersion });
  }
}

async function handleWAFControl(request, response, pathname) {
  try {
    if (pathname === '/control/waf' && request.method === 'GET') {
      return sendJSON(response, 200, await wafPolicyStore.current());
    }
    if (pathname === '/control/waf' && request.method === 'PUT') {
      const body = await readJSON(request);
      const version = await wafPolicyStore.update(body.policy, body.expectedVersion, { actor: request.headers['x-actor'] });
      return sendJSON(response, 202, { version, status: 'accepted' });
    }
    if (pathname === '/control/waf/history' && request.method === 'GET') {
      const limit = new URL(request.url, 'http://edgegate.local').searchParams.get('limit');
      return sendJSON(response, 200, { entries: await wafPolicyStore.history(limit) });
    }
    if (pathname === '/control/waf/rollback' && request.method === 'POST') {
      const body = await readJSON(request);
      const version = await wafPolicyStore.rollback(body.targetVersion, body.expectedVersion, request.headers['x-actor']);
      return sendJSON(response, 202, { version, status: 'accepted' });
    }
    if (pathname === '/control/waf/false-positive' && request.method === 'POST') {
      const body = await readJSON(request);
      if (typeof body.rule !== 'string' || !body.rule.trim()) return sendJSON(response, 400, { error: 'rule is required' });
      gateway.metrics.recordWAFFalsePositive(body.rule.trim().slice(0, 128));
      console.warn(JSON.stringify({ level: 'warn', event: 'waf_false_positive', rule: body.rule, actor: request.headers['x-actor'] || 'unknown' }));
      return sendJSON(response, 202, { status: 'recorded' });
    }
    return gateway.send(response, 405, 'method not allowed');
  } catch (error) {
    return sendJSON(response, error.statusCode || 400, { error: error.message, currentVersion: error.currentVersion });
  }
}

async function readJSON(request) {
  return JSON.parse((await readBody(request, 1024 * 1024)).toString('utf8'));
}

function sendJSON(response, statusCode, payload) {
  response.writeHead(statusCode, { 'content-type': 'application/json' });
  response.end(JSON.stringify(payload));
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
    await wafPolicyStore.close();
    await redis.quit();
    process.exit(0);
  });
  setTimeout(() => process.exit(1), 10_000).unref();
}

process.on('SIGINT', () => shutdown('SIGINT'));
process.on('SIGTERM', () => shutdown('SIGTERM'));
