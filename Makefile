# Argus — Agentic Orchestration, Monitoring & Predictive Failure Platform
# Top-level Makefile

SHELL := /bin/bash

# Go workspace settings
export GOFLAGS := -mod=readonly
GOBUILD := CGO_ENABLED=0 go build -trimpath -ldflags="-s -w"
GOTEST := go test

# Docker
COMPOSE := docker compose -f deployments/docker/docker-compose.yml
DOCKER_REGISTRY ?= argus
DOCKER_TAG ?= latest

# Service list
SERVICES := control-plane orchestrator telemetry identity gateway
SIDECAR := sidecar

# Output directory
BIN_DIR := bin

.PHONY: help dev dev-logs dev-down proto \
        build build-control-plane build-orchestrator build-telemetry build-identity build-gateway build-sidecar \
        test test-int test-e2e test-cover \
        lint fmt \
        run-control-plane run-orchestrator run-telemetry run-identity run-gateway run-sidecar \
        dashboard-dev dashboard-build \
        clean docker-build

# ─── Help ────────────────────────────────────────────────────────────────────

help: ## Show all available targets
	@echo "Argus Makefile targets:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-24s\033[0m %s\n", $$1, $$2}'
	@echo ""

# ─── Development ─────────────────────────────────────────────────────────────

dev: ## Start all services via docker-compose
	$(COMPOSE) up --build

dev-logs: ## Tail all service logs
	$(COMPOSE) logs -f

dev-down: ## Stop all services and remove volumes
	$(COMPOSE) down -v

# ─── Protobuf ────────────────────────────────────────────────────────────────

proto: ## Regenerate protobuf code (buf generate)
	cd proto && buf lint && buf generate

# ─── Build ───────────────────────────────────────────────────────────────────

build: build-control-plane build-orchestrator build-telemetry build-identity build-gateway build-sidecar ## Build all Go services

build-control-plane: ## Build control-plane service
	@mkdir -p $(BIN_DIR)
	$(GOBUILD) -o $(BIN_DIR)/control-plane ./services/control-plane/cmd/main

build-orchestrator: ## Build orchestrator service
	@mkdir -p $(BIN_DIR)
	$(GOBUILD) -o $(BIN_DIR)/orchestrator ./services/orchestrator/cmd/main

build-telemetry: ## Build telemetry service
	@mkdir -p $(BIN_DIR)
	$(GOBUILD) -o $(BIN_DIR)/telemetry ./services/telemetry/cmd/main

build-identity: ## Build identity service
	@mkdir -p $(BIN_DIR)
	$(GOBUILD) -o $(BIN_DIR)/identity ./services/identity/cmd/main

build-gateway: ## Build gateway service
	@mkdir -p $(BIN_DIR)
	$(GOBUILD) -o $(BIN_DIR)/gateway ./services/gateway/cmd/main

build-sidecar: ## Build sidecar binary
	@mkdir -p $(BIN_DIR)
	$(GOBUILD) -o $(BIN_DIR)/sidecar ./sidecar/cmd/main

# ─── Test ────────────────────────────────────────────────────────────────────

test: ## Run all unit tests across all Go modules
	$(GOTEST) ./pkg/... ./services/control-plane/... ./services/orchestrator/... \
		./services/telemetry/... ./services/identity/... ./services/gateway/... \
		./sidecar/... ./sdk/go/...

test-int: test-infra-up ## Run integration tests (starts test DB + NATS automatically)
	$(GOTEST) -v -count=1 github.com/argus-platform/argus/services/orchestrator/tests/integration/...
	@$(MAKE) test-infra-down

test-infra-up: ## Start test infrastructure (PostgreSQL + NATS)
	docker compose -f deployments/docker/docker-compose.test.yml up -d --wait

test-infra-down: ## Stop test infrastructure
	docker compose -f deployments/docker/docker-compose.test.yml down -v

test-e2e: ## Run end-to-end tests against local stack
	$(GOTEST) -tags=e2e ./services/control-plane/... ./services/orchestrator/... \
		./services/telemetry/... ./services/identity/... ./services/gateway/... \
		./sidecar/...

test-cover: ## Run tests with coverage report
	$(GOTEST) -coverprofile=coverage.out -covermode=atomic \
		./pkg/... ./services/control-plane/... ./services/orchestrator/... \
		./services/telemetry/... ./services/identity/... ./services/gateway/... \
		./sidecar/... ./sdk/go/...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# ─── Lint & Format ──────────────────────────────────────────────────────────

lint: ## Run golangci-lint across workspace
	@CONFIG="$$(pwd)/.golangci.yml"; \
	for dir in pkg services/control-plane services/orchestrator services/telemetry services/identity services/gateway sidecar sdk/go; do \
		echo "=== Linting $$dir ==="; \
		(cd $$dir && golangci-lint run --config "$$CONFIG" ./...); \
	done

fmt: ## Run gofmt on all Go source files
	gofmt -s -w pkg/ services/ sidecar/

# ─── Run Individual Services ────────────────────────────────────────────────

run-control-plane: ## Run control-plane service locally
	go run ./services/control-plane/cmd/main

run-orchestrator: ## Run orchestrator service locally
	go run ./services/orchestrator/cmd/main

run-telemetry: ## Run telemetry service locally
	go run ./services/telemetry/cmd/main

run-identity: ## Run identity service locally
	go run ./services/identity/cmd/main

run-gateway: ## Run gateway service locally
	go run ./services/gateway/cmd/main

run-sidecar: ## Run sidecar locally
	go run ./sidecar/cmd/main

# ─── Dashboard ───────────────────────────────────────────────────────────────

dashboard-dev: ## Start dashboard dev server (http://localhost:5173)
	cd dashboard && npm install && npm run dev

dashboard-build: ## Build dashboard for production
	cd dashboard && npm install && npm run build

# ─── Docker ──────────────────────────────────────────────────────────────────

docker-build: ## Build all Docker images
	docker build -t $(DOCKER_REGISTRY)/control-plane:$(DOCKER_TAG) -f deployments/docker/Dockerfile.control-plane .
	docker build -t $(DOCKER_REGISTRY)/orchestrator:$(DOCKER_TAG) -f deployments/docker/Dockerfile.orchestrator .
	docker build -t $(DOCKER_REGISTRY)/telemetry:$(DOCKER_TAG) -f deployments/docker/Dockerfile.telemetry .
	docker build -t $(DOCKER_REGISTRY)/identity:$(DOCKER_TAG) -f deployments/docker/Dockerfile.identity .
	docker build -t $(DOCKER_REGISTRY)/gateway:$(DOCKER_TAG) -f deployments/docker/Dockerfile.gateway .
	docker build -t $(DOCKER_REGISTRY)/sidecar:$(DOCKER_TAG) -f deployments/docker/Dockerfile.sidecar .

# ─── Clean ───────────────────────────────────────────────────────────────────

clean: ## Remove build artifacts
	rm -rf $(BIN_DIR)/
	rm -f coverage.out coverage.html
