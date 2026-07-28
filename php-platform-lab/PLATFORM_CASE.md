# PHP Platform Lab

Small platform engineering project around a PHP backend.

## What it demonstrates

- PHP-FPM and nginx running as separate containers;
- MySQL and Redis dependencies;
- Docker Compose environment with health checks;
- liveness and readiness endpoints;
- Prometheus-style metrics endpoint;
- reproducible end-to-end checks;
- GitHub Actions validation;
- dependency failure handling;
- transactional outbox with a dedicated Kafka worker;
- idempotent write requests.
- retry and terminal failure handling for outbox messages;
- graceful worker shutdown;
- an e2e dependency failure check for Redis.

## Run locally

```bash
docker compose up --build -d
./scripts/e2e.sh
```

The API is available at `http://localhost:8081`.

Useful endpoints:

```text
GET  /health
GET  /ready
GET  /metrics
GET  /items
POST /items
```

The project focuses on operating a PHP service in a predictable local and CI
environment: container boundaries, dependency readiness, configuration and
basic failure checks.
