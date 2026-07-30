# QueueLens Architecture

```text
HTTP API
   |
   | PostgreSQL transaction
   v
jobs + outbox
   |
   v
Outbox Dispatcher -- retry/backoff --> Redis Streams
                                      |
                                      v
                              Worker Consumer Group
                                |             |
                                v             v
                           PostgreSQL      Kafka events
```

The API never depends on a successful Redis write to commit a job. The
dispatcher is responsible for delivery and can reclaim a record left in
`PROCESSING` after a crash. The queue and Kafka integrations use at-least-once
delivery, so consumers should use the job ID as an idempotency key.

Workers use a Redis consumer group. New messages are read with `XREADGROUP`;
messages left pending by a crashed worker are recovered with `XAUTOCLAIM` after
30 seconds of inactivity. This prevents a single worker failure from silently
stopping a job.
