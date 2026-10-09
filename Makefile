# Development commands. Run `make` or `make help` to list them.

.DEFAULT_GOAL := help

.PHONY: help
help: ## List the available commands
	@awk 'BEGIN {FS = ":.*## "} /^[a-zA-Z_-]+:.*## / {printf "  %-14s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.PHONY: run
run: ## Run the API (reads .env when present)
	go run ./cmd/api

.PHONY: build
build: ## Compile the API into bin/api
	go build -trimpath -o bin/api ./cmd/api

.PHONY: docker-up
docker-up: ## Start PostgreSQL and wait until it is ready
	docker compose up -d --wait

.PHONY: docker-down
docker-down: ## Stop PostgreSQL, keeping its data
	docker compose down

.PHONY: docker-reset
docker-reset: ## Stop PostgreSQL and delete its data
	docker compose down --volumes

.PHONY: docker-logs
docker-logs: ## Follow the PostgreSQL logs
	docker compose logs --follow postgres

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
