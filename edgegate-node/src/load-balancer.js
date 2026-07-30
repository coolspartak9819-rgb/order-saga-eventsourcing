import { createHash } from 'node:crypto';
import http from 'node:http';
import https from 'node:https';

export class NoHealthyBackendsError extends Error {
  constructor() {
    super('no healthy backends available');
    this.statusCode = 503;
  }
}

export class LoadBalancer {
  constructor(backends, strategy = 'round_robin', options = {}) {
    this.backends = backends.map(url => ({
      url: new URL(url),
      active: 0,
      healthy: true,
      failures: 0,
      circuitState: 'closed',
      openedAt: 0
    }));
    this.strategy = strategy;
    this.cursor = 0;
    this.healthCheck = options.healthCheck ?? { enabled: false };
    this.circuitBreaker = options.circuitBreaker ?? { failureThreshold: 3, cooldownMs: 10_000 };
  }

  acquire(key = '') {
    const candidates = this.backends.filter(backend => this.isAvailable(backend));
    if (candidates.length === 0) throw new NoHealthyBackendsError();
    let backend;
    if (this.strategy === 'least_connections') {
      backend = candidates.reduce((best, current) => current.active < best.active ? current : best);
    } else if (this.strategy === 'consistent_hash') {
      const hash = createHash('sha256').update(key).digest().readUInt32BE(0);
      backend = candidates[hash % candidates.length];
    } else {
      backend = candidates[this.cursor % candidates.length];
      this.cursor = (this.cursor + 1) % candidates.length;
    }
    backend.active += 1;
    return {
      backend,
      release: () => { backend.active = Math.max(0, backend.active - 1); },
      recordSuccess: () => this.recordSuccess(backend),
      recordFailure: () => this.recordFailure(backend)
    };
  }

  isAvailable(backend) {
    if (!backend.healthy) return false;
    if (backend.circuitState !== 'open') return backend.circuitState !== 'half_open' || backend.active === 0;
    if (Date.now() - backend.openedAt < this.circuitBreaker.cooldownMs) return false;
    backend.circuitState = 'half_open';
    return backend.active === 0;
  }

  recordSuccess(backend) {
    backend.failures = 0;
    backend.circuitState = 'closed';
  }

  recordFailure(backend) {
    backend.failures += 1;
    if (backend.circuitState === 'half_open' || backend.failures >= this.circuitBreaker.failureThreshold) {
      backend.circuitState = 'open';
      backend.openedAt = Date.now();
    }
  }

  startHealthChecks() {
    if (!this.healthCheck.enabled) return () => {};
    const checkAll = () => Promise.allSettled(this.backends.map(backend => this.checkBackend(backend)));
    void checkAll();
    const timer = setInterval(checkAll, this.healthCheck.intervalMs);
    timer.unref();
    return () => clearInterval(timer);
  }

  checkBackend(backend) {
    return new Promise(resolve => {
      const target = new URL(this.healthCheck.path, backend.url);
      const transport = target.protocol === 'https:' ? https : http;
      const controller = new AbortController();
      const timeout = setTimeout(() => controller.abort(), this.healthCheck.timeoutMs);
      let finished = false;
      const finish = healthy => {
        if (finished) return;
        finished = true;
        clearTimeout(timeout);
        backend.healthy = healthy;
        resolve(healthy);
      };
      const request = transport.request(target, { method: 'GET', signal: controller.signal }, response => {
        response.resume();
        finish(response.statusCode >= 200 && response.statusCode < 400);
      });
      request.on('error', () => finish(false));
      request.end();
    });
  }
}
