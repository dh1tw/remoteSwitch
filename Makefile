#!/usr/bin/env bash

PKG := github.com/dh1tw/remoteSwitch
COMMITID := $(shell git describe --always --long --dirty)
COMMIT := $(shell git rev-parse --short HEAD)
VERSION := $(shell git describe --tags)

PKG_LIST := $(shell go list ${PKG}/... | grep -v /vendor/)
GO_FILES := $(shell find . -name '*.go' | grep -v /vendor/)
all: build

build:
	go build -v -ldflags="-X github.com/dh1tw/remoteSwitch/cmd.commitHash=${COMMIT} \
		-X github.com/dh1tw/remoteSwitch/cmd.version=${VERSION}"

# strip off dwraf table - used for travis CI

generate:
	cd hub; \
	rice embed-go

dist:
	go build -v -ldflags="-w -X github.com/dh1tw/remoteSwitch/cmd.commitHash=${COMMIT} \
		-X github.com/dh1tw/remoteSwitch/cmd.version=${VERSION}"

# test:
# 	@go test -short ${PKG_LIST}

vet:
	@go vet ${PKG_LIST}

lint:
	@for file in ${GO_FILES} ;  do \
		golint $$file ; \
	done

test:
	go test ./...

install:
	go install -v -ldflags="-w -X github.com/dh1tw/remoteSwitch/cmd.commitHash=${COMMIT} \
		-X github.com/dh1tw/remoteSwitch/cmd.version=${VERSION}"

install-deps:
	go get github.com/GeertJohan/go.rice/rice
	go get ./...

windows:
	GOOS=windows GOARCH=386 go get ./...
	GOOS=windows GOARCH=386 go build -v -ldflags="-w -X github.com/dh1tw/remoteSwitch/cmd.commitHash=${COMMIT} \
		-X github.com/dh1tw/remoteSwitch/cmd.version=${VERSION}"


# static: vet lint
# 	go build -i -v -o ${OUT}-v${VERSION} -tags netgo -ldflags="-extldflags \"-static\" -w -s -X main.version=${VERSION}" ${PKG}

clean:
	-@rm remoteSwitch remoteSwitch-v*

.PHONY: build server install vet lint clean install-deps generate