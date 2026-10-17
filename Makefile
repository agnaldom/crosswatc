GO_LDFLAGS := -X github.com/agnaldom/crosswatc/internal/version.Version=$(shell git describe --tags --always --dirty 2>/dev/null || echo dev) \
              -X github.com/agnaldom/crosswatc/internal/version.Commit=$(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATABASE_URL ?= postgres://crosswatc:crosswatc@localhost:5432/crosswatc?sslmode=disable

.PHONY: build test test-integration lint fmt web-install web-lint web-build db-up db-down

build:
	go build -ldflags "$(GO_LDFLAGS)" -o bin/crosswatc-api ./services/api
	go build -ldflags "$(GO_LDFLAGS)" -o bin/crosswatc-worker ./services/worker

test:
	go test -race -count=1 ./...

test-integration:
	CROSSWATC_TEST_DATABASE_URL=$(DATABASE_URL) go test -race -count=1 -tags integration ./...

lint:
	golangci-lint run ./...

fmt:
	gofmt -w services internal

web-install:
	pnpm install --frozen-lockfile

web-lint:
	pnpm --filter web lint

web-build:
	pnpm --filter web build

db-up:
	docker compose up -d postgres

db-down:
	docker compose down
