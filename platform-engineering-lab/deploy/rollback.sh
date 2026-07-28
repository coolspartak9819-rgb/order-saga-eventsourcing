#!/bin/sh
set -eu

version=${1:?usage: rollback.sh IMAGE_TAG}
export APP_VERSION="$version"
docker compose up -d --no-build app proxy
./scripts/smoke.sh
