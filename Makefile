.PHONY: build test test-integration sqlc-generate docker-up docker-down

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
