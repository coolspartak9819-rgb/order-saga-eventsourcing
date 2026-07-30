import test from 'node:test';
import assert from 'node:assert/strict';
import { Gateway, matchesPrefix } from '../src/gateway.js';
import { WAF } from '../src/waf.js';

test('route prefix respects path segment boundaries', () => {
  assert.equal(matchesPrefix('/api', '/api'), true);
  assert.equal(matchesPrefix('/api/orders', '/api'), true);
  assert.equal(matchesPrefix('/apix', '/api'), false);
  assert.equal(matchesPrefix('/anything', '/'), true);
});

test('applies a new WAF policy to active routes without rebuilding them', () => {
  const gateway = new Gateway(null);
  gateway.routes = [{ waf: new WAF(), wafExtraRules: [{ id: 'route-rule', source: 'route-signature' }] }];
  gateway.applyWAFPolicy({ mode: 'monitor', rules: [{ id: 'global-rule', source: 'global-signature' }] });

  assert.equal(gateway.routes[0].waf.mode, 'monitor');
  assert.equal(gateway.routes[0].waf.inspect({ url: '/global-signature', headers: {} }), 'global-rule');
  assert.equal(gateway.routes[0].waf.inspect({ url: '/route-signature', headers: {} }), 'route-rule');
});
