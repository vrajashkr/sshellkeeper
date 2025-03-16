.PHONY: all build test lint run test-race-coverage format

all: build test

build:
	go build -race

test:
	go test -v -failfast ./...

test-race-coverage:
	go test -race -v -cover -coverprofile=coverage.out ./...

format:
	gofumpt -l -w .

lint:
	golangci-lint run

run:
	go run .
