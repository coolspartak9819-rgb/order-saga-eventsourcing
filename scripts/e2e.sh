#!/bin/sh

set -eu

BASE_URL=${BASE_URL:-http://localhost:8080}

create_order() {
  key=$1
  curl --fail --silent --show-error \
    -X POST "$BASE_URL/api/orders" \
    -H 'Content-Type: application/json' \
    -H "X-Idempotency-Key: $key" \
    -d '{"customer_id":"e2e-customer","items":[{"product_id":"e2e-product","quantity":2,"price":19.99}]}'
}

echo "Checking readiness"
test "$(curl --fail --silent "$BASE_URL/readyz")" = '{"status":"ready"}'

echo "Checking successful order and idempotency"
response=$(create_order "e2e-success-1")
order_id=$(printf '%s' "$response" | sed -n 's/.*"order_id":"\([^"]*\)".*/\1/p')
test -n "$order_id"
test "$(create_order "e2e-success-1")" = "$response"
order=$(curl --fail --silent "$BASE_URL/api/orders/$order_id")
printf '%s\n' "$order" | grep '"status":"COMPLETED"' >/dev/null
printf '%s\n' "$order" | grep '"total_amount":39.98' >/dev/null

echo "Checking metrics"
curl --fail --silent "$BASE_URL/metrics" | grep 'order_service_http_requests_total' >/dev/null

if [ "${RUN_FAILURE_SCENARIOS:-false}" = "true" ]; then
  echo "Checking payment failure path"
  PAYMENT_FAIL=true docker compose up -d --force-recreate payment-service >/dev/null
  payment_status=$(curl --silent --output /tmp/order-saga-payment-failure.json --write-out '%{http_code}' \
    -X POST "$BASE_URL/api/orders" \
    -H 'Content-Type: application/json' \
    -H 'X-Idempotency-Key: e2e-payment-failure-1' \
    -d '{"customer_id":"e2e-customer","items":[{"product_id":"e2e-product","quantity":1,"price":10}]}' )
  test "$payment_status" = "500"
  docker compose exec -T db psql -U postgres -d orders -Atc \
    "SELECT COUNT(*) FROM events WHERE event_type = 'payment.failed'" | grep -v '^0$' >/dev/null
  docker compose up -d --force-recreate payment-service >/dev/null

  echo "Checking inventory failure and refund path"
  INVENTORY_FAIL=true docker compose up -d --force-recreate inventory-service >/dev/null
  inventory_status=$(curl --silent --output /tmp/order-saga-inventory-failure.json --write-out '%{http_code}' \
    -X POST "$BASE_URL/api/orders" \
    -H 'Content-Type: application/json' \
    -H 'X-Idempotency-Key: e2e-inventory-failure-1' \
    -d '{"customer_id":"e2e-customer","items":[{"product_id":"e2e-product","quantity":1,"price":10}]}' )
  test "$inventory_status" = "500"
  docker compose exec -T db psql -U postgres -d orders -Atc \
    "SELECT COUNT(*) FROM events WHERE event_type = 'inventory.failed'" | grep -v '^0$' >/dev/null
  docker compose exec -T db psql -U postgres -d orders -Atc \
    "SELECT COUNT(*) FROM events WHERE event_type = 'order.cancelled'" | grep -v '^0$' >/dev/null
  docker compose up -d --force-recreate inventory-service >/dev/null
fi

echo "E2E checks passed"
