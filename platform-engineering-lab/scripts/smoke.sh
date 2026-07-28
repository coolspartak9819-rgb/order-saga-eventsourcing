#!/bin/sh
set -eu

base_url=${BASE_URL:-http://localhost:8082}

for attempt in $(seq 1 30); do
  if curl -fsS "$base_url/health" >/dev/null; then
    break
  fi
  sleep 1
done

curl -fsS "$base_url/health"
curl -fsS "$base_url/api/content"
curl -fsS "$base_url/metrics"

cache_status=$(curl -sS -D - "$base_url/api/content" -o /dev/null | awk '/X-Cache-Status:/ {print $2}' | tr -d '\r')
test -n "$cache_status"

echo "smoke checks passed; cache status: $cache_status"
