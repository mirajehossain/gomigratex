# ==============================================================================
# gomigratex Makefile
# A lightweight, production-ready MySQL migration tool for Go applications
# ==============================================================================

# ==============================================================================
# CONFIGURATION
# ==============================================================================

# Project settings
BINARY_NAME := migratex
MODULE_NAME := github.com/mirajehossain/gomigratex
CMD_PATH := ./cmd/migrate

# Database settings for testing
DB_HOST := 127.0.0.1
DB_PORT := 3306
DB_USER := admin
DB_PASS := testpass1
DB_NAME := test
DB_URL := $(DB_USER):$(DB_PASS)@tcp($(DB_HOST):$(DB_PORT))/$(DB_NAME)?parseTime=true&multiStatements=true

# Version detection
# Try to get version from git tag, fallback to dev-{commit} for development
GIT_TAG := $(shell git describe --tags --exact-match 2>/dev/null)
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')

ifdef GIT_TAG
    VERSION := $(GIT_TAG)
else
    VERSION := dev-$(GIT_COMMIT)
endif

# Allow VERSION override from command line
ifdef VERSION_OVERRIDE
    VERSION := $(VERSION_OVERRIDE)
endif

# Build flags
LDFLAGS := -ldflags "-s -w -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT)"
BUILD_FLAGS := $(LDFLAGS)

# Go settings
GOOS := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)

# ==============================================================================
# HELP & INFO
# ==============================================================================

.DEFAULT_GOAL := help

.PHONY: help
help: ## Show this help message
	@echo 'Usage: make [target] [VERSION_OVERRIDE=version]'
	@echo ''
	@echo 'Current settings:'
	@echo '  Binary name: $(BINARY_NAME)'
	@echo '  Version:     $(VERSION)'
	@echo '  Git commit:  $(GIT_COMMIT)'
	@echo '  Build time:  $(BUILD_TIME)'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.PHONY: info
info: ## Show build information
	@echo "Project Information:"
	@echo "  Module:      $(MODULE_NAME)"
	@echo "  Binary:      $(BINARY_NAME)"
	@echo "  Version:     $(VERSION)"
	@echo "  Git commit:  $(GIT_COMMIT)"
	@echo "  Build time:  $(BUILD_TIME)"
	@echo "  Go version:  $$(go version)"
	@echo "  Platform:    $(GOOS)/$(GOARCH)"

# ==============================================================================
# BUILD TARGETS
# ==============================================================================

.PHONY: build
build: ## Build the binary for current platform
	@echo "Building $(BINARY_NAME) $(VERSION)..."
	go build $(BUILD_FLAGS) -o $(BINARY_NAME) $(CMD_PATH)
	@echo "✓ Built $(BINARY_NAME)"

.PHONY: build-linux
build-linux: ## Build for Linux (amd64)
	@echo "Building for Linux..."
	GOOS=linux GOARCH=amd64 go build $(BUILD_FLAGS) -o $(BINARY_NAME)-linux $(CMD_PATH)
	@echo "✓ Built $(BINARY_NAME)-linux"

.PHONY: build-darwin
build-darwin: ## Build for macOS (amd64)
	@echo "Building for macOS..."
	GOOS=darwin GOARCH=amd64 go build $(BUILD_FLAGS) -o $(BINARY_NAME)-darwin $(CMD_PATH)
	@echo "✓ Built $(BINARY_NAME)-darwin"

.PHONY: build-windows
build-windows: ## Build for Windows (amd64)
	@echo "Building for Windows..."
	GOOS=windows GOARCH=amd64 go build $(BUILD_FLAGS) -o $(BINARY_NAME)-windows.exe $(CMD_PATH)
	@echo "✓ Built $(BINARY_NAME)-windows.exe"

.PHONY: build-all
build-all: build-linux build-darwin build-windows ## Build for all platforms
	@echo "✓ Built binaries for all platforms"
	@ls -la $(BINARY_NAME)-*

# ==============================================================================
# DEVELOPMENT TARGETS
# ==============================================================================

.PHONY: dev
dev: build ## Build and show version (development workflow)
	@echo ""
	@./$(BINARY_NAME) version

