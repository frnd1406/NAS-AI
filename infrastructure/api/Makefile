.PHONY: help build run test test-coverage test-security lint security-scan clean

help:  ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

build: ## Build the API binary
	@echo "Building API..."
	@mkdir -p bin
	@go build -o bin/api ./src/main.go
	@echo "✅ Build complete: bin/api"

run: ## Run the API server (requires JWT_SECRET env var!)
	@if [ -z "$$JWT_SECRET" ]; then \
		echo "❌ ERROR: JWT_SECRET environment variable is required!"; \
		echo "Generate one with: export JWT_SECRET=\$$(openssl rand -base64 32)"; \
		exit 1; \
	fi
	@echo "Starting API server..."
	@./bin/api

test: ## Run all tests
	@echo "Running tests..."
	@go test -v ./...

test-coverage: ## Run tests with coverage (min 80%)
	@echo "Running tests with coverage..."
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@go tool cover -func=coverage.out | grep total | awk '{print "Coverage: " $$3}'
	@echo "Coverage report: coverage.html"

test-coverage-ci: ## Run tests with coverage for CI (exits non-zero if <80%)
	@echo "Running tests with coverage (CI mode)..."
	@go test -coverprofile=coverage.out -covermode=atomic ./...
	@go tool cover -func=coverage.out | tee coverage.txt
	@COVERAGE=$$(go tool cover -func=coverage.out | grep total | awk '{print $$3}' | tr -d '%'); \
	if [ "$${COVERAGE%.*}" -lt 80 ]; then \
		echo "❌ Coverage $${COVERAGE}% is below 80% threshold"; \
		exit 1; \
	else \
		echo "✅ Coverage $${COVERAGE}% meets 80% threshold"; \
	fi

test-security: ## Run security tests only
	@echo "Running security tests..."
	@go test -v ./tests/security/...

lint: ## Run golangci-lint
	@echo "Running linter..."
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run ./...; \
	else \
		echo "⚠️  golangci-lint not installed. Install: https://golangci-lint.run/usage/install/"; \
		exit 1; \
	fi

security-scan: ## Run security scans (gosec + gitleaks)
	@echo "Running security scans..."
	@echo "=== gosec (Go security checker) ==="
	@if command -v gosec > /dev/null; then \
		gosec -quiet ./...; \
	else \
		echo "⚠️  gosec not installed. Install: go install github.com/securego/gosec/v2/cmd/gosec@latest"; \
	fi
	@echo ""
	@echo "=== gitleaks (Secret scanner) ==="
	@if command -v gitleaks > /dev/null; then \
		gitleaks detect --source . --verbose; \
	else \
		echo "⚠️  gitleaks not installed. Install: https://github.com/gitleaks/gitleaks"; \
	fi

clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -rf bin/
	@rm -f coverage.out coverage.html
	@echo "✅ Clean complete"

.DEFAULT_GOAL := help
