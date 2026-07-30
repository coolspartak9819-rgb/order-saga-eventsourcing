import test from 'node:test';
import assert from 'node:assert/strict';
import { createHmac } from 'node:crypto';
import { resolvePlugins } from '../src/plugins.js';

test('API key plugin accepts configured keys and rejects invalid keys', async () => {
  process.env.EDGEGATE_TEST_KEYS = 'alpha,beta';
  const [plugin] = await resolvePlugins([{ name: 'api-key', options: { keysEnv: 'EDGEGATE_TEST_KEYS' } }]);
  let called = false;
  await plugin(context({ 'x-api-key': 'beta' }), async () => { called = true; });
  assert.equal(called, true);

  const rejected = context({ 'x-api-key': 'wrong' });
  await plugin(rejected, async () => assert.fail('next must not be called'));
  assert.equal(rejected.response.statusCode, 401);
});

test('JWT plugin validates signature, expiry and required claims', async () => {
  process.env.EDGEGATE_TEST_JWT = 'test-secret';
  const [plugin] = await resolvePlugins([{
    name: 'jwt-hs256',
    options: { secretEnv: 'EDGEGATE_TEST_JWT', issuer: 'edgegate', requiredClaims: { roles: ['admin'] } }
  }]);
  const token = signJWT({ sub: 'user-1', iss: 'edgegate', roles: ['admin', 'operator'], exp: Math.floor(Date.now() / 1000) + 60 }, 'test-secret');
  const accepted = context({ authorization: `Bearer ${token}` });
  await plugin(accepted, async () => { accepted.nextCalled = true; });
  assert.equal(accepted.nextCalled, true);
  assert.equal(accepted.auth.sub, 'user-1');

  const expired = signJWT({ iss: 'edgegate', roles: ['admin'], exp: 1 }, 'test-secret');
  const rejected = context({ authorization: `Bearer ${expired}` });
  await plugin(rejected, async () => assert.fail('next must not be called'));
  assert.equal(rejected.response.statusCode, 401);
});

function signJWT(payload, secret) {
  const header = Buffer.from(JSON.stringify({ alg: 'HS256', typ: 'JWT' })).toString('base64url');
  const body = Buffer.from(JSON.stringify(payload)).toString('base64url');
  const signature = createHmac('sha256', secret).update(`${header}.${body}`).digest('base64url');
  return `${header}.${body}.${signature}`;
}

function context(headers) {
  const response = {
    statusCode: 200,
    headers: {},
    setHeader(name, value) { this.headers[name] = value; },
    writeHead(statusCode, nextHeaders) { this.statusCode = statusCode; Object.assign(this.headers, nextHeaders); },
    end(body) { this.body = body; }
  };
  return { request: { headers }, response };
}
