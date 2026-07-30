export default function addHeader(options = {}) {
  return async function addHeaderMiddleware(context, next) {
    context.response.setHeader(options.name || 'x-edgegate-plugin', options.value || 'enabled');
    await next();
  };
}
