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
  await waitForFailover(10, 30_000);

  const metrics = await fetch(`${gatewayUrl}/metrics`).then(response => response.text());
  assert(metrics.includes('edgegate_backend_healthy'), 'backend health metric is missing');
  console.log('EdgeGate e2e failover check passed');
} catch (error) {
  console.error(`EdgeGate e2e failed: ${error.stack || error.message}`);
  console.error(`::error title=EdgeGate e2e failed::${githubEscape(error.message)}`);
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

async function waitForFailover(requiredSuccesses, timeoutMs) {
  const deadline = Date.now() + timeoutMs;
  let consecutive = 0;
  while (Date.now() < deadline) {
    try {
      const response = await fetch(`${gatewayUrl}/api/failover-${consecutive}`);
      if (response.ok && (await response.json()).name === 'backend-b') consecutive += 1;
      else consecutive = 0;
      if (consecutive >= requiredSuccesses) return;
    } catch { consecutive = 0; }
    await wait(500);
  }
  throw new Error(`failover did not produce ${requiredSuccesses} consecutive backend-b responses`);
}

function assert(condition, message) {
  if (!condition) throw new Error(message);
}

function wait(milliseconds) {
  return new Promise(resolve => setTimeout(resolve, milliseconds));
}

function githubEscape(value) {
  return String(value).replaceAll('%', '%25').replaceAll('\r', '%0D').replaceAll('\n', '%0A');
}
