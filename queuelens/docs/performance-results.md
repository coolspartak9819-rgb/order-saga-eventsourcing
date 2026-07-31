# Performance Result

Local benchmark captured on 2026-07-31 with the following command:

```bash
REQUESTS=5000 CONCURRENCY=100 ./scripts/performance.sh
```

The stack was started with Docker Compose and the benchmark used the public
`POST /api/jobs` endpoint. The benchmark wrapper raised the API rate limit for
the run; the normal application default remains 60 requests per minute.

Result:

```json
{
  "requests": 5000,
  "concurrency": 100,
  "elapsedSeconds": 1.26,
  "requestsPerSecond": 3952.74,
  "latencyMs": {
    "p50": 21.72,
    "p95": 34.14,
    "p99": 96.6,
    "max": 132.64
  },
  "statuses": {
    "202": 5000
  }
}
```

This is a repeatable local reference point, not a production capacity claim.
The result should be compared only when the request mix, concurrency, Docker
limits, dependency versions and host environment are kept comparable.
