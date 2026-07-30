# Changelog

## 0.2.0 - 2026-07-30

### Added

- Versioned WAF policies stored in Redis with optimistic concurrency control.
- `block` and `monitor` policy modes for safe rule rollout.
- Redis Pub/Sub propagation of WAF changes across gateway replicas.
- WAF audit history with actor, action, timestamp and source version.
- Rollback that creates a new version instead of rewriting history.
- Control endpoints for policy updates, history, rollback and false-positive feedback.
- Prometheus metrics for WAF detections and reported false positives.
- Docker e2e coverage for WAF rollout, rollback and backend failover.

### Verification

- 19 Node.js tests pass.
- Docker Compose e2e passes locally.
- GitHub Actions test, e2e and container publication jobs pass.
