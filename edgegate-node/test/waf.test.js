import test from 'node:test';
import assert from 'node:assert/strict';
import { WAF } from '../src/waf.js';

const waf = new WAF();

test('blocks encoded XSS and SQL injection signatures', () => {
  assert.equal(waf.inspect({ url: '/api?q=%3Cscript%3Ealert(1)%3C/script%3E', headers: {} }), 'xss');
  assert.equal(waf.inspect({ url: '/api', headers: {}, body: 'id=1 union select password from users' }), 'sqli');
});

test('allows normal requests', () => {
  assert.equal(waf.inspect({ url: '/api/items?page=2', headers: { accept: 'application/json' } }), null);
});
