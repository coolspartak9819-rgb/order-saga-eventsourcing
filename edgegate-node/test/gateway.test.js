import test from 'node:test';
import assert from 'node:assert/strict';
import { matchesPrefix } from '../src/gateway.js';

test('route prefix respects path segment boundaries', () => {
  assert.equal(matchesPrefix('/api', '/api'), true);
  assert.equal(matchesPrefix('/api/orders', '/api'), true);
  assert.equal(matchesPrefix('/apix', '/api'), false);
  assert.equal(matchesPrefix('/anything', '/'), true);
});
