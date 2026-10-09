# Development commands. Run `make` or `make help` to list them.

.DEFAULT_GOAL := help

.PHONY: help
help: ## List the available commands
	@awk 'BEGIN {FS = ":.*## "} /^[a-zA-Z_-]+:.*## / {printf "  %-14s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.PHONY: test
test: ## Run the test suite
	go test ./...

.PHONY: lint
lint: ## Check formatting and go.mod, then run go vet and staticcheck
	@unformatted="$$(gofmt -l .)"; \
	if [ -n "$$unformatted" ]; then echo "Run make fmt. Unformatted files:"; echo "$$unformatted"; exit 1; fi
	go mod tidy -diff
	go vet ./...
	go tool staticcheck ./...

.PHONY: fmt
fmt: ## Format every Go file
	gofmt -w .

.PHONY: tidy
tidy: ## Sync go.mod and go.sum with the imports
	go mod tidy
