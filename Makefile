# See https://tech.davis-hansson.com/p/make/
SHELL := bash
.DELETE_ON_ERROR:
.SHELLFLAGS := -eu -o pipefail -c
.DEFAULT_GOAL := all
MAKEFLAGS += --warn-undefined-variables
MAKEFLAGS += --no-builtin-rules
MAKEFLAGS += --no-print-directory
BIN := .tmp/bin
export PATH := $(BIN):$(PATH)
export GOBIN := $(abspath $(BIN))
COPYRIGHT_HOLDER := Buf Technologies, Inc.
COPYRIGHT_YEARS := 2023
LICENSE_IGNORE :=

.PHONY: help
help: ## Describe useful make targets
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "%-30s %s\n", $$1, $$2}'

.PHONY: all
all: ## Build, test, and lint (default)
	$(MAKE) test
	$(MAKE) lint

.PHONY: clean
clean: ## Delete intermediate build artifacts
	@# -X only removes untracked files, -d recurses into directories, -f actually removes files/dirs
	git clean -Xdf

.PHONY: test
test: build ## Run unit tests
	go test -vet=off -race -cover ./...

.PHONY: bench
bench: BENCH ?= .*
bench: build ## Run benchmarks for root package
	go test -vet=off -run '^$$' -bench '$(BENCH)' -benchmem -cpuprofile cpu.pprof -memprofile mem.pprof .

.PHONY: build
build: generate ## Build all packages
	go build ./...

.PHONY: install
install: ## Install all binaries
	go install ./...

.PHONY: lint
lint: $(BIN)/golangci-lint ## Lint Go
	go vet ./...
	golangci-lint run --modules-download-mode=readonly --timeout=3m0s

.PHONY: lintfix
lintfix: $(BIN)/golangci-lint ## Automatically fix some lint errors
	golangci-lint run --fix --modules-download-mode=readonly --timeout=3m0s

.PHONY: generate
generate: $(BIN)/license-header ## Regenerate code and licenses
	license-header \
		--license-type apache \
		--copyright-holder "$(COPYRIGHT_HOLDER)" \
		--year-range "$(COPYRIGHT_YEARS)" $(LICENSE_IGNORE)

.PHONY: upgrade
upgrade: ## Upgrade dependencies
	go get -u -t ./... && go mod tidy -v

.PHONY: checkgenerate
checkgenerate:
	@# Used in CI to verify that `make generate` doesn't produce a diff.
	test -z "$$(git status --porcelain | tee /dev/stderr)"

.PHONY: release
release: $(BIN)/minisign ## Generate release assets
	DOCKER_IMAGE=golang:1.21-bullseye bash scripts/release.bash

$(BIN)/license-header: Makefile
	@mkdir -p $(@D)
	go install github.com/bufbuild/buf/private/pkg/licenseheader/cmd/license-header@v1.28.1

$(BIN)/golangci-lint: Makefile
	@mkdir -p $(@D)
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.60.3

$(BIN)/minisign: Makefile
	@mkdir -p $(@D)
	go install aead.dev/minisign/cmd/minisign@v0.2.1
