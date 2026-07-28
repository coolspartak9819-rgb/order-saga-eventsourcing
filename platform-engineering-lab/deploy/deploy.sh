#!/bin/sh
set -eu

environment=${1:-staging}
project="platform-${environment}"

case "$environment" in
  staging|production) ;;
  *) echo "usage: $0 staging|production" >&2; exit 2 ;;
esac

export COMPOSE_PROJECT_NAME="$project"
export APP_VERSION="${APP_VERSION:-$(git rev-parse --short HEAD 2>/dev/null || echo unknown)}"

docker compose up -d --build --remove-orphans
./scripts/smoke.sh
