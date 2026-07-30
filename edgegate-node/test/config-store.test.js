import test from 'node:test';
import assert from 'node:assert/strict';
import { ConfigStore } from '../src/config-store.js';

test('stores versioned configuration with optimistic concurrency', async () => {
  const redis = new FakeRedis();
  const store = new ConfigStore(redis);
  const config = { routes: [{ path: '/api', backends: ['http://backend:8080'] }] };
  const version = await store.update(config, 0);
  assert.equal(version, 1);
  const current = await store.current();
  assert.equal(current.version, 1);
  assert.equal(current.config.routes[0].path, '/api');

  await assert.rejects(() => store.update(config, 0), error => {
    assert.equal(error.statusCode, 409);
    assert.equal(error.currentVersion, 1);
    return true;
  });
});

class FakeRedis {
  constructor() {
    this.version = 0;
    this.document = null;
  }

  async mGet() {
    return [String(this.version), this.document];
  }

  async eval(_script, { arguments: values }) {
    const expected = Number(values[0]);
    if (expected !== this.version) return [0, this.version];
    this.version += 1;
    this.document = values[1];
    return [1, this.version];
  }
}
