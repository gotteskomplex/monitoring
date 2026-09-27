# Central developer commands. See CLAUDE.md.
SHELL := /bin/bash
GOBIN ?= $(shell go env GOPATH)/bin
export PATH := $(GOBIN):$(PATH)

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w \
	-X github.com/gotteskomplex/monitoring/internal/platform/version.Version=$(VERSION) \
	-X github.com/gotteskomplex/monitoring/internal/platform/version.Commit=$(COMMIT) \
	-X github.com/gotteskomplex/monitoring/internal/platform/version.Date=$(DATE)

# Pinned tool versions (also used in CI).
BUF_VERSION            := v1.47.2
PROTOC_GEN_GO_VERSION  := v1.36.5
PROTOC_GEN_GRPC_VERSION := v1.5.1
SQLC_VERSION           := v1.27.0
OAPI_CODEGEN_VERSION   := v2.4.1
GOLANGCI_VERSION       := v2.5.0

COMPOSE := docker compose -f deploy/compose/docker-compose.yml

.PHONY: help tools gen check-gen lint lint-go lint-proto lint-web test test-go test-web test-int \
	build build-all web run stop migrate clean

help: ## Show targets
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-12s %s\n", $$1, $$2}'

# Tools are built with the project's Go version (go.mod), not the system default.
tools: export GOTOOLCHAIN = $(shell go env GOVERSION)
tools: ## Install pinned code generators and linters
	go install github.com/bufbuild/buf/cmd/buf@$(BUF_VERSION)
	go install google.golang.org/protobuf/cmd/protoc-gen-go@$(PROTOC_GEN_GO_VERSION)
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@$(PROTOC_GEN_GRPC_VERSION)
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@$(SQLC_VERSION)
	go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@$(OAPI_CODEGEN_VERSION)
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_VERSION)

gen: ## Regenerate code from proto, SQL and OpenAPI
	buf generate
	sqlc generate
	oapi-codegen -config api/oapi-codegen.yaml api/openapi.yaml
	cd web && npm run gen

check-gen: gen ## Fail if generated code is out of date
	git diff --exit-code -- internal/gen web/src/gen

lint: lint-go lint-proto lint-web ## Run all linters

lint-go:
	golangci-lint run ./...

lint-proto:
	buf lint

lint-web:
	cd web && npm run lint && npm run typecheck

test: test-go test-web ## Unit tests (fast, no Docker)

test-go:
	go test -race ./...

test-web:
	cd web && npm test

test-int: ## Integration tests with Testcontainers (Docker required)
	go test -race -tags integration -count=1 ./test/...

build: ## Build master and satellite for the host platform
	CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o bin/master ./cmd/master
	CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o bin/satellite ./cmd/satellite

build-all: ## Cross-compile satellite (linux amd64/arm64, windows amd64) and master (linux)
	for t in linux/amd64 linux/arm64 windows/amd64; do \
		os=$${t%/*}; arch=$${t#*/}; ext=$$( [ $$os = windows ] && echo .exe ); \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch go build -trimpath -ldflags "$(LDFLAGS)" \
			-o bin/satellite-$$os-$$arch$$ext ./cmd/satellite || exit 1; \
	done
	for a in amd64 arm64; do \
		CGO_ENABLED=0 GOOS=linux GOARCH=$$a go build -trimpath -ldflags "$(LDFLAGS)" -o bin/master-linux-$$a ./cmd/master || exit 1; \
	done

web: ## Build the web UI and copy it into the master's embed directory
	cd web && npm ci && npm run build
	find internal/master/webui/dist -mindepth 1 ! -name .keep -delete
	cp -r web/dist/. internal/master/webui/dist/

run: ## Start the complete local environment (docker compose)
	$(COMPOSE) up --build

stop: ## Stop the local environment
	$(COMPOSE) down

migrate: ## Apply migrations against MON_DB_MIGRATOR_DSN
	go run ./cmd/master migrate

clean:
	rm -rf bin web/dist
	find internal/master/webui/dist -mindepth 1 ! -name .keep -delete
