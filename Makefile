SHELL := /usr/bin/env bash
VERSION := $(shell tr -d '[:space:]' < VERSION)
DIST := dist

.PHONY: all build test vet check desktop-validate container clean version

all: check build

version:
	@echo $(VERSION)

build:
	@mkdir -p $(DIST)
	go build -trimpath -ldflags "-s -w -X main.version=$(VERSION)" -o $(DIST)/kingai ./cmd/kingai
	go build -trimpath -ldflags "-s -w -X main.version=$(VERSION)" -o $(DIST)/kingaid ./cmd/kingaid
	go build -trimpath -ldflags "-s -w -X main.version=$(VERSION)" -o $(DIST)/kingai-execd ./cmd/kingai-execd
	go build -trimpath -ldflags "-s -w -X main.version=$(VERSION)" -o $(DIST)/kingai-update ./cmd/kingai-update
	go build -trimpath -ldflags "-s -w -X main.version=$(VERSION)" -o $(DIST)/kingai-installer ./cmd/kingai-installer
	go build -trimpath -ldflags "-s -w -X main.version=$(VERSION)" -o $(DIST)/kingai-recovery ./cmd/kingai-recovery
	go build -trimpath -ldflags "-s -w -X main.version=$(VERSION)" -o $(DIST)/kingai-desktop-bridge ./cmd/kingai-desktop-bridge

vet:
	go vet ./...

test:
	go test ./...

desktop-validate:
	bash scripts/validate-desktop.sh
	bash scripts/validate-desktop-private.sh
	bash scripts/test-desktop-intelligence-launcher.sh
	go test ./internal/desktop ./internal/statuspub ./internal/desktopbridge ./internal/memory ./cmd/kingaid ./cmd/kingai-desktop-bridge

check: vet test desktop-validate
	bash -n scripts/*.sh
	@test -f profiles/server.yaml
	@test -f profiles/desktop.yaml
	@test ! -e profiles/pc.yaml
	@test -f profiles/iot.yaml
	@test -f profiles/container.yaml
	@test -f container/Dockerfile
	@test -f systemd/kingai-execd.service

container:
	KINGAI_CONTAINER_PLATFORMS=linux/amd64 bash scripts/build-container.sh

clean:
	rm -rf $(DIST) out build
