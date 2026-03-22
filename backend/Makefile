BIN        := bin/allure-hub
ENTRYPOINT := ./cmd/server

VERSION    ?= dev
BUILD_TIME  = $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
GO_VERSION  = $(shell go version | awk '{print $$3}')
PKG         = github.com/tlmanz/allure-hub/internal/transport/handler

LDFLAGS = -X $(PKG).Version=$(VERSION) \
          -X '$(PKG).BuildTime=$(BUILD_TIME)' \
          -X '$(PKG).GoVersion=$(GO_VERSION)'

-include .env
export

.PHONY: build test lint tidy run clean

build:
	go build -trimpath -ldflags "$(LDFLAGS)" -o $(BIN) $(ENTRYPOINT)

test:
	go test ./...

lint:
	go vet ./...

tidy:
	go mod tidy

run:
	mkdir -p ./data
	go run -ldflags "$(LDFLAGS)" $(ENTRYPOINT)

clean:
	rm -rf bin/
