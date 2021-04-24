#!/usr/bin/env bash

SHELL := /bin/bash

PKG := github.com/dh1tw/remoteSwitch
COMMITID := $(shell git describe --always --long --dirty)
COMMIT := $(shell git rev-parse --short HEAD)
VERSION := $(shell git describe --tags)

PKG_LIST := $(shell go list ${PKG}/... | grep -v /vendor/)
GO_FILES := $(shell find . -name '*.go' | grep -v /vendor/)

GOOS := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)

all: vue_production generate build

# replace the debug version of vue.js with it's production version
vue_production:
	find hub/html/index.html -exec sed -i '' 's/vue.js/vue.min.js/g' {} \;

# replace the debug version of vue.js with it's production version
vue_debug:
	find hub/html/index.html -exec sed -i '' 's/vue.min.js/vue.js/g' {} \;

# embed the static files into a go file
generate:
	go generate ./...

build:
	go build -v -ldflags="-X github.com/dh1tw/remoteSwitch/cmd.commitHash=${COMMIT} \
		-X github.com/dh1tw/remoteSwitch/cmd.version=${VERSION}"

# build and strip off the dwraf table. This will reduce the file size
dist:
	go build -v -ldflags="-w -X github.com/dh1tw/remoteSwitch/cmd.commitHash=${COMMIT} \
		-X github.com/dh1tw/remoteSwitch/cmd.version=${VERSION}"
	# compress binary
	# there is a know issue that upx currently doesn't work with darwin/arm64.
	# See https://github.com/upx/upx/issues/424
	# until it's resolved, we ship darwin/arm64 uncompressed.
	if [ "${GOOS}" == "windows" ]; \
		then upx remoteSwitch.exe; \
		else \
		if [ "${GOOS}" == "darwin" ] && [ "${GOARCH}" == "arm64" ]; \
			then true; \
		else upx remoteSwitch; \
		fi \
	fi

vet:
	@go vet ${PKG_LIST}

lint:
	@for file in ${GO_FILES} ;  do \
		golint $$file ; \
	done

install-deps:
	go get golang.org/x/tools/cmd/stringer

clean:
	-@rm remoteSwitch remoteSwitch-v*

.PHONY: build dist vet lint clean install-deps generate vue_production vue_debug