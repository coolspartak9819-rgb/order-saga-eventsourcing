import { randomUUID } from 'node:crypto';
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
    if (!spec?.module) throw new Error('custom plugin requires module');
    const moduleUrl = pathToFileURL(resolve(baseDirectory, spec.module));
    moduleUrl.searchParams.set('reload', String(Date.now()));
    const imported = await import(moduleUrl.href);
    if (typeof imported.default !== 'function') throw new Error(`plugin ${spec.module} must export a default factory`);
    plugins.push(await imported.default(spec.options ?? {}));
  }
  return plugins;
}

export function compose(middlewares, terminal) {
  return middlewares.reduceRight((next, middleware) => {
    return context => middleware(context, () => next(context));
  }, terminal);
}
