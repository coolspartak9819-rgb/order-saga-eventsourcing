# PHP Platform Lab

A small production-style PHP service built to demonstrate platform engineering
work around a PHP backend: container boundaries, health checks, dependency
readiness, metrics, reproducible local environments and failure testing.

## Stack

- PHP-FPM 8.3
- nginx
- MySQL 8
- Redis 7
- Docker Compose
- GitHub Actions
- Kafka and transactional outbox
- Idempotent write requests

The application exposes a small items API. The interesting part of the
project is the runtime setup and operational checks, not the business logic.

## Run

```bash
docker compose up --build -d
./scripts/e2e.sh
```

Endpoints:

- `GET /health` - process liveness
- `GET /ready` - MySQL and Redis readiness
- `GET /metrics` - request counters and latency
- `GET /items` - read items
- `POST /items` - create an item

Stop the stack:

```bash
docker compose down -v
```

## Operational checks

The stack uses separate nginx and PHP-FPM containers. MySQL and Redis have
health checks, and the API does not report ready until both dependencies can
be reached. The e2e script waits for readiness, exercises the API and checks
that a dependency failure produces a non-ready response. Write requests use
the `Idempotency-Key` header and are stored together with an outbox event in
one MySQL transaction. A separate worker publishes pending events to Kafka.
