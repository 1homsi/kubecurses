BINARY    := kubecurses
MODULE    := github.com/1homsi/kubecurses

VERSION   ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT    ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
BUILDDATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS = -s -w \
	-X main.version=$(VERSION) \
	-X main.commit=$(COMMIT) \
	-X 'main.buildDate=$(BUILDDATE)'

.PHONY: all build run tidy lint clean

all: build

## build: compile the binary with version info
build:
	go build -ldflags="$(LDFLAGS)" -o $(BINARY) .

## run: build and run (requires a valid kubeconfig)
run: build
	./$(BINARY)

## tidy: tidy and vendor dependencies
tidy:
	go mod tidy

## lint: run golangci-lint
lint:
	golangci-lint run ./...

## clean: remove compiled binary
clean:
	rm -f $(BINARY)
