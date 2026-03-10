.PHONY: build run test clean lint fmt check

BINARY_NAME=nightfall

build:
	go build -o $(BINARY_NAME) ./cmd/main.go

run:
	go run ./cmd/main.go

run-otel:
	ENABLE_OTEL=true go run ./cmd/main.go

test:
	go test -race ./...

testv:
	go test -race -v ./...

clean:
	go clean
	rm -f $(BINARY_NAME)

lint:
	golangci-lint run ./...

fmt:
	goimports -w -local github.com/gabrielrnascimento/nightfall .

check: fmt lint