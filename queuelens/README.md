# QueueLens

QueueLens is a small self-hosted queue operations dashboard.

It demonstrates how queued work moves through a system and what happens when
processing fails: pending jobs, running workers, retries and dead-letter
messages are visible from one place.

## Current scope

- Go HTTP API;
- Redis Streams queue and consumer group;
- PostgreSQL job history;
- PostgreSQL transactional outbox and a dedicated dispatcher;
- worker process with graceful shutdown;
- retry policy and dead-letter stream;
- manual retry for failed jobs;
- stats endpoint and responsive dashboard;
- Docker Compose for API, worker, Redis and PostgreSQL.
- Kafka lifecycle events for completed, retried and failed jobs;
- Prometheus-style API metrics and a repeatable load script.
- Redis stream length and consumer-group pending metrics in the dashboard;
- `X-Request-ID` propagation from HTTP responses into API logs;
- Prometheus scrape configuration.
- Redis pending-message recovery with `XAUTOCLAIM`;
- graceful shutdown for the HTTP API;
- support for running multiple worker replicas.
- optional API key protection for write and retry operations;
- PostgreSQL-backed idempotency for job creation;
- per-client rate limiting;
- middleware unit tests and a real-stack integration script.
- Kafka audit consumer with a PostgreSQL job event timeline;
- Grafana dashboard provisioned from the repository;
- retention worker for removing old jobs and their audit history.
- Kubernetes manifests with liveness/readiness probes and resource limits;
- GitHub Actions checks for tests, vet and Compose validation.
- GitHub Actions builds and publishes one GHCR image per service on `main`.

## Delivery guarantee

The API writes a job and its outbox record in one PostgreSQL transaction. The
dispatcher claims pending outbox records and publishes them to Redis Streams.
Failed deliveries use exponential backoff. A record that was published before
the dispatcher crashed can be delivered again, so workers must remain
idempotent. This is an intentional at-least-once delivery model.

## Run

```bash
docker compose up --build
```

Open http://localhost:8083.

Kubernetes deployment manifests are available in `deploy/k8s`.

Prometheus is available at http://localhost:9090.

Grafana is available at http://localhost:3000 (`admin` / `admin`) with a
provisioned QueueLens dashboard.

Create a normal job in the dashboard, or create a deterministic failure:

```bash
curl -X POST http://localhost:8083/api/jobs \
  -H 'Content-Type: application/json' \
  -d '{"type":"image.process","payload":{"fail":true}}'
```

After three attempts the job is marked `FAILED` and copied to the Redis
`jobs.dlq` stream. A failed job can be requeued from the dashboard.

Generate a small load sample:

```bash
chmod +x scripts/load.sh
COUNT=100 ./scripts/load.sh
```

Metrics are available at `http://localhost:8083/metrics`. Kafka lifecycle
events are published to the `job-events` topic.

Run the integration check against the full stack:

```bash
chmod +x scripts/integration.sh
./scripts/integration.sh
```

The integration check also submits the same create request twice with one
`Idempotency-Key` and verifies that both responses contain the same job ID.

Set `API_KEY` in the environment to protect `POST /api/jobs` and manual retry
requests with the `X-API-Key` header.

The retention worker removes jobs older than `RETENTION_DAYS` (30 by default).
Because `job_events`, outbox records and idempotency records use cascading
foreign keys, the cleanup keeps operational and audit data consistent:

```bash
RETENTION_DAYS=14 docker compose up --build
```

Use `Idempotency-Key` when creating a job. Repeating the same request with the
same key returns the existing job instead of creating a duplicate:

```bash
curl -X POST http://localhost:8083/api/jobs \
  -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: import-2026-01-01' \
  -d '{"type":"image.process","payload":{"file":"demo.jpg"}}'
```

Run several workers locally to exercise the consumer group:

```bash
docker compose up --build --scale worker=3
```

If a worker dies while holding a message, another worker can reclaim it after
the pending entry has been idle for 30 seconds.

Click any job in the dashboard to inspect its Kafka-backed event timeline,
including attempts, errors and event payloads.