.PHONY: install
install: ## Install binary to GOPATH/bin
	@echo "Installing $(BINARY_NAME)..."
	go install $(BUILD_FLAGS) $(CMD_PATH)
	@echo "✓ Installed $(BINARY_NAME) to $$(go env GOPATH)/bin"

.PHONY: install-dev
install-dev: ## Install development version
	go install $(BUILD_FLAGS) $(CMD_PATH)

# ==============================================================================
# TESTING TARGETS
# ==============================================================================

.PHONY: test
test: ## Run all tests
	go test ./...

.PHONY: test-verbose
test-verbose: ## Run tests with verbose output
	go test -v ./...

.PHONY: test-coverage
test-coverage: ## Run tests with coverage
	go test -cover ./...

.PHONY: test-coverage-html
test-coverage-html: ## Generate HTML coverage report
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "✓ Coverage report: coverage.html"

.PHONY: test-race
test-race: ## Run tests with race detection
	go test -race ./...

.PHONY: test-integration
test-integration: ## Run integration tests (requires MySQL)
	@echo "Running integration tests against $(DB_HOST):$(DB_PORT)..."
	DB_DSN="$(DB_URL)" go test -tags=integration ./...

# ==============================================================================
# CODE QUALITY TARGETS
# ==============================================================================

.PHONY: fmt
fmt: ## Format Go code
	go fmt ./...

.PHONY: vet
vet: ## Run go vet
	go vet ./...

.PHONY: lint
lint: ## Run golangci-lint (if available)
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "⚠ golangci-lint not installed, skipping..."; \
		echo "  Install: curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b \$$(go env GOPATH)/bin"; \
	fi

.PHONY: check
check: fmt vet test ## Run all checks (format, vet, test)
	@echo "✓ All checks passed"

# ==============================================================================
# MIGRATION TARGETS (for development/testing)
# ==============================================================================

.PHONY: migration-create
migration-create: build ## Create a new migration (usage: make migration-create name=migration_name)
	@if [ -z "$(name)" ]; then \
		echo "Error: name parameter required"; \
		echo "Usage: make migration-create name=add_users_table"; \
		exit 1; \
	fi
	./$(BINARY_NAME) create $(name) --dir migrations

.PHONY: migration-up
migration-up: build ## Apply all pending migrations
	./$(BINARY_NAME) up --dsn "$(DB_URL)" --dir "./migrations" --verbose

.PHONY: migration-down
migration-down: build ## Roll back last migration
	./$(BINARY_NAME) down 1 --dsn "$(DB_URL)" --dir "./migrations" --verbose

.PHONY: migration-down-all
migration-down-all: build ## Roll back all migrations
	./$(BINARY_NAME) down all --dsn "$(DB_URL)" --dir "./migrations" --verbose

.PHONY: migration-status
migration-status: build ## Show migration status
	./$(BINARY_NAME) status --dsn "$(DB_URL)" --dir "./migrations"

.PHONY: migration-status-json
migration-status-json: build ## Show migration status in JSON
	./$(BINARY_NAME) status --dsn "$(DB_URL)" --dir "./migrations" --json

# ==============================================================================
# DOCKER TARGETS
# ==============================================================================

.PHONY: docker-up
docker-up: ## Start MySQL with docker-compose
	docker-compose up -d
	@echo "✓ MySQL started on $(DB_HOST):$(DB_PORT)"

.PHONY: docker-down
docker-down: ## Stop MySQL with docker-compose
	docker-compose down

.PHONY: docker-logs
docker-logs: ## Show MySQL logs
	docker-compose logs mysql

.PHONY: docker-test
docker-test: docker-up ## Run tests with Docker MySQL
	@echo "Waiting for MySQL to be ready..."
	@sleep 5
	$(MAKE) test-integration
	$(MAKE) docker-down

# ==============================================================================
# RELEASE TARGETS
# ==============================================================================

