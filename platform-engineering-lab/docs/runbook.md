# Incident Runbook

## Service is unavailable

```bash
docker compose ps
docker compose logs --tail=100 app proxy
curl -i http://localhost:8082/health
curl -i http://localhost:8082/ready
```

Check whether the application container is healthy first. If the application
is healthy but requests fail, inspect nginx errors and upstream timeouts.

## High error rate

```bash
curl http://localhost:8082/metrics
docker compose logs --since=10m proxy
```

Enable the failure scenario for a local test:

```bash
FAIL_MODE=true docker compose up -d app
curl -i http://localhost:8082/api/content
```

Restore the service with:

```bash
FAIL_MODE=false docker compose up -d app
```

## Cache problems

Inspect `X-Cache-Status` on `/api/content`. Use `./scripts/purge-cache.sh` only
after confirming that stale content is the issue. In production, purge should
be authenticated and executed through the CDN/provider API.

## Rollback

```bash
./deploy/rollback.sh <previous-image-tag>
```

After rollback, run the smoke checks and record the incident timeline and
root cause.
