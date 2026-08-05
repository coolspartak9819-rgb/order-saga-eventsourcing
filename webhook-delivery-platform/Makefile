PROJECT := webhook-delivery-platform

.PHONY: test run build image compose-up k8s-validate

test:
	GOCACHE=/tmp/$(PROJECT)-go-cache go test -race ./...

run:
	go run ./cmd/api

build:
	CGO_ENABLED=0 go build -trimpath -o bin/webhook-api ./cmd/api

image:
	docker build -t $(PROJECT):dev .

compose-up:
	docker compose up --build

k8s-validate:
	kubectl apply --dry-run=client -f deploy/k8s/
