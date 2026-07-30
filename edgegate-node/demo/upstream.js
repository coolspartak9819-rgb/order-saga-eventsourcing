import http from 'node:http';

const name = process.env.INSTANCE_NAME || 'backend';
const port = Number(process.env.PORT || 8080);
let healthy = true;
let failRequests = false;

const server = http.createServer((request, response) => {
  const url = new URL(request.url, `http://${request.headers.host}`);
  if (url.pathname === '/healthz') {
    response.writeHead(healthy ? 200 : 503, { 'content-type': 'application/json' });
    return response.end(JSON.stringify({ name, healthy }));
  }
  if (url.pathname === '/admin/state' && request.method === 'POST') {
    healthy = url.searchParams.get('healthy') !== 'false';
    failRequests = url.searchParams.get('fail') === 'true';
    response.writeHead(200, { 'content-type': 'application/json' });
    return response.end(JSON.stringify({ name, healthy, failRequests }));
  }
  if (failRequests) {
    response.writeHead(503, { 'content-type': 'application/json' });
    return response.end(JSON.stringify({ name, error: 'injected failure' }));
  }
  response.writeHead(200, { 'content-type': 'application/json', 'x-upstream': name });
  response.end(JSON.stringify({ name, path: url.pathname, requestId: request.headers['x-request-id'] }));
});

server.listen(port, '0.0.0.0', () => console.log(`${name} listening on ${port}`));

const shutdown = () => server.close(() => process.exit(0));
process.on('SIGINT', shutdown);
process.on('SIGTERM', shutdown);
