#!/bin/sh
set -eu

docker compose exec -T proxy sh -c 'rm -rf /var/cache/nginx/*'
echo "nginx cache cleared"
