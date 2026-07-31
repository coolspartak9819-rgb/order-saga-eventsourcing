const target = process.env.TARGET_URL || 'http://localhost:8083/api/jobs';
const total = positiveInteger(process.env.REQUESTS, 1000);
const concurrency = positiveInteger(process.env.CONCURRENCY, 50);
const latencies = [];
const statuses = new Map();
let nextRequest = 0;

const started = performance.now();
await Promise.all(Array.from({ length: concurrency }, async () => {
  while (true) {
    const requestNumber = nextRequest++;
    if (requestNumber >= total) return;

    const requestStarted = performance.now();
    try {
      const response = await fetch(target, {
        method: 'POST',
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify({
          type: 'benchmark.job',
          payload: { request: requestNumber }
        })
      });
      await response.arrayBuffer();
      statuses.set(String(response.status), (statuses.get(String(response.status)) ?? 0) + 1);
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
  elapsedSeconds: round(elapsedSeconds),
  requestsPerSecond: round(total / elapsedSeconds),
  latencyMs: {
    p50: percentile(50),
    p95: percentile(95),
    p99: percentile(99),
    max: round(latencies.at(-1) ?? 0)
  },
  statuses: Object.fromEntries(statuses)
}, null, 2));

function percentile(value) {
  const index = Math.min(latencies.length - 1, Math.ceil((value / 100) * latencies.length) - 1);
  return round(latencies[index] ?? 0);
}

function positiveInteger(value, fallback) {
  const parsed = Number.parseInt(value ?? '', 10);
  return parsed > 0 ? parsed : fallback;
}

function round(value) {
  return Math.round(value * 100) / 100;
}
