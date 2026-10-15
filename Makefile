GO ?= go
GOLANGCI_LINT ?= golangci-lint

.PHONY: lint test

lint:
	@test -z "$$(gofmt -l .)" || { gofmt -l .; exit 1; }
	$(GO) vet ./...
	$(GOLANGCI_LINT) run ./...

test:
	$(GO) test ./...
