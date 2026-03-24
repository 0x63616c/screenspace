BINARY      := screenspace-server
BUILD_DIR   := bin
MODULE      := github.com/0x63616c/screenspace/server
SQLC        := go run github.com/sqlc-dev/sqlc/cmd/sqlc
OAPI        := go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen
MIGRATE_URL ?= $(DATABASE_URL)

.PHONY: build run test lint generate migrate/up migrate/down audit clean

## build: compile the server binary
build:
	go build -o $(BUILD_DIR)/$(BINARY) .

## run: build and run the server
run: build
	./$(BUILD_DIR)/$(BINARY)

## test: run all tests with race detector
test:
	go test -race ./...

## lint: run golangci-lint
lint:
	golangci-lint run ./...

## generate: run sqlc and oapi-codegen to regenerate all generated code
generate: generate/sqlc generate/oapi

generate/sqlc:
	$(SQLC) generate -f db/sqlc.yaml

generate/oapi:
	$(OAPI) --config api/cfg.yaml api/openapi.yaml

## migrate/up: run pending database migrations
migrate/up:
	@test -n "$(MIGRATE_URL)" || (echo "MIGRATE_URL or DATABASE_URL must be set" && exit 1)
	go run -tags 'pgx5' github.com/golang-migrate/migrate/v4/cmd/migrate \
		-path db/migrations \
		-database "$(MIGRATE_URL)" \
		up

## migrate/down: roll back the last migration
migrate/down:
	@test -n "$(MIGRATE_URL)" || (echo "MIGRATE_URL or DATABASE_URL must be set" && exit 1)
	go run -tags 'pgx5' github.com/golang-migrate/migrate/v4/cmd/migrate \
		-path db/migrations \
		-database "$(MIGRATE_URL)" \
		down 1

## audit: run all quality checks
audit: lint test
	go vet ./...
	go mod tidy
	@git diff --exit-code go.mod go.sum || (echo "go.mod/go.sum have uncommitted changes after tidy" && exit 1)
	$(SQLC) vet -f db/sqlc.yaml
	$(MAKE) generate
	@git diff --exit-code db/generated/ generated/ || (echo "generated files are out of date — run make generate" && exit 1)

## clean: remove build artifacts
clean:
	rm -rf $(BUILD_DIR)

## help: list available targets
help:
	@grep -E '^## ' Makefile | sed 's/## //'
