#!/usr/bin/env bash

SHELL := /bin/bash

PKG := github.com/dh1tw/remoteSwitch
COMMITID := $(shell git describe --always --long --dirty)
COMMIT := $(shell git rev-parse --short HEAD)
VERSION := $(shell git describe --tags)

PKG_LIST := $(shell go list ${PKG}/... | grep -v /vendor/)
GO_FILES := $(shell find . -name '*.go' | grep -v /vendor/)

all: vue_production generate dist vue_debug

# replace the debug version of vue.js with it's production version
vue_production:
	find html/index.html -exec sed -i '' 's/vue.js/vue.min.js/g' {} \;

# replace the debug version of vue.js with it's production version
vue_debug:
	find html/index.html -exec sed -i '' 's/vue.min.js/vue.js/g' {} \;

# embed the static files into a go file
generate:
	go generate ./...
	cd hub; \
	rice embed-go

build:
	go build -v -ldflags="-X github.com/dh1tw/remoteSwitch/cmd.commitHash=${COMMIT} \
		-X github.com/dh1tw/remoteSwitch/cmd.version=${VERSION}"

# build and strip off the dwraf table. This will reduce the file size
dist:
	go build -v -ldflags="-w -X github.com/dh1tw/remoteSwitch/cmd.commitHash=${COMMIT} \
		-X github.com/dh1tw/remoteSwitch/cmd.version=${VERSION}"
	# compress binary
	if [ "${GOOS}" == "windows" ]; then upx remoteSwitch.exe; else upx remoteSwitch; fi

vet:
	@go vet ${PKG_LIST}

lint:
	@for file in ${GO_FILES} ;  do \
		golint $$file ; \
	done

install-deps:
	go get golang.org/x/tools/cmd/stringer
	go get github.com/GeertJohan/go.rice/rice

clean:
	-@rm remoteSwitch remoteSwitch-v*

.PHONY: build dist vet lint clean install-deps generate vue_production vue_debug