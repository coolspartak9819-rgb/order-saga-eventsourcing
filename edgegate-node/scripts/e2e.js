import { execFileSync } from 'node:child_process';

const gatewayUrl = 'http://localhost:8080';

try {
  docker('compose', 'up', '--build', '-d', '--wait');
  await waitFor(`${gatewayUrl}/readyz`, 30_000);

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
  await wait(6500);
  for (let index = 0; index < 10; index += 1) {
    const response = await fetchJSON(`${gatewayUrl}/api/failover-${index}`);
    assert(response.name === 'backend-b', `request reached unhealthy backend: ${response.name}`);
  }

  const metrics = await fetch(`${gatewayUrl}/metrics`).then(response => response.text());
  assert(metrics.includes('edgegate_backend_healthy'), 'backend health metric is missing');
  console.log('EdgeGate e2e failover check passed');
} finally {
  docker('compose', 'down', '--remove-orphans');
}

function docker(...args) {
  execFileSync('docker', args, { stdio: 'inherit' });
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

function assert(condition, message) {
  if (!condition) throw new Error(message);
}

function wait(milliseconds) {
  return new Promise(resolve => setTimeout(resolve, milliseconds));
}
