import { createHash } from 'node:crypto';

export class LoadBalancer {
  constructor(backends, strategy = 'round_robin') {
    this.backends = backends.map(url => ({ url: new URL(url), active: 0 }));
    this.strategy = strategy;
    this.cursor = 0;
  }

  acquire(key = '') {
    let backend;
    if (this.strategy === 'least_connections') {
      backend = this.backends.reduce((best, current) => current.active < best.active ? current : best);
    } else if (this.strategy === 'consistent_hash') {
      const hash = createHash('sha256').update(key).digest().readUInt32BE(0);
      backend = this.backends[hash % this.backends.length];
    } else {
      backend = this.backends[this.cursor % this.backends.length];
      this.cursor = (this.cursor + 1) % this.backends.length;
    }
    backend.active += 1;
    return {
      backend,
      release: () => { backend.active = Math.max(0, backend.active - 1); }
    };
  }
}