.PHONY: release-check
release-check: ## Check if ready for release
	@echo "Checking release readiness..."
	@if [ -n "$$(git status --porcelain)" ]; then \
		echo "❌ Working directory is not clean"; \
		git status --short; \
		exit 1; \
	fi
	@if [ "$(VERSION)" = "dev-$(GIT_COMMIT)" ]; then \
		echo "❌ Not on a tagged commit"; \
		echo "Current version: $(VERSION)"; \
		echo "Create a tag first: git tag v1.0.0 && git push origin v1.0.0"; \
		exit 1; \
	fi
	@echo "✓ Release check passed for version $(VERSION)"

.PHONY: release-tag
release-tag: ## Create and push a git tag (usage: make release-tag VERSION_OVERRIDE=v1.0.0)
	@if [ -z "$(VERSION_OVERRIDE)" ]; then \
		echo "Error: VERSION_OVERRIDE required"; \
		echo "Usage: make release-tag VERSION_OVERRIDE=v1.0.0"; \
		exit 1; \
	fi
	@if git tag -l | grep -q "^$(VERSION_OVERRIDE)$$"; then \
		echo "❌ Tag $(VERSION_OVERRIDE) already exists"; \
		exit 1; \
	fi
	@echo "Creating and pushing tag $(VERSION_OVERRIDE)..."
	git tag -a $(VERSION_OVERRIDE) -m "Release $(VERSION_OVERRIDE)"
	git push origin $(VERSION_OVERRIDE)
	@echo "✓ Tag $(VERSION_OVERRIDE) created and pushed"

.PHONY: release-build
release-build: release-check build-all ## Build release binaries
	@echo "✓ Release binaries built for version $(VERSION)"

.PHONY: release-create
release-create: ## Create a complete release (usage: make release-create VERSION_OVERRIDE=v1.0.0)
	@if [ -z "$(VERSION_OVERRIDE)" ]; then \
		echo "Error: VERSION_OVERRIDE required"; \
		echo "Usage: make release-create VERSION_OVERRIDE=v1.0.0"; \
		exit 1; \
	fi
	$(MAKE) release-tag VERSION_OVERRIDE=$(VERSION_OVERRIDE)
	$(MAKE) release-build
	@echo "✓ Release $(VERSION_OVERRIDE) created successfully!"
	@echo ""
	@echo "Next steps:"
	@echo "1. Go to https://github.com/mirajehossain/gomigratex/releases"
	@echo "2. Create a new release from tag $(VERSION_OVERRIDE)"
	@echo "3. Upload these binaries:"
	@ls -la $(BINARY_NAME)-* 2>/dev/null || true

# ==============================================================================
# CLEANUP TARGETS
# ==============================================================================

.PHONY: clean
clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_NAME)-linux
	rm -f $(BINARY_NAME)-darwin
	rm -f $(BINARY_NAME)-windows.exe
	rm -f coverage.out
	rm -f coverage.html
	@echo "✓ Cleaned"

.PHONY: clean-all
clean-all: clean ## Clean everything including test artifacts
	rm -rf test-migrations/
	go clean -testcache
	@echo "✓ Deep clean completed"

# ==============================================================================
# UTILITY TARGETS
# ==============================================================================

.PHONY: deps
deps: ## Download and tidy dependencies
	@echo "Managing dependencies..."
	go mod download
	go mod tidy
	@echo "✓ Dependencies updated"

.PHONY: deps-upgrade
deps-upgrade: ## Upgrade all dependencies
	@echo "Upgrading dependencies..."
	go get -u ./...
	go mod tidy
	@echo "✓ Dependencies upgraded"

.PHONY: example
example: build ## Run the embedded example
	@echo "Running embedded example..."
	go run -tags examples ./examples/embedded

# ==============================================================================
# CI/CD TARGETS
# ==============================================================================

.PHONY: ci-test
ci-test: ## Run CI test suite
	go test -race -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

.PHONY: ci-build
ci-build: ## Build for CI (current platform only)
	go build -o $(BINARY_NAME) $(CMD_PATH)

.PHONY: ci-release
ci-release: ## CI release build (all platforms)
	$(MAKE) build-all

# ==============================================================================
# END OF MAKEFILE
# ==============================================================================