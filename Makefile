# Scry API Makefile
# This file provides standardized commands for common development tasks

# Default target
.DEFAULT_GOAL := help

# Variables
BINARY_NAME := scry-api
BUILD_DIR := ./bin
MAIN_PACKAGE := ./cmd/server

# Help target
.PHONY: help
help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Server operations
.PHONY: run-server
run-server: ## Run the API server
	go run $(MAIN_PACKAGE) $(FLAGS)

# Database operations
.PHONY: migrate-up
migrate-up: ## Apply database migrations
	go run $(MAIN_PACKAGE) -migrate=up

.PHONY: migrate-down
migrate-down: ## Rollback the last migration
	go run $(MAIN_PACKAGE) -migrate=down

.PHONY: migrate-status
migrate-status: ## Show migration status
	go run $(MAIN_PACKAGE) -migrate=status

.PHONY: migrate-version
migrate-version: ## Show current migration version
	go run $(MAIN_PACKAGE) -migrate=version

.PHONY: migrate-create
migrate-create: ## Create a new migration (use NAME=<migration_name>)
	go run $(MAIN_PACKAGE) -migrate=create -name=$(NAME)

# Build operations
.PHONY: build
build: ## Build the application binary
	go build -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PACKAGE)

.PHONY: build-all
build-all: ## Build all binaries
	go build ./...

# Testing
.PHONY: test
test: ## Run all tests
	go test ./...

.PHONY: test-verbose
test-verbose: ## Run tests with verbose output
	go test -v ./...

.PHONY: test-integration
test-integration: ## Run integration tests
	go test -v -tags=integration ./...

.PHONY: test-no-deps
test-no-deps: ## Run tests without external dependencies
	go test -v -tags=test_without_external_deps ./...

.PHONY: test-coverage
test-coverage: ## Run tests with coverage report
	go test -cover -tags="test_without_external_deps" ./...

.PHONY: test-coverage-html
test-coverage-html: ## Generate HTML coverage report
	go test -coverprofile=coverage.out -tags="test_without_external_deps" ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

.PHONY: test-coverage-all
test-coverage-all: ## Run all tests (unit + integration) with coverage
	go test -coverprofile=coverage-unit.out -covermode=atomic ./...
	go test -tags=integration -coverprofile=coverage-integration.out -covermode=atomic ./...
	go run scripts/merge-coverage.go coverage-unit.out coverage-integration.out > coverage.out
	go tool cover -func=coverage.out

.PHONY: test-coverage-postgres
test-coverage-postgres: ## Run postgres package tests with coverage
	go test -coverprofile=coverage-postgres-unit.out -covermode=atomic ./internal/platform/postgres/...
	go test -tags=integration -coverprofile=coverage-postgres-integration.out -covermode=atomic -coverpkg=./internal/platform/postgres ./internal/platform/postgres/...
	go run scripts/merge-coverage.go coverage-postgres-unit.out coverage-postgres-integration.out > coverage-postgres.out
	go tool cover -func=coverage-postgres.out
	go tool cover -html=coverage-postgres.out -o coverage-postgres.html

# Code quality
.PHONY: lint
lint: ## Run golangci-lint
	golangci-lint run

.PHONY: lint-fix
lint-fix: ## Run golangci-lint with fix flag
	golangci-lint run --fix

.PHONY: fmt
fmt: ## Format code using gofmt and goimports
	go fmt ./...
	goimports -w .

# Dependencies
.PHONY: deps
deps: ## Download dependencies
	go mod download

.PHONY: deps-tidy
deps-tidy: ## Tidy up dependencies
	go mod tidy

.PHONY: deps-verify
deps-verify: ## Verify dependencies
	go mod verify

# Cleanup
.PHONY: clean
clean: ## Remove build artifacts
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

# Development helpers
.PHONY: check
check: lint test ## Run linting and tests

.PHONY: pre-commit
pre-commit: fmt deps-tidy lint test ## Run pre-commit checks

# Docker operations (if applicable)
.PHONY: docker-build
docker-build: ## Build Docker image
	docker build -t $(BINARY_NAME) .

.PHONY: docker-run
docker-run: ## Run Docker container
	docker run -p 8080:8080 $(BINARY_NAME)

# CI operations
.PHONY: ci-build
ci-build: deps lint test build ## Run CI build steps

.PHONY: ci-test
ci-test: lint test-coverage ## Run CI test steps

# Infrastructure testing (integration tests, no coverage expected)
.PHONY: test-infrastructure
test-infrastructure: ## Run infrastructure integration tests
	@echo "Running infrastructure integration tests (no coverage expected)..."
	go test -v ./infrastructure/...

.PHONY: test-infrastructure-docker
test-infrastructure-docker: ## Run Docker-based infrastructure tests
	@echo "Running Docker-based infrastructure tests..."
	DOCKER_TEST=1 go test -v ./infrastructure/local_dev/...

.PHONY: test-infrastructure-terraform
test-infrastructure-terraform: ## Run Terraform infrastructure tests
	@echo "Running Terraform infrastructure tests..."
	TERRATEST_ENABLED=1 go test -v ./infrastructure/terraform/test/...

# CI-matching commands
.PHONY: test-ci-local
test-ci-local: ## Run tests matching CI environment
	go test -v -race -cover -tags=integration,test_without_external_deps ./...

.PHONY: test-ci-package
test-ci-package: ## Run tests for specific package matching CI (use PKG=package/path)
	@if [ -z "$(PKG)" ]; then echo "Usage: make test-ci-package PKG=internal/service/card_review"; exit 1; fi
	go test -v -json -race -coverprofile=coverage.out -tags=integration,test_without_external_deps ./$(PKG)/...

.PHONY: lint-ci
lint-ci: ## Run linting with CI build tags
	golangci-lint run --build-tags=integration,test_without_external_deps
