# Performance Check

QueueLens includes a repeatable HTTP benchmark that uses only the Node.js
runtime. It creates jobs through the public API and reports throughput, status
distribution and latency percentiles.

Start the local stack first:

```bash
docker compose up --build -d
```

Run a small check:

```bash
REQUESTS=1000 CONCURRENCY=50 node scripts/benchmark.js
```

To build the stack, wait for the API and clean it up automatically after the
run, use:

```bash
REQUESTS=1000 CONCURRENCY=50 ./scripts/performance.sh
```

The performance wrapper raises the API rate limit for the benchmark only. The
normal Compose default remains 60 requests per minute. Set
`RATE_LIMIT_REQUESTS` explicitly when comparing different limits.

Set `RESULT_FILE` to keep the JSON result in a chosen location:

```bash
RESULT_FILE=docs/performance-result.json REQUESTS=5000 CONCURRENCY=100 ./scripts/performance.sh
```

The output includes:

• total request count and concurrency;
• requests per second;
• p50, p95, p99 and maximum latency;
• HTTP status distribution and transport errors.

For a repeatable comparison, keep the following fixed:

• Docker image and dependency versions;
• request count and concurrency;
• host CPU and memory limits;
• PostgreSQL, Redis and Kafka configuration.

The API also exports a Prometheus latency histogram. Query p95 during the
benchmark with:

```promql
histogram_quantile(0.95, rate(queuelens_http_request_duration_seconds_bucket[5m]))
```

Do not describe the service as high load based on one local run. Record the
environment and the benchmark output in a dated report before comparing
changes or making performance claims.
