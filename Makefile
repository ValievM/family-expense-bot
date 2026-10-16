GO ?= go
GO_PACKAGES = ./cmd/... ./internal/... ./migrations/...
GOLANGCI_LINT ?= golangci-lint

.PHONY: lint test test-integration

lint:
	@test -z "$$(gofmt -l cmd internal migrations)" || { gofmt -l cmd internal migrations; exit 1; }
	$(GO) vet $(GO_PACKAGES)
	$(GOLANGCI_LINT) run $(GO_PACKAGES)

test:
	$(GO) test $(GO_PACKAGES)

test-integration:
	@set -e; \
		trap 'docker compose -p family-expense-test -f docker-compose.test.yml down' EXIT; \
		docker compose -p family-expense-test -f docker-compose.test.yml up -d --wait; \
		TEST_DATABASE_URL='postgres://family_expense:local-test-only@127.0.0.1:55432/family_expense_test?sslmode=disable' \
			$(GO) test -count=1 ./internal/adapters/postgres
