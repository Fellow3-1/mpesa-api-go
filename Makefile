GO ?= go

.PHONY: all check fmt vet test cover build demo clean

all: check

fmt:
	$(GO) fmt ./...

vet:
	$(GO) vet ./...

test:
	$(GO) test -race ./...

cover:
	$(GO) test -race -coverprofile=coverage.out ./...
	$(GO) tool cover -func=coverage.out | tail -n 1

check: fmt vet test

build:
	$(GO) build ./...

demo:
	$(GO) run ./demo

clean:
	rm -f coverage.out coverage.html demo/demo
