import { createHmac, randomUUID, timingSafeEqual } from 'node:crypto';
import { pathToFileURL } from 'node:url';
import { resolve } from 'node:path';

const builtins = {
  'request-id': () => async (context, next) => {
    const requestId = context.request.headers['x-request-id'] || randomUUID();
    context.request.headers['x-request-id'] = requestId;
    context.response.setHeader('x-request-id', requestId);
    await next();
  },
  'access-log': () => async (context, next) => {
    const started = performance.now();
    await next();
    console.log(JSON.stringify({
      level: 'info',
      method: context.request.method,
      path: context.request.url,
      backend: context.backend?.url.href,
      durationMs: Math.round((performance.now() - started) * 100) / 100
    }));
  },
  'api-key': options => {
    const keys = readSecretList(options.keysEnv || 'EDGEGATE_API_KEYS');
    return async (context, next) => {
      const candidate = context.request.headers[options.header || 'x-api-key'];
      if (!candidate || !keys.some(key => safeEqual(candidate, key))) return deny(context.response, 401, 'invalid API key');
      await next();
    };
  },
  'jwt-hs256': options => {
    const secret = process.env[options.secretEnv || 'EDGEGATE_JWT_SECRET'];
    if (!secret) throw new Error(`missing JWT secret environment variable: ${options.secretEnv || 'EDGEGATE_JWT_SECRET'}`);
    return async (context, next) => {
      let claims;
      try {
        const authorization = context.request.headers.authorization || '';
        if (!authorization.startsWith('Bearer ')) throw new Error('missing bearer token');
        claims = verifyJWT(authorization.slice(7), secret, options);
      } catch {
        return deny(context.response, 401, 'invalid bearer token');
      }
      context.auth = claims;
      await next();
    };
  }
};

export async function resolvePlugins(specs = [], baseDirectory = process.cwd()) {
  const plugins = [];
  for (const spec of specs) {
    if (typeof spec === 'string') {
      const factory = builtins[spec];
      if (!factory) throw new Error(`unknown plugin: ${spec}`);
      plugins.push(factory({}));
      continue;
    }
    if (spec?.name) {
      const factory = builtins[spec.name];
      if (!factory) throw new Error(`unknown plugin: ${spec.name}`);
      plugins.push(await factory(spec.options ?? {}));
      continue;
    }
    if (!spec?.module) throw new Error('custom plugin requires module');
    const moduleUrl = pathToFileURL(resolve(baseDirectory, spec.module));
    moduleUrl.searchParams.set('reload', String(Date.now()));
    const imported = await import(moduleUrl.href);
    if (typeof imported.default !== 'function') throw new Error(`plugin ${spec.module} must export a default factory`);
    plugins.push(await imported.default(spec.options ?? {}));
  }
  return plugins;
}

function readSecretList(environmentName) {
  const values = (process.env[environmentName] || '').split(',').map(value => value.trim()).filter(Boolean);
  if (values.length === 0) throw new Error(`missing API keys environment variable: ${environmentName}`);
  return values;
}

function safeEqual(left, right) {
  const leftBuffer = Buffer.from(String(left));
  const rightBuffer = Buffer.from(String(right));
  return leftBuffer.length === rightBuffer.length && timingSafeEqual(leftBuffer, rightBuffer);
}

function verifyJWT(token, secret, options) {
  const parts = token.split('.');
  if (parts.length !== 3) throw new Error('invalid token format');
  const [encodedHeader, encodedPayload, signature] = parts;
  const header = JSON.parse(Buffer.from(encodedHeader, 'base64url'));
  const payload = JSON.parse(Buffer.from(encodedPayload, 'base64url'));
  if (header.alg !== 'HS256') throw new Error('unsupported JWT algorithm');
  const expected = createHmac('sha256', secret).update(`${encodedHeader}.${encodedPayload}`).digest('base64url');
  if (!safeEqual(signature, expected)) throw new Error('invalid JWT signature');
  const now = Math.floor(Date.now() / 1000);
  if (payload.exp !== undefined && payload.exp <= now) throw new Error('expired JWT');
  if (payload.nbf !== undefined && payload.nbf > now) throw new Error('JWT is not active');
  if (options.issuer && payload.iss !== options.issuer) throw new Error('invalid issuer');
  if (options.audience && !matchesAudience(payload.aud, options.audience)) throw new Error('invalid audience');
  for (const [name, expectedValue] of Object.entries(options.requiredClaims ?? {})) {
    if (!matchesClaim(payload[name], expectedValue)) throw new Error(`invalid claim: ${name}`);
  }
  return payload;
}

function matchesAudience(actual, expected) {
  return Array.isArray(actual) ? actual.includes(expected) : actual === expected;
}

function matchesClaim(actual, expected) {
  if (Array.isArray(expected)) return Array.isArray(actual) && expected.every(value => actual.includes(value));
  return actual === expected;
}

function deny(response, statusCode, message) {
  response.writeHead(statusCode, { 'content-type': 'text/plain; charset=utf-8', 'www-authenticate': 'Bearer' });
  response.end(`${message}\n`);
}

export function compose(middlewares, terminal) {
  return middlewares.reduceRight((next, middleware) => {
    return context => middleware(context, () => next(context));
  }, terminal);
}
