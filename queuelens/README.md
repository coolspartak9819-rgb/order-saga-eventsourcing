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
