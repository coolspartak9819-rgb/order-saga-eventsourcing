CREATE TABLE IF NOT EXISTS items (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    title VARCHAR(255) NOT NULL,
    price INT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id)
);

INSERT INTO items (title, price) VALUES ('alpha', 100), ('beta', 200);

CREATE TABLE IF NOT EXISTS outbox (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    aggregate_id BIGINT UNSIGNED NOT NULL,
    event_type VARCHAR(100) NOT NULL,
    payload JSON NOT NULL,
    status ENUM('PENDING', 'PUBLISHED', 'FAILED') NOT NULL DEFAULT 'PENDING',
    attempts INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    published_at TIMESTAMP NULL,
    PRIMARY KEY (id),
    INDEX idx_outbox_pending (status, id)
);

CREATE TABLE IF NOT EXISTS idempotency_keys (
    idempotency_key VARCHAR(255) NOT NULL,
    response_body JSON NOT NULL,
    status_code INT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (idempotency_key)
);
