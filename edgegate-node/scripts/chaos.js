import { execFileSync } from 'node:child_process';

const gatewayUrl = process.env.GATEWAY_URL || 'http://localhost:8080';
const compose = ['compose'];

try {
  docker('up', '--build', '-d');
  await waitFor(`${gatewayUrl}/readyz`, 60_000);
  await expectStatus(`${gatewayUrl}/api/chaos-baseline`, 200, 10_000);

  docker('stop', 'redis');
  await expectStatus(`${gatewayUrl}/api/chaos-redis-down`, 503, 15_000);

  docker('start', 'redis');
  await waitFor(`${gatewayUrl}/readyz`, 30_000);
  await expectStatus(`${gatewayUrl}/api/chaos-recovered`, 200, 15_000);

  console.log('EdgeGate chaos check passed: Redis outage and recovery handled');
} catch (error) {
  console.error(`EdgeGate chaos check failed: ${error.stack || error.message}`);
  process.exitCode = 1;
} finally {
  safeDocker('down', '--remove-orphans');
}

function docker(...args) {
  execFileSync('docker', [...compose, ...args], { stdio: 'inherit' });
}

function safeDocker(...args) {
  try {
    docker(...args);
  } catch {}
}

async function expectStatus(url, expectedStatus, timeoutMs) {
  const deadline = Date.now() + timeoutMs;
  let lastStatus = 'error';
  while (Date.now() < deadline) {
    try {
      const response = await fetch(url);
      lastStatus = response.status;
      if (response.status === expectedStatus) return;
    } catch {
      lastStatus = 'error';
    }
    await wait(250);
  }
  throw new Error(`${url} returned ${lastStatus}, expected ${expectedStatus}`);
}

async function waitFor(url, timeoutMs) {
  await expectStatus(url, 200, timeoutMs);
}

function wait(milliseconds) {
  return new Promise(resolve => setTimeout(resolve, milliseconds));
}
