.PHONY: dev dev-logs proto build test test-int test-e2e test-cover lint \
       run-gateway run-identity run-orchestrator run-telemetry run-control-plane run-sidecar

# Development
dev:
	docker compose -f deployments/docker/docker-compose.yml up --build

dev-logs:
	docker compose -f deployments/docker/docker-compose.yml logs -f

dev-down:
	docker compose -f deployments/docker/docker-compose.yml down -v

# Protobuf
proto:
	cd proto && buf lint && buf generate

# Build
build:
	go build ./services/gateway/cmd/main
	go build ./services/identity/cmd/main
	go build ./services/orchestrator/cmd/main
	go build ./services/telemetry/cmd/main
	go build ./services/control-plane/cmd/main
	go build ./sidecar/cmd/main

# Test
test:
	go test ./...

test-int:
	go test -tags=integration ./...

test-e2e:
	go test -tags=e2e ./...

test-cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Lint
lint:
	golangci-lint run ./...
	cd proto && buf lint
	cd dashboard && npm run lint

# Run individual services
run-gateway:
	go run ./services/gateway/cmd/main

run-identity:
	go run ./services/identity/cmd/main

run-orchestrator:
	go run ./services/orchestrator/cmd/main

run-telemetry:
	go run ./services/telemetry/cmd/main

run-control-plane:
	go run ./services/control-plane/cmd/main

run-sidecar:
	go run ./sidecar/cmd/main
