# Webhook Delivery Platform

Production-oriented Go service for reliable, multi-tenant webhook delivery.

The project is intentionally built around operational evidence rather than a long technology list:

- tenant-scoped idempotency for incoming events;
- delivery state machine: `pending`, `retrying`, `delivered`, `dead_letter`;
- bounded retries with exponential backoff;
- dead-letter capture after five failed attempts;
- health endpoint and structured JSON logging;
- Prometheus-compatible counters at `/metrics`;
- Docker, Kubernetes, Terraform and load-test entry points.

## Status

The first vertical slice is implemented in Go with an in-memory adapter so the API and delivery lifecycle can be tested without external services. PostgreSQL, NATS/Kafka and Redis adapters are the next implementation layer, not claims of completed production integrations.

## Run

```bash
go test ./...
go run ./cmd/api
```

Create an event:

```bash
curl -X POST http://localhost:8080/v1/events \
  -H 'Content-Type: application/json' \
  -H 'X-Tenant-ID: demo' \
  -d '{"event_id":"evt_001","endpoint_id":"billing","endpoint_url":"http://localhost:9000/webhook","type":"invoice.created","payload":{"amount":1200}}'
```

## Production hardening roadmap

1. PostgreSQL outbox and delivery attempts table.
2. NATS JetStream durable consumer with explicit ack and redelivery.
3. Redis token-bucket limits per tenant and endpoint.
4. OpenTelemetry traces and Prometheus histograms for delivery latency.
5. Kubernetes HPA, PodDisruptionBudget, readiness probes and Terraform modules.
6. k6 load test with a reported p95 latency, throughput and error rate.

Only measurements from the finished load test should be added to the resume.
