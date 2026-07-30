import test from 'node:test';
import assert from 'node:assert/strict';
import { validateConfig } from '../src/config.js';

test('validates a route configuration', () => {
  const config = { routes: [{ path: '/api', backends: ['http://localhost:9000'] }] };
  assert.doesNotThrow(() => validateConfig(config));
  assert.equal(config.routes[0].strategy, 'round_robin');
});

test('rejects duplicate paths and invalid strategies', () => {
  assert.throws(() => validateConfig({ routes: [
    { path: '/api', backends: ['http://localhost:9000'] },
    { path: '/api', backends: ['http://localhost:9001'] }
  ] }), /duplicate route/);
  assert.throws(() => validateConfig({ routes: [
    { path: '/api', backends: ['http://localhost:9000'], strategy: 'random' }
  ] }), /unsupported strategy/);
});
