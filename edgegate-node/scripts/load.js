const target = process.env.TARGET_URL || 'http://localhost:8080/api/hello';
const total = positiveInteger(process.env.REQUESTS, 1000);
const concurrency = positiveInteger(process.env.CONCURRENCY, 25);
const latencies = [];
const statuses = new Map();
let next = 0;

const started = performance.now();
await Promise.all(Array.from({ length: concurrency }, async () => {
  while (true) {
    const current = next++;
    if (current >= total) return;
    const requestStarted = performance.now();
    try {
      const response = await fetch(target);
      await response.arrayBuffer();
      statuses.set(response.status, (statuses.get(response.status) ?? 0) + 1);
    } catch {
      statuses.set('error', (statuses.get('error') ?? 0) + 1);
    }
    latencies.push(performance.now() - requestStarted);
  }
}));

const elapsedSeconds = (performance.now() - started) / 1000;
latencies.sort((a, b) => a - b);
console.log(JSON.stringify({
  target,
  requests: total,
  concurrency,
  requestsPerSecond: Math.round((total / elapsedSeconds) * 100) / 100,
  latencyMs: { p50: percentile(50), p95: percentile(95), p99: percentile(99) },
  statuses: Object.fromEntries(statuses)
}, null, 2));

function percentile(value) {
  const index = Math.min(latencies.length - 1, Math.ceil((value / 100) * latencies.length) - 1);
  return Math.round((latencies[index] ?? 0) * 100) / 100;
}

function positiveInteger(value, fallback) {
  const parsed = Number.parseInt(value ?? '', 10);
  return parsed > 0 ? parsed : fallback;
}
