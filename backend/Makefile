.PHONY: build run test clean

BINARY_NAME=nightfall

build:
	go build -o $(BINARY_NAME) ./cmd/main.go

run:
	go run ./cmd/main.go

test:
	go test -race -v ./...

clean:
	go clean
	rm -f $(BINARY_NAME)