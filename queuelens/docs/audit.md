# Event Audit

The worker publishes job lifecycle events to Kafka. The `audit` service reads
the `job-events` topic with its own consumer group and stores the events in
PostgreSQL. This keeps a durable timeline separate from the current job state.

The timeline is available through:

```text
GET /api/jobs/{id}/events
```

Kafka delivery and audit persistence are at-least-once. Kafka partition and
offset are stored with a unique constraint, so a consumer restart does not
create duplicate audit records.
