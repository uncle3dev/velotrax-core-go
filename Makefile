.PHONY: help run build proto lint test docker-build docker-run docker-down

help:
	@echo "Available targets:"
	@echo "  make run          - Run server in development mode"
	@echo "  make build        - Build binary"
	@echo "  make proto        - Regenerate protobuf code"
	@echo "  make lint         - Run golangci-lint"
	@echo "  make test         - Run tests"
	@echo "  make docker-build - Build Docker image"
	@echo "  make docker-run   - Run container with docker-compose"
	@echo "  make docker-down  - Stop container"

run:
	go run ./cmd/server

build:
	go build -o ./bin/server ./cmd/server

proto:
	@bash ./scripts/gen_proto.sh

lint:
	golangci-lint run ./...

test:
	go test -v ./...

docker-build:
	docker build -t velotrax-gateway:latest .

docker-run:
	docker compose up -d

docker-down:
	docker compose down
