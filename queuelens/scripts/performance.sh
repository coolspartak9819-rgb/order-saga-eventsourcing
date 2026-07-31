#!/bin/sh
set -eu

base_url=${BASE_URL:-http://localhost:8083}
requests=${REQUESTS:-1000}
concurrency=${CONCURRENCY:-50}
rate_limit_requests=${RATE_LIMIT_REQUESTS:-100000}
result_file=${RESULT_FILE:-"/tmp/queuelens-performance-${requests}-${concurrency}.json"}

cleanup() {
  docker compose down --remove-orphans >/dev/null 2>&1 || true
}
trap cleanup EXIT INT TERM

RATE_LIMIT_REQUESTS="$rate_limit_requests" docker compose up --build -d

ready=0
for attempt in $(seq 1 60); do
  if curl -fsS "$base_url/health" >/dev/null; then
    ready=1
    break
  fi
  sleep 1
done

if [ "$ready" -ne 1 ]; then
  echo "QueueLens API did not become ready" >&2
  docker compose ps
  exit 1
fi

TARGET_URL="$base_url/api/jobs" REQUESTS="$requests" CONCURRENCY="$concurrency" \
  node scripts/benchmark.js | tee "$result_file"

echo "Performance result saved to $result_file"
