.PHONY: build test lint clean install bench coverage

BINARY=filehound
VERSION?=$(shell git describe --tags --always --dirty)
LDFLAGS=-ldflags "-s -w -X github.com/ripkitten-co/filehound/internal/version.Version=$(VERSION)"

build:
	go build $(LDFLAGS) -o bin/$(BINARY) .

test:
	go test -race -count=1 ./...

coverage:
	go test -race -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

lint:
	golangci-lint run ./...

clean:
	rm -rf bin/
	rm -f coverage.out coverage.html

install: build
	go install $(LDFLAGS) ./...

bench:
	go test -bench=. -benchmem ./...

all: lint test build

.DEFAULT_GOAL := build
