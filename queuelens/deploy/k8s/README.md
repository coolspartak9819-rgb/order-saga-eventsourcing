# Kubernetes deployment

`queue-lens.yaml` describes the application workloads for a cluster where
PostgreSQL, Redis and Kafka are reachable as `postgres`, `redis` and `kafka`.
The API has two probes:

- `/health` is a liveness check and does not depend on external services;
- `/ready` checks PostgreSQL and Redis before receiving traffic.

The GitHub Actions workflow builds and publishes these images to GHCR after a
push to `main`. Replace the secret values before applying:

```bash
kubectl apply -f deploy/k8s/queue-lens.yaml
kubectl -n queuelens rollout status deployment/queuelens-api
kubectl -n queuelens port-forward service/queuelens-api 8083:80
```

The worker deployment uses three replicas and the Redis consumer group for
horizontal processing. Dispatcher, audit and retention remain singletons
because each one coordinates a shared stream or scheduled database work.
