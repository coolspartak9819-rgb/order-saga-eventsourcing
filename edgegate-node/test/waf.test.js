import test from 'node:test';
import assert from 'node:assert/strict';
import { WAF, validateWAFPolicy } from '../src/waf.js';

const waf = new WAF();

test('blocks encoded XSS and SQL injection signatures', () => {
  assert.equal(waf.inspect({ url: '/api?q=%3Cscript%3Ealert(1)%3C/script%3E', headers: {} }), 'xss');
  assert.equal(waf.inspect({ url: '/api', headers: {}, body: 'id=1 union select password from users' }), 'sqli');
});

test('allows normal requests', () => {
  assert.equal(waf.inspect({ url: '/api/items?page=2', headers: { accept: 'application/json' } }), null);
});

test('supports monitor policies, custom rules and disabled defaults', () => {
  const monitor = new WAF({
    mode: 'monitor',
    disabledDefaultRules: ['xss'],
    rules: [{ id: 'scanner', source: 'masscan', enabled: true }]
  });
  assert.equal(monitor.mode, 'monitor');
  assert.equal(monitor.inspect({ url: '/?q=<script>', headers: {} }), null);
  assert.equal(monitor.inspect({ url: '/', headers: { 'user-agent': 'masscan/1.3' } }), 'scanner');
});

test('rejects invalid WAF policies before activation', () => {
  assert.throws(() => validateWAFPolicy({ mode: 'disabled' }), /mode/);
  assert.throws(() => validateWAFPolicy({ rules: [{ id: 'bad', source: '[' }] }), /invalid regular expression/);
  assert.throws(() => validateWAFPolicy({ rules: [
    { id: 'duplicate', source: 'one' },
    { id: 'duplicate', source: 'two' }
  ] }), /duplicate/);
});
