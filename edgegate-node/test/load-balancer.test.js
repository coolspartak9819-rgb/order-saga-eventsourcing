import test from 'node:test';
import assert from 'node:assert/strict';
import { LoadBalancer, NoHealthyBackendsError } from '../src/load-balancer.js';

test('round robin rotates through backends', () => {
  const balancer = new LoadBalancer(['http://a.local', 'http://b.local'], 'round_robin');
  const first = balancer.acquire();
  const second = balancer.acquire();
  assert.equal(first.backend.url.hostname, 'a.local');
  assert.equal(second.backend.url.hostname, 'b.local');
  first.release();
  second.release();
});

test('least connections selects the backend with fewer active requests', () => {
  const balancer = new LoadBalancer(['http://a.local', 'http://b.local'], 'least_connections');
  const first = balancer.acquire();
  const second = balancer.acquire();
  assert.notEqual(first.backend.url.hostname, second.backend.url.hostname);
  first.release();
  second.release();
});

test('consistent hashing keeps the same key on the same backend', () => {
  const balancer = new LoadBalancer(['http://a.local', 'http://b.local'], 'consistent_hash');
  const first = balancer.acquire('customer-42');
  first.release();
  const second = balancer.acquire('customer-42');
  assert.equal(first.backend.url.href, second.backend.url.href);
  second.release();
});

test('skips unhealthy backends', () => {
  const balancer = new LoadBalancer(['http://a.local', 'http://b.local']);
  balancer.backends[0].healthy = false;
  const selected = balancer.acquire();
  assert.equal(selected.backend.url.hostname, 'b.local');
  selected.release();
  balancer.backends[1].healthy = false;
  assert.throws(() => balancer.acquire(), NoHealthyBackendsError);
});

test('opens and recovers a circuit breaker', () => {
  const balancer = new LoadBalancer(['http://a.local'], 'round_robin', {
    circuitBreaker: { failureThreshold: 2, cooldownMs: 50 },
    healthCheck: { enabled: false }
  });
  for (let attempt = 0; attempt < 2; attempt += 1) {
    const selected = balancer.acquire();
    selected.recordFailure();
    selected.release();
  }
  assert.throws(() => balancer.acquire(), NoHealthyBackendsError);
  balancer.backends[0].openedAt = Date.now() - 100;
  const probe = balancer.acquire();
  assert.equal(probe.backend.circuitState, 'half_open');
  probe.recordSuccess();
  probe.release();
  assert.equal(probe.backend.circuitState, 'closed');
});
