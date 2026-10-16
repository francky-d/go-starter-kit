# Development commands. Run `make` or `make help` to list them.
#
# Docker and make are the only requirements: the Go targets run in the go
# container of docker-compose.yml. Where Go is installed, as in CI and in the
# container itself, GO_IN_DOCKER=false runs them directly.

.DEFAULT_GOAL := help

GO_IN_DOCKER ?= true

# The container runs as the host user, so the files it writes stay yours.
export HOST_UID ?= $(shell id -u)
export HOST_GID ?= $(shell id -g)

.PHONY: help
help: ## List the available commands
	@awk 'BEGIN {FS = ":.*## "} /^[a-zA-Z_-]+:.*## / {printf "  %-14s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.PHONY: dev
dev: .env ## Start PostgreSQL and the API with live reload, in containers
	docker compose up api-dev

.PHONY: shell
shell: ## Open a shell in the go container, to run any Go command
	docker compose run --rm go bash

# Created once, from .env.example, with a random 32-byte JWT_SECRET, readable
# by you only.
.env:
	@umask 077; secret="$$(head -c 32 /dev/urandom | od -An -tx1 | tr -d ' \n')"; \
	sed "s/^JWT_SECRET=\$$/JWT_SECRET=$$secret/" .env.example > .env
	@echo "Created .env from .env.example, with a random JWT_SECRET."

.PHONY: docker-up
docker-up: ## Start PostgreSQL and wait until it is ready
	docker compose up -d --wait

.PHONY: docker-full
docker-full: .env ## Build the API image and start it with PostgreSQL
	docker compose --profile full up -d --build --wait

.PHONY: docker-down
docker-down: ## Stop the containers, keeping the database data
	docker compose --profile '*' down

.PHONY: docker-reset
docker-reset: ## Stop the containers and delete the database data and Go caches
	docker compose --profile '*' down --volumes

.PHONY: docker-logs
docker-logs: ## Follow the logs of the running containers
	docker compose --profile '*' logs --follow

GO_TARGETS := run build migrate-up migrate-down migrate-version migrate-create \
	test test-cover lint fmt tidy

ifeq ($(GO_IN_DOCKER),true)

# Each Go target runs again in the go container. `run` uses api-dev instead,
# which publishes the API port.
.PHONY: $(GO_TARGETS)
$(filter-out run,$(GO_TARGETS)):
	docker compose run --rm go make $@ $(if $(name),name=$(name))

run: .env
	docker compose run --rm --service-ports api-dev make run

else

.PHONY: run
run: .env migrate-up ## Apply pending migrations, then run the API without live reload
	go run ./cmd/api

.PHONY: build
build: ## Compile the API and the migrate command into bin/
	go build -trimpath -o bin/ ./cmd/...

.PHONY: migrate-up
migrate-up: ## Apply every pending migration
	go run ./cmd/migrate up

.PHONY: migrate-down
migrate-down: ## Roll back the last migration
	go run ./cmd/migrate down

.PHONY: migrate-version
migrate-version: ## Print the current migration version
	go run ./cmd/migrate version

.PHONY: migrate-create
migrate-create: ## Create a migration pair: make migrate-create name=create_posts_table
	go run ./cmd/migrate create $(name)

# -p 1: every package shares the test database, so packages run one at a time.
.PHONY: test
test: ## Run the test suite
	go test -p 1 ./...

.PHONY: test-cover
test-cover: ## Run the test suite with coverage; writes coverage.out and coverage.html
	go test -p 1 -coverprofile=coverage.out -coverpkg=./internal/... ./...
	go tool cover -html=coverage.out -o coverage.html
	@go tool cover -func=coverage.out | tail -n 1

.PHONY: lint
lint: ## Check formatting and go.mod, then run go vet and staticcheck
	@unformatted="$$(find . -name '.?*' -prune -o -type f -name '*.go' -exec gofmt -l {} +)"; \
	if [ -n "$$unformatted" ]; then echo "Run make fmt. Unformatted files:"; echo "$$unformatted"; exit 1; fi
	go mod tidy -diff
	go vet ./...
	go tool staticcheck ./...

.PHONY: fmt
fmt: ## Format every Go file outside hidden directories
	find . -name '.?*' -prune -o -type f -name '*.go' -exec gofmt -w {} +

.PHONY: tidy
tidy: ## Sync go.mod and go.sum with the imports
	go mod tidy

endif
