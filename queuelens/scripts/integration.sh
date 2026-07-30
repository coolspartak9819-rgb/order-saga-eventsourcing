#!/bin/sh
set -eu

base_url=${BASE_URL:-http://localhost:8083}
compose="docker compose"

cleanup() { $compose down -v >/dev/null 2>&1 || true; }
trap cleanup EXIT

$compose up --build -d
for attempt in $(seq 1 45); do
  if curl -fsS "$base_url/health" >/dev/null; then break; fi
  sleep 1
done

response=$(curl -fsS -X POST "$base_url/api/jobs" -H 'Content-Type: application/json' -H 'Idempotency-Key: integration-1' -d '{"type":"integration.success","payload":{"value":1}}')
job_id=$(printf '%s' "$response" | sed -n 's/.*"job_id":"\([^"]*\)".*/\1/p')
test -n "$job_id"
repeat=$(curl -fsS -X POST "$base_url/api/jobs" -H 'Content-Type: application/json' -H 'Idempotency-Key: integration-1' -d '{"type":"integration.success","payload":{"value":1}}')
repeat_id=$(printf '%s' "$repeat" | sed -n 's/.*"job_id":"\([^"]*\)".*/\1/p')
test "$repeat_id" = "$job_id"

for attempt in $(seq 1 20); do
  status=$(curl -fsS "$base_url/api/jobs/$job_id" | sed -n 's/.*"status":"\([^"]*\)".*/\1/p')
  [ "$status" = "COMPLETED" ] && break
  sleep 1
done
test "$status" = "COMPLETED"

curl -fsS http://localhost:8083/metrics | grep -q queuelens_http_requests_total
echo "integration checks passed"
