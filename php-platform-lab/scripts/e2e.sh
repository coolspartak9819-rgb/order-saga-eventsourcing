#!/bin/sh
set -eu

base_url=${BASE_URL:-http://localhost:8081}

for attempt in $(seq 1 30); do
  if curl -fsS "$base_url/ready" >/dev/null; then
    break
  fi
  sleep 1
done

curl -fsS "$base_url/health"
curl -fsS "$base_url/ready"
curl -fsS "$base_url/items"
first_response=$(curl -fsS -X POST "$base_url/items" \
  -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: e2e-1' \
  -d '{"title":"e2e-item","price":42}'
second_response=$(curl -fsS -X POST "$base_url/items" \
  -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: e2e-1' \
  -d '{"title":"e2e-item","price":42}')
test "$first_response" = "$second_response"
curl -fsS "$base_url/metrics"

echo "e2e checks passed"
