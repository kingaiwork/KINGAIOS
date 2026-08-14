SHELL := /usr/bin/env bash
VERSION := $(shell tr -d '[:space:]' < VERSION)
DIST := dist

.PHONY: all build test vet check sentinel container clean version

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

sentinel:
	@mkdir -p $(DIST)
	go build -trimpath -ldflags "-s -w -X main.version=$(VERSION)" -o $(DIST)/kingai-sentinel ./cmd/kingai-sentinel

vet:
	go vet ./...

test:
	go test ./...

check: vet test
	bash -n scripts/*.sh
	@test -f profiles/server.yaml
	@test -f profiles/desktop.yaml
	@test -f profiles/iot.yaml
	@test -f profiles/container.yaml
	@test -f profiles/sentinel.yaml
	@test -f configs/sentinel.json
	@test -f sentinel/web/index.html
	@test -f sentinel/feeds/feeds.json
	@test -f sentinel/packs/catalog.json
	@test -f container/Dockerfile
	@test -f systemd/kingai-execd.service
	@test -f systemd/kingai-sentinel.service
	@test -f systemd/kingai-threat-intel.timer

container:
	KINGAI_CONTAINER_PLATFORMS=linux/amd64 bash scripts/build-container.sh

clean:
	rm -rf $(DIST) out build
