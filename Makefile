ifndef ($(GOPATH))
	GOPATH = $(HOME)/go
endif

APP_NAME=rancher-service-up

PATH := $(GOPATH)/bin:$(PATH)
VERSION = $(shell git describe --tags --always --dirty)
BRANCH = $(shell git rev-parse --abbrev-ref HEAD)
REVISION = $(shell git rev-parse HEAD)
REVSHORT = $(shell git rev-parse --short HEAD)
USER = $(shell whoami)

ifneq ($(OS), Windows_NT)
	CURRENT_PLATFORM = linux

	# If on macOS, set the shell to bash explicitly
	ifeq ($(shell uname), Darwin)
		SHELL := /bin/bash
		CURRENT_PLATFORM = darwin
	endif

	# To populate version metadata, we use unix tools to get certain data
	GOVERSION = $(shell go version | awk '{print $$3}')
	NOW	= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
else
	CURRENT_PLATFORM = windows
	# To populate version metadata, we use windows tools to get the certain data
	GOVERSION_CMD = "(go version).Split()[2]"
	GOVERSION = $(shell powershell $(GOVERSION_CMD))
	NOW	= $(shell powershell Get-Date -format s)
endif

all: build-all
.PHONY: build

clean:
	rm -rf build
	
deps:
	go get -u github.com/golang/dep/cmd/dep
	dep ensure -vendor-only

.pre-build:
	mkdir -p build/darwin
	mkdir -p build/linux

build: .pre-build
	go build -i -o build/$(APP_NAME)

build-all: .pre-build
	GOOS=darwin CGO_ENABLED=0 go build -i -o build/darwin/$(APP_NAME)
	GOOS=linux CGO_ENABLED=0 go build -i -o build/linux/$(APP_NAME)

test:
	go test -cover -race -v ./...