SHELL := /bin/bash

.DEFAULT_GOAL := build

.PHONY: build test fmt fmt-check lint docs check-generated release-smoke

BIN_DIR := $(CURDIR)/dist
BIN := $(BIN_DIR)/beehiiv
CMD := ./cmd/beehiiv
GEN_DOCS_CMD := ./cmd/gen-docs
REFERENCE_DIR := $(CURDIR)/docs/reference/cli
MAN_DIR := $(CURDIR)/share/man/man1
COMPLETIONS_DIR := $(CURDIR)/share/completions

VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT := $(shell git rev-parse --short=12 HEAD 2>/dev/null || echo none)
DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -X github.com/deldrid1/beehiiv-cli/internal/buildinfo.Version=$(VERSION) -X github.com/deldrid1/beehiiv-cli/internal/buildinfo.Commit=$(COMMIT) -X github.com/deldrid1/beehiiv-cli/internal/buildinfo.Date=$(DATE)

build:
	@mkdir -p $(BIN_DIR)
	@go build -ldflags "$(LDFLAGS)" -o $(BIN) $(CMD)

test:
	@go test ./...

fmt:
	@find . -name '*.go' -not -path './dist/*' -print0 | xargs -0 gofmt -w

fmt-check:
	@test -z "$$(find . -name '*.go' -not -path './dist/*' -print0 | xargs -0 gofmt -l)"

lint:
	@go vet ./...

docs:
	@go run $(GEN_DOCS_CMD) --reference-dir $(REFERENCE_DIR) --man-dir $(MAN_DIR) --completion-dir $(COMPLETIONS_DIR)

check-generated:
	@./scripts/check-generated.sh

release-smoke:
	@./scripts/release-smoke.sh
