.PHONY: all build test clean web-install web-build lint

BINARY_NAME=cassettecraft
VERSION?=1.0.0
COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT)"

all: build

web-install:
	cd web && npm install

web-build:
	cd web && npm run build

build: web-build
	go build $(LDFLAGS) -o bin/$(BINARY_NAME) ./cmd/cassettecraft

build-go-only:
	go build $(LDFLAGS) -o bin/$(BINARY_NAME) ./cmd/cassettecraft

test:
	go test -v -race ./...

lint:
	go vet ./...

clean:
	rm -rf bin/ web/dist/
