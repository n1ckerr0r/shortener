GO ?= go
GOCACHE ?= $(CURDIR)/.cache/go-build
GOLANGCI_LINT_CACHE ?= $(CURDIR)/.cache/golangci-lint
GOLANGCI_LINT ?= golangci-lint

.PHONY: all build test test-race vet lint fmt check run docker-build compose-up compose-down

all: check

build:
	GOCACHE=$(GOCACHE) $(GO) build ./cmd/api ./cmd/analytics ./cmd/urlcheck

test:
	GOCACHE=$(GOCACHE) $(GO) test ./...

test-race:
	GOCACHE=$(GOCACHE) $(GO) test -race ./...

vet:
	GOCACHE=$(GOCACHE) $(GO) vet ./...

lint:
	GOCACHE=$(GOCACHE) GOLANGCI_LINT_CACHE=$(GOLANGCI_LINT_CACHE) $(GOLANGCI_LINT) run ./...

fmt:
	GOCACHE=$(GOCACHE) $(GO) fmt ./...

check: fmt vet test lint

run:
	GOCACHE=$(GOCACHE) $(GO) run ./cmd/api

docker-build:
	docker build --build-arg SERVICE=api -t shortener-api:local .
	docker build --build-arg SERVICE=analytics -t shortener-analytics:local .
	docker build --build-arg SERVICE=urlcheck -t shortener-urlcheck:local .

compose-up:
	docker compose up --build

compose-down:
	docker compose down
