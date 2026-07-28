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
  -d '{"title":"e2e-item","price":42}')
second_response=$(curl -fsS -X POST "$base_url/items" \
  -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: e2e-1' \
  -d '{"title":"e2e-item","price":42}')
test "$first_response" = "$second_response"
curl -fsS "$base_url/metrics"

docker compose stop redis >/dev/null
if curl -fsS "$base_url/ready" >/dev/null 2>&1; then
  echo "readiness check did not detect Redis failure" >&2
  exit 1
fi
docker compose start redis >/dev/null

for attempt in $(seq 1 30); do
  if curl -fsS "$base_url/ready" >/dev/null; then
    break
  fi
  sleep 1
done

echo "e2e checks passed"
