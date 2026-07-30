import { execFileSync } from 'node:child_process';

const gatewayUrl = 'http://localhost:8080';

try {
  docker('compose', 'up', '--build', '-d');
  await waitFor(`${gatewayUrl}/readyz`, 60_000);

  const baseline = await fetchJSON(`${gatewayUrl}/api/e2e`);
  assert(['backend-a', 'backend-b'].includes(baseline.name), 'baseline request did not reach an upstream');

  const currentConfig = await fetchJSON(`${gatewayUrl}/control/config`, { headers: { 'x-control-key': 'demo-control-key' } });
  const updateResponse = await fetch(`${gatewayUrl}/control/config`, {
    method: 'PUT',
    headers: { 'content-type': 'application/json', 'x-control-key': 'demo-control-key' },
    body: JSON.stringify({ expectedVersion: currentConfig.version, config: currentConfig.config })
  });
  assert(updateResponse.status === 202, `config update returned ${updateResponse.status}`);
  const updatedConfig = await updateResponse.json();
  assert(updatedConfig.version === currentConfig.version + 1, 'config version did not advance');

  docker('compose', 'stop', 'backend-a');
  await waitForMetric('edgegate_backend_healthy{route="/api",backend="http://backend-a:8080"} 0', 20_000);
  for (let index = 0; index < 10; index += 1) {
    const response = await fetchJSON(`${gatewayUrl}/api/failover-${index}`);
    assert(response.name === 'backend-b', `request reached unhealthy backend: ${response.name}`);
  }

  const metrics = await fetch(`${gatewayUrl}/metrics`).then(response => response.text());
  assert(metrics.includes('edgegate_backend_healthy'), 'backend health metric is missing');
  console.log('EdgeGate e2e failover check passed');
} catch (error) {
  console.error(`EdgeGate e2e failed: ${error.stack || error.message}`);
  safeDocker('compose', 'ps');
  safeDocker('compose', 'logs', '--no-color', 'edgegate', 'backend-a', 'backend-b', 'redis');
  process.exitCode = 1;
} finally {
  safeDocker('compose', 'down', '--remove-orphans');
}

function docker(...args) {
  execFileSync('docker', args, { stdio: 'inherit' });
}

function safeDocker(...args) {
  try { docker(...args); } catch {}
}

async function fetchJSON(url, options) {
  const response = await fetch(url, options);
  assert(response.ok, `${url} returned ${response.status}`);
  return response.json();
}

async function waitFor(url, timeoutMs) {
  const deadline = Date.now() + timeoutMs;
  while (Date.now() < deadline) {
    try {
      const response = await fetch(url);
      if (response.ok) return;
    } catch {}
    await wait(500);
  }
  throw new Error(`timed out waiting for ${url}`);
}

async function waitForMetric(expectedLine, timeoutMs) {
  const deadline = Date.now() + timeoutMs;
  while (Date.now() < deadline) {
    try {
      const metrics = await fetch(`${gatewayUrl}/metrics`).then(response => response.text());
      if (metrics.includes(expectedLine)) return;
    } catch {}
    await wait(500);
  }
  throw new Error(`timed out waiting for metric: ${expectedLine}`);
}

function assert(condition, message) {
  if (!condition) throw new Error(message);
}

function wait(milliseconds) {
  return new Promise(resolve => setTimeout(resolve, milliseconds));
}
