import test from 'node:test';
import assert from 'node:assert/strict';
import { WAFPolicyStore } from '../src/waf-policy-store.js';

test('versions WAF policies and keeps audit history', async () => {
  const redis = new FakeRedis();
  const store = new WAFPolicyStore(redis);
  const first = { mode: 'monitor', rules: [{ id: 'scanner', source: 'masscan' }] };
  assert.equal(await store.update(first, 0, { actor: 'platform@example.com' }), 1);
  assert.equal((await store.current()).policy.mode, 'monitor');

  const second = { mode: 'block', rules: [] };
  assert.equal(await store.update(second, 1, { actor: 'security@example.com' }), 2);
  const history = await store.history();
  assert.deepEqual(history.map(entry => entry.version), [2, 1]);
  assert.equal(history[0].actor, 'security@example.com');

  await assert.rejects(() => store.update(second, 1), error => {
    assert.equal(error.statusCode, 409);
    assert.equal(error.currentVersion, 2);
    return true;
  });
});

test('rollback creates a new audited version', async () => {
  const redis = new FakeRedis();
  const store = new WAFPolicyStore(redis);
  await store.update({ mode: 'monitor' }, 0, { actor: 'operator' });
  await store.update({ mode: 'block' }, 1, { actor: 'operator' });

  assert.equal(await store.rollback(1, 2, 'incident-commander'), 3);
  const current = await store.current();
  assert.equal(current.version, 3);
  assert.equal(current.policy.mode, 'monitor');
  const [rollback] = await store.history(1);
  assert.equal(rollback.action, 'rollback');
  assert.equal(rollback.sourceVersion, 1);
});

class FakeRedis {
  constructor() {
    this.version = 0;
    this.policy = null;
    this.versions = new Map();
    this.audit = [];
  }

  async mGet() {
    return [String(this.version), this.policy];
  }

  async eval(_script, { arguments: values }) {
    const expected = Number(values[0]);
    if (expected !== this.version) return [0, this.version];
    this.version += 1;
    this.policy = values[1];
    this.versions.set(String(this.version), values[2]);
    this.audit.unshift(values[2]);
    this.audit = this.audit.slice(0, 200);
    return [1, this.version];
  }

  async lRange(_key, start, end) {
    return this.audit.slice(start, end + 1);
  }

  async hGet(_key, version) {
    return this.versions.get(version) ?? null;
  }
}
