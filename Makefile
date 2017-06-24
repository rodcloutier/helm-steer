HELM_HOME ?= $(shell helm home)
HELM_PLUGIN_DIR ?= $(HELM_HOME)/plugins/helm-template
HAS_GLIDE := $(shell command -v glide;)
VERSION := $(shell sed -n -e 's/version:[ "]*\([^"]*\).*/\1/p' plugin.yaml)
DIST := $(CURDIR)/_dist
LDFLAGS := "-X main.version=${VERSION}"

ifeq ($(OS),Windows_NT)
	EXT := .exe
endif

.PHONY: install
install: bootstrap build
	cp steer $(HELM_PLUGIN_DIR)
	cp plugin.yaml $(HELM_PLUGIN_DIR)

.PHONY: hookInstall
hookInstall: bootstrap build

.PHONY: build
build: #generate
	go build -o steer$(EXT) -ldflags $(LDFLAGS) ./main.go

.PHONY: dist
dist: generate
	mkdir -p $(DIST)
	GOOS=linux GOARCH=amd64 go build -o steer -ldflags $(LDFLAGS) ./main.go
	tar -zcvf $(DIST)/helm-steer-linux-$(VERSION).tgz steer README.md LICENSE.txt plugin.yaml
	GOOS=darwin GOARCH=amd64 go build -o steer -ldflags $(LDFLAGS) ./main.go
	tar -zcvf $(DIST)/helm-steer-macos-$(VERSION).tgz steer README.md LICENSE.txt plugin.yaml
	GOOS=windows GOARCH=amd64 go build -o steer.exe -ldflags $(LDFLAGS) ./main.go
	tar -zcvf $(DIST)/helm-steer-windows-$(VERSION).tgz steer.exe README.md LICENSE.txt plugin.yaml

.PHONY: bootstrap
bootstrap:
ifndef HAS_GLIDE
	go get -u github.com/Masterminds/glide
endif
	glide install --strip-vendor
