.PHONY: up down logs smoke validate deploy

up:
	docker compose up --build -d

down:
	docker compose down -v

logs:
	docker compose logs -f --tail=100

smoke:
	./scripts/smoke.sh

purge-cache:
	./scripts/purge-cache.sh

validate:
	docker compose config -q

deploy:
	./deploy/deploy.sh staging

prod-validate:
	APP_VERSION=local docker compose -f docker-compose.yml -f docker-compose.prod.yml config -q
