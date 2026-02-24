.PHONY: build test test-integration sqlc-generate docker-up docker-down verify-e2e sync-schema

build:
	go build -o bin/server ./cmd/server
	go build -o bin/worker ./cmd/worker

test:
	go test ./...

test-integration:
	go test -v ./internal/integration/...

sqlc-generate:
	sqlc generate

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

verify-e2e:
	./scripts/verify-e2e.sh $(API_URL) $(JWT_TOKEN)

sync-schema:
	./scripts/sync-schema.sh
	./scripts/verify-e2e.sh $(API_URL) $(JWT_TOKEN)
