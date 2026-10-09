# syntax=docker/dockerfile:1

# Build stage: compile static binaries for the API and the migrate command.
# The cache mounts keep downloaded modules and compiled packages between
# builds, so a rebuild after a code change only recompiles what changed.
FROM golang:1.26-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/ ./cmd/api ./cmd/migrate

# Runtime stage: no shell, no package manager, runs as an unprivileged user.
FROM gcr.io/distroless/static-debian13:nonroot

COPY --from=build /out/api /out/migrate /usr/local/bin/

USER nonroot:nonroot
EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/api"]
