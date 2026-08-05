# Architecture Notes

## Delivery contract

The producer sends an event with `X-Tenant-ID` and an `event_id`. The tuple `(tenant_id, event_id, endpoint_id)` is the idempotency boundary. A repeated request returns the original delivery instead of creating a second delivery.

## State machine

```text
pending -> delivered
pending -> retrying -> delivered
retrying -> dead_letter
```

Every attempt stores the response code, duration and error. Retry delay is exponential and bounded. A dead-letter record is retained for operator replay.

## Production migration path

The current Go adapter keeps the first slice executable without infrastructure. The SQL schema defines the durable version of the same contract. A production worker should claim rows with `FOR UPDATE SKIP LOCKED`, publish the claim to a durable queue, and acknowledge the queue message only after persisting the attempt result.

The important correctness property is that a crash can cause a duplicate delivery, but it cannot silently lose an accepted event. Receiver-side idempotency keys are therefore part of the HTTP contract.

## Operational signals

- `webhook_events_accepted_total`
- `webhook_deliveries_delivered_total`
- `webhook_delivery_retries_total`
- `webhook_dead_letters_total`
- delivery latency histogram in the production adapter

The k6 scenario defines a starting target of 100 events per second and p95 under 250 ms. Those values are test thresholds, not resume claims until the environment is measured.
