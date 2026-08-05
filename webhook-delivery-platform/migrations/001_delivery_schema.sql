CREATE TABLE webhook_events (
    tenant_id       text        NOT NULL,
    event_id        text        NOT NULL,
    event_type      text        NOT NULL,
    payload         jsonb       NOT NULL,
    created_at      timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, event_id)
);

CREATE TABLE webhook_deliveries (
    delivery_id     uuid        PRIMARY KEY,
    tenant_id       text        NOT NULL,
    event_id        text        NOT NULL,
    endpoint_id     text        NOT NULL,
    endpoint_url    text        NOT NULL,
    status          text        NOT NULL CHECK (status IN ('pending', 'retrying', 'delivered', 'dead_letter')),
    attempts        integer     NOT NULL DEFAULT 0,
    next_attempt_at timestamptz NOT NULL,
    last_error      text,
    created_at      timestamptz NOT NULL DEFAULT now(),
    delivered_at    timestamptz,
    UNIQUE (tenant_id, event_id, endpoint_id),
    FOREIGN KEY (tenant_id, event_id) REFERENCES webhook_events (tenant_id, event_id)
);

CREATE INDEX webhook_deliveries_ready_idx
    ON webhook_deliveries (next_attempt_at)
    WHERE status IN ('pending', 'retrying');

CREATE TABLE webhook_delivery_attempts (
    id             bigserial    PRIMARY KEY,
    delivery_id    uuid         NOT NULL REFERENCES webhook_deliveries (delivery_id),
    attempt        integer      NOT NULL,
    status_code    integer,
    duration_ms    integer,
    error          text,
    created_at     timestamptz  NOT NULL DEFAULT now()
);
