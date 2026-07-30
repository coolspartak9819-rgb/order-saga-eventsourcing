import { readFile } from 'node:fs/promises';

const STRATEGIES = new Set(['round_robin', 'least_connections', 'consistent_hash']);

export async function loadConfig(path) {
  const config = JSON.parse(await readFile(path, 'utf8'));
  config.listen ??= { host: '0.0.0.0', port: 8080 };
  config.redisUrl ??= 'redis://localhost:6379';
  validateConfig(config);
  return config;
}

export function validateConfig(config) {
  if (!config || typeof config !== 'object') throw new Error('config must be an object');
  if (!Array.isArray(config.routes) || config.routes.length === 0) throw new Error('at least one route is required');

  const paths = new Set();
  for (const [index, route] of config.routes.entries()) {
    if (typeof route.path !== 'string' || !route.path.startsWith('/')) throw new Error(`routes[${index}].path must start with /`);
    if (paths.has(route.path)) throw new Error(`duplicate route path: ${route.path}`);
    paths.add(route.path);
    if (!Array.isArray(route.backends) || route.backends.length === 0) throw new Error(`route ${route.path} has no backends`);
    for (const backend of route.backends) {
      const url = new URL(backend);
      if (!['http:', 'https:'].includes(url.protocol)) throw new Error(`unsupported backend protocol: ${url.protocol}`);
    }
    route.strategy ??= 'round_robin';
    if (!STRATEGIES.has(route.strategy)) throw new Error(`unsupported strategy: ${route.strategy}`);
    route.plugins ??= ['request-id', 'access-log'];
    if (route.rateLimit && (!(route.rateLimit.requestsPerSecond > 0) || !(route.rateLimit.burst > 0))) {
      throw new Error(`invalid rate limit for route ${route.path}`);
    }
  }
}
