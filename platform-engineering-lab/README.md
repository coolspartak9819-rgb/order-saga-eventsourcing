# Platform Engineering Lab

An infrastructure-focused PHP service used to practice the operational work
expected from a DevOps / Platform Engineer.

## What is included

- Debian-like Linux deployment notes and an Ansible playbook;
- Docker Compose for local and production-like environments;
- nginx reverse proxy with HTTP caching and cache invalidation;
- MySQL and Redis dependencies;
- Prometheus metrics and Grafana provisioning;
- GitLab CI stages for validation, build, smoke test and manual deploy;
- deploy and rollback scripts;
- Kubernetes manifests for the same service;
- runbooks for incidents, logs, TLS and CDN configuration.

## Run locally

```bash
docker compose up --build -d
./scripts/smoke.sh
```

Services:

- API: `http://localhost:8082`
- Prometheus: `http://localhost:9090`
- Grafana: `http://localhost:3000`

## Main scenarios

```bash
curl http://localhost:8082/health
curl http://localhost:8082/api/content
curl http://localhost:8082/metrics
curl -X POST http://localhost:8082/cache/purge
```

The project is intentionally small in business scope. The focus is on how the
service is built, deployed, monitored, cached, diagnosed and rolled back.

See `docs/runbook.md` for incident procedures and `docs/cdn.md` for CDN and
cache strategy notes.
