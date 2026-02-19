BINARY  := kubecurses
MODULE  := github.com/1homsi/kubecurses

.PHONY: all build run tidy lint clean

all: build

## build: compile the binary
build:
	go build -o $(BINARY) ./...

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
