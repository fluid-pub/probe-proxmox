# Makefile for Fluid Proxmox VE probe

BINARY_NAME=proxmox-probe
BUILD_DIR=build
CONFIG_DIR=config
STATE_DIR=state

GO=go
GOOS?=$(shell go env GOOS)
GOARCH?=$(shell go env GOARCH)

VERSION?=0.1.0
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

LDFLAGS=-ldflags "-X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME} -X main.GitCommit=${GIT_COMMIT}"

.PHONY: all build clean run dev test deps fmt help

all: clean build

deps:
	@test -d core || test -d ../core || (echo "Run: git submodule update --init --recursive (public repo) or develop from fluid monorepo with ../core" && exit 1)
	$(GO) mod download
	@$(GO) mod tidy

build: deps
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd

clean:
	@rm -rf $(BUILD_DIR)

run: build
	@cd $(BUILD_DIR) && ./$(BINARY_NAME) -config ../$(CONFIG_DIR)/probe.yml

dev: deps
	@test -f env.secrets || (echo "Missing env.secrets (see env.secrets.example)" && exit 1)
	@set -a; . ./env.secrets; set +a; $(GO) run ./cmd -config $(CONFIG_DIR)/probe.yml

test: deps
	$(GO) test ./...

fmt:
	gofmt -w .
	$(GO) fmt ./...

help:
	@echo "Targets: deps build clean run dev test fmt"
