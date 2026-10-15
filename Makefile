GO ?= go
GOLANGCI_LINT ?= golangci-lint

.PHONY: lint test test-integration

lint:
	@test -z "$$(gofmt -l .)" || { gofmt -l .; exit 1; }
	$(GO) vet ./...
	$(GOLANGCI_LINT) run ./...

test:
	$(GO) test ./...

test-integration:
	@set -e; \
		trap 'docker compose -p family-expense-test -f docker-compose.test.yml down' EXIT; \
		docker compose -p family-expense-test -f docker-compose.test.yml up -d --wait; \
		TEST_DATABASE_URL='postgres://family_expense:local-test-only@127.0.0.1:55432/family_expense_test?sslmode=disable' \
			$(GO) test -count=1 ./internal/adapters/postgres
