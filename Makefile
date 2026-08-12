SHELL := /usr/bin/env bash
VERSION := $(shell tr -d '[:space:]' < VERSION)
DIST := dist

.PHONY: all build test vet check clean version

all: check build

version:
	@echo $(VERSION)

build:
	@mkdir -p $(DIST)
	go build -trimpath -ldflags "-s -w -X main.version=$(VERSION)" -o $(DIST)/kingai ./cmd/kingai

vet:
	go vet ./...

test:
	go test ./...

check: vet test
	bash -n scripts/*.sh
	@test -f profiles/server.yaml
	@test -f profiles/desktop.yaml
	@test -f profiles/iot.yaml

clean:
	rm -rf $(DIST) out build
