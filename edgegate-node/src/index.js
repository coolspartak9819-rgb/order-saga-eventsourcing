import fs from 'node:fs';
import http from 'node:http';
import http2 from 'node:http2';
import { resolve } from 'node:path';
import { createClient } from 'redis';
import { loadConfig } from './config.js';
import { Gateway } from './gateway.js';

const configPath = resolve(process.env.CONFIG_PATH || 'config.json');
let config = await loadConfig(configPath);
const redis = createClient({ url: config.redisUrl });
redis.on('error', error => console.error(JSON.stringify({ level: 'error', event: 'redis_error', message: error.message })));
await redis.connect();

const gateway = new Gateway(redis);
await gateway.applyConfig(config, configPath);

const handler = (request, response) => {
  if (request.url === '/healthz') return gateway.send(response, 200, 'ok');
  if (request.url === '/readyz') return redis.isReady ? gateway.send(response, 200, 'ready') : gateway.send(response, 503, 'not ready');
  gateway.handle(request, response);
};

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
    await redis.quit();
    process.exit(0);
  });
  setTimeout(() => process.exit(1), 10_000).unref();
}

process.on('SIGINT', () => shutdown('SIGINT'));
process.on('SIGTERM', () => shutdown('SIGTERM'));
