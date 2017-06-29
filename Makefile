HELM_HOME ?= $(shell helm home)
HELM_PLUGIN_DIR ?= $(HELM_HOME)/plugins/helm-steer
HAS_DEP := $(shell command -v dep;)
VERSION = $(shell cat VERSION)
DIST := $(CURDIR)/_dist

ifeq ($(OS),Windows_NT)
	EXT := .exe
endif

.PHONY: all
all: build

VERSION:
	@(git describe --candidates 0 --dirty 2>/dev/null || (echo -n "0.0.0-${USERNAME}-"; git rev-parse --short HEAD)) > $@
	@echo -n "Version: " && cat $@

.PHONY: generate
generate: VERSION
	scripts/generate-version.sh

.PHONY: install
install: bootstrap build
	cp steer $(HELM_PLUGIN_DIR)
	cp plugin.yaml $(HELM_PLUGIN_DIR)

.PHONY: build
build: generate
	go build -o steer$(EXT) ./main.go

.PHONY: test
test:
	go test ./pkg/... --cover
	go test ./cmd/... --cover

.PHONY: dist
dist: generate
	mkdir -p $(DIST)
	GOOS=linux GOARCH=amd64 go build -o steer ./main.go
	tar -zcvf $(DIST)/helm-steer-linux-$(VERSION).tgz steer README.md LICENSE.txt plugin.yaml
	GOOS=darwin GOARCH=amd64 go build -o steer ./main.go
	tar -zcvf $(DIST)/helm-steer-macos-$(VERSION).tgz steer README.md LICENSE.txt plugin.yaml
	GOOS=windows GOARCH=amd64 go build -o steer.exe ./main.go
	tar -zcvf $(DIST)/helm-steer-windows-$(VERSION).tgz steer.exe README.md LICENSE.txt plugin.yaml

.PHONY: bootstrap
bootstrap:
ifndef HAS_DEP
	go get -u github.com/golang/dep/cmd/dep
endif
	dep ensure -update
