import http from 'node:http';
import https from 'node:https';
import { dirname } from 'node:path';
import { LoadBalancer } from './load-balancer.js';
import { WAF, readBody } from './waf.js';
import { RedisTokenBucket } from './rate-limiter.js';
import { compose, resolvePlugins } from './plugins.js';
import { Metrics } from './metrics.js';

export class Gateway {
  constructor(redis) {
    this.redis = redis;
    this.routes = [];
    this.metrics = new Metrics();
  }

  async applyConfig(config, configPath) {
    const routes = [];
    for (const routeConfig of config.routes) {
      const balancer = new LoadBalancer(routeConfig.backends, routeConfig.strategy, {
        healthCheck: routeConfig.healthCheck,
        circuitBreaker: routeConfig.circuitBreaker
      });
      const waf = routeConfig.waf?.enabled ? new WAF(routeConfig.waf.extraRules) : null;
      const limiter = routeConfig.rateLimit ? new RedisTokenBucket(this.redis, routeConfig.rateLimit) : null;
      const plugins = await resolvePlugins(routeConfig.plugins, dirname(configPath));

      const terminal = async context => this.proxy(context, balancer);
      const middleware = [];
      if (waf) middleware.push(async (context, next) => {
        context.body = await readBody(context.request, routeConfig.waf.maxBodyBytes);
        const rule = waf.inspect({ url: context.request.url, headers: context.request.headers, body: context.body.toString('utf8') });
        if (rule) {
          this.metrics.recordWAFBlock(rule);
          console.warn(JSON.stringify({ level: 'warn', event: 'waf_block', rule, path: context.request.url }));
          return this.send(context.response, 403, 'request blocked by WAF');
        }
        await next();
      });
      if (limiter) middleware.push(async (context, next) => {
        const key = context.request.headers['x-user-id'] || context.request.socket.remoteAddress || 'unknown';
        if (!await limiter.allow(key)) {
          this.metrics.recordRateLimit(routeConfig.path);
          return this.send(context.response, 429, 'rate limit exceeded');
        }
        await next();
      });
      middleware.push(...plugins);
      routes.push({ path: routeConfig.path, handler: compose(middleware, terminal), balancer, stopHealthChecks: null });
    }
    routes.sort((a, b) => b.path.length - a.path.length);
    for (const route of routes) route.stopHealthChecks = route.balancer.startHealthChecks();
    for (const route of this.routes) route.stopHealthChecks?.();
    this.routes = routes;
  }

  async handle(request, response) {
    const pathname = new URL(request.url, 'http://edgegate.local').pathname;
    const route = this.routes.find(candidate => matchesPrefix(pathname, candidate.path));
    if (!route) return this.send(response, 404, 'route not found');
    const started = process.hrtime.bigint();
    response.once('finish', () => {
      const seconds = Number(process.hrtime.bigint() - started) / 1e9;
      this.metrics.recordRequest(route.path, response.statusCode, seconds);
    });
    try {
      await route.handler({ request, response, body: null, backend: null, routePath: route.path });
    } catch (error) {
      console.error(JSON.stringify({ level: 'error', event: 'gateway_error', message: error.message }));
      if (!response.headersSent) this.send(response, error.statusCode || 503, 'gateway unavailable');
      else response.destroy(error);
    }
  }

  proxy(context, balancer) {
    return new Promise((resolve, reject) => {
      const key = context.request.headers['x-user-id'] || context.request.socket.remoteAddress || '';
      const { backend, release, recordSuccess, recordFailure } = balancer.acquire(key);
      let released = false;
      let outcomeRecorded = false;
      const releaseOnce = () => {
        if (!released) {
          released = true;
          release();
        }
      };
      const successOnce = () => {
        if (!outcomeRecorded) {
          outcomeRecorded = true;
          recordSuccess();
        }
      };
      const failureOnce = () => {
        if (!outcomeRecorded) {
          outcomeRecorded = true;
          recordFailure();
        }
      };
      context.backend = backend;
      const target = new URL(context.request.url, backend.url);
      const transport = target.protocol === 'https:' ? https : http;
      const headers = {
        ...context.request.headers,
        host: target.host,
        'x-forwarded-for': context.request.socket.remoteAddress,
        'x-forwarded-proto': context.request.socket.encrypted ? 'https' : 'http'
      };
      const upstream = transport.request(target, { method: context.request.method, headers }, upstreamResponse => {
        clearTimeout(hardTimeout);
        this.metrics.recordUpstream(backend.url.origin, upstreamResponse.statusCode ?? 502);
        if ((upstreamResponse.statusCode ?? 500) >= 500) failureOnce();
        else successOnce();
        context.response.writeHead(upstreamResponse.statusCode ?? 502, upstreamResponse.headers);
        upstreamResponse.pipe(context.response);
        upstreamResponse.on('end', () => { releaseOnce(); resolve(); });
        upstreamResponse.on('error', error => { failureOnce(); releaseOnce(); reject(error); });
      });
      const hardTimeout = setTimeout(() => upstream.destroy(new Error('upstream timeout')), 10_000);
      upstream.on('error', error => { clearTimeout(hardTimeout); failureOnce(); releaseOnce(); reject(error); });
      context.response.on('close', releaseOnce);
      if (context.body) upstream.end(context.body);
      else context.request.pipe(upstream);
    });
  }

  send(response, statusCode, message) {
    response.writeHead(statusCode, { 'content-type': 'text/plain; charset=utf-8' });
    response.end(`${message}\n`);
  }

  close() {
    for (const route of this.routes) route.stopHealthChecks?.();
  }
}

export function matchesPrefix(pathname, prefix) {
  return prefix === '/' || pathname === prefix || pathname.startsWith(`${prefix}/`);
}
