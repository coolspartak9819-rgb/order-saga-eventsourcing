#!/bin/sh
set -eu

base_url=${BASE_URL:-http://localhost:8083}
count=${COUNT:-100}

seq "$count" | xargs -P 10 -I{} curl -fsS -X POST "$base_url/api/jobs" \
  -H 'Content-Type: application/json' \
  -d '{"type":"image.process","payload":{"file":"load-{}"}}' >/dev/null

echo "queued $count jobs"
