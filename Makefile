.PHONY: help build install test docs clean

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-15s %s\n", $$1, $$2}'

build: ## Build the ramp binary
	go build -o ramp .

install: build ## Build and install to /usr/local/bin (requires sudo)
	sudo ./install.sh

test: ## Run all tests
	go test ./...

test-coverage: ## Run tests with coverage report
	go test ./... -cover -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

docs: ## Generate command documentation
	go run scripts/gen-docs.go

docs-verify: docs ## Verify docs are up to date (fails if changes needed)
	@git diff --exit-code docs/commands/ || (echo "❌ Docs are out of sync. Run 'make docs' and commit changes." && exit 1)
	@echo "✅ Docs are up to date"

clean: ## Remove build artifacts
	rm -f ramp coverage.out coverage.html

dev: build ## Build and run with --help
	./ramp --help

.DEFAULT_GOAL := help
