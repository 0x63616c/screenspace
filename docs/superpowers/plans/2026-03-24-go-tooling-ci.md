# Go Tooling & CI Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Wire up golangci-lint, Lefthook pre-commit hooks, updated CI workflows, and a production-ready Dockerfile so every commit is linted, every PR is verified, and the container build is correct.

**Architecture:** Track C is fully independent of Tracks A (Go server refactor) and B (Swift app refactor). The golangci-lint config targets the refactored server layout (chi, pgx, sqlc, oapi-codegen) but can be dropped in before or after the server rewrite -- the linter will simply flag existing violations. CI workflows gate on lint + vuln + generate-verify before running tests. Lefthook runs the same linters locally on staged files at commit time.

**Tech Stack:** golangci-lint v1.64+, Lefthook v1.x, govulncheck, sqlc, oapi-codegen, SwiftFormat, SwiftLint, GitHub Actions, Docker BuildKit.

**Spec reference:** `docs/superpowers/specs/2026-03-23-refactor-design.md` Sections 3.1, 3.2, 3.6, 3.7

---

## C1: golangci-lint Config

**Files:**
- `server/.golangci.yml` (Create)

### Tasks

- [ ] Create `server/.golangci.yml` with the full linter config below.

**Full file contents — `server/.golangci.yml`:**

```yaml
version: "2"

run:
  timeout: 5m
  modules-download-mode: readonly

linters:
  default: none
  enable:
    # Defaults (re-enabled explicitly)
    - errcheck
    - govet
    - staticcheck
    - unused
    - ineffassign
    - gosimple

    # Error handling
    - errorlint       # enforce errors.Is/errors.As (NOT wrapcheck: conflicts with handler adapter pattern)

    # Security
    - gosec

    # Performance
    - bodyclose
    - perfsprint

    # Style
    - revive
    - misspell
    - unconvert
    - nakedret

    # Complexity
    - cyclop
    - gocyclo

    # Modern Go
    - modernize
    - sloglint

    # Completeness
    - exhaustive
    - noctx

    # Import control
    - depguard

linters-settings:
  cyclop:
    max-complexity: 15

  gocyclo:
    min-complexity: 15

  nakedret:
    max-func-lines: 5

  exhaustive:
    default-signifies-exhaustive: false

  sloglint:
    no-mixed-args: true
    kv-only: true
    static-msg: true
    no-global: true

  revive:
    rules:
      - name: exported
      - name: var-naming
      - name: error-return
      - name: error-naming
      - name: if-return
      - name: increment-decrement
      - name: range
      - name: time-naming
      - name: unexported-return
      - name: blank-imports
      - name: context-as-argument
      - name: context-keys-type
      - name: dot-imports
      - name: empty-block

  depguard:
    rules:
      no-stdlib-log:
        deny:
          - pkg: "log"
            desc: "Use slog instead of the standard log package"
      no-lib-pq:
        deny:
          - pkg: "github.com/lib/pq"
            desc: "Use pgx/v5 instead of lib/pq"

  gosec:
    excludes:
      - G115  # integer overflow conversion: too noisy for HTTP status codes

issues:
  max-issues-per-linter: 0
  max-same-issues: 0
  exclude-rules:
    # Test files: relax some rules
    - path: "_test\\.go"
      linters:
        - gosec
        - errcheck
```

- [ ] Verify lint runs cleanly in the server directory:

```bash
cd server && golangci-lint run --config .golangci.yml
# Expected: exits 0 OR lists violations to fix (do not suppress, fix them)
```

- [ ] Commit: `chore(server): add golangci-lint config`

---

## C2: Lefthook Config

**Files:**
- `lefthook.yml` (Create at project root)

### Tasks

- [ ] Verify `lefthook` is installed (`lefthook --version`). If not, install: `brew install lefthook`.
- [ ] Verify `goimports` is installed (`goimports -h`). If not, install: `go install golang.org/x/tools/cmd/goimports@latest`.

- [ ] Create `lefthook.yml` at the project root with the full config below.

**Full file contents — `lefthook.yml`:**

```yaml
# lefthook.yml
# Pre-commit hooks for Go (server/) and Swift (app/)
# Install: lefthook install

pre-commit:
  parallel: true
  commands:
    go-imports:
      root: server/
      glob: "**/*.go"
      run: goimports -w {staged_files} && git add {staged_files}

    go-lint:
      root: server/
      glob: "**/*.go"
      run: golangci-lint run --config .golangci.yml

    swift-format:
      root: app/
      glob: "**/*.swift"
      run: swiftformat --config .swiftformat {staged_files}
      stage_fixed: true

    swift-lint:
      root: app/
      glob: "**/*.swift"
      run: swiftlint lint --strict {staged_files}
```

- [ ] Run `lefthook install` at the project root to wire up the git hooks:

```bash
cd /path/to/screenspace && lefthook install
# Expected: Lefthook v... installed
```

- [ ] Smoke-test by staging a `.go` file with a formatting issue and attempting a commit. Confirm goimports runs and the lint hook fires.
- [ ] Commit: `chore: add lefthook pre-commit hooks for Go and Swift`

---

## C3: CI Workflow Updates

**Files:**
- `.github/workflows/server.yml` (Modify)
- `.github/workflows/app.yml` (Modify)

### C3a: server.yml

**Current state:** Single `test` job with Go 1.22, no lint step, no vuln step, no generate-verify step. Docker build job on main.

**Target state:** Go 1.26, four steps in the test job: lint, vuln, generate-verify, test with race detector. Coverage threshold kept at 95%.

- [ ] Replace `.github/workflows/server.yml` with the full contents below.

**Full file contents — `.github/workflows/server.yml`:**

```yaml
name: Server CI

on:
  push:
    paths: ["server/**", ".github/workflows/server.yml"]
  pull_request:
    paths: ["server/**", ".github/workflows/server.yml"]

jobs:
  lint:
    name: Lint
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: "1.26"
          cache: true
          cache-dependency-path: server/go.sum

      - name: golangci-lint
        uses: golangci/golangci-lint-action@v6
        with:
          version: v1.64
          working-directory: server
          args: --config .golangci.yml --timeout 5m

  vuln:
    name: Vulnerability Scan
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: "1.26"
          cache: true
          cache-dependency-path: server/go.sum

      - name: Install govulncheck
        run: go install golang.org/x/vuln/cmd/govulncheck@latest

      - name: govulncheck
        working-directory: server
        run: govulncheck ./...
        # Expected: exits 0 if no known vulnerabilities

  generate-verify:
    name: Verify Generated Code
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: "1.26"
          cache: true
          cache-dependency-path: server/go.sum

      - name: Install sqlc
        run: go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest

      - name: Install oapi-codegen
        run: go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest

      - name: Run generate
        working-directory: server
        run: make generate

      - name: Check for diff
        run: |
          if ! git diff --quiet; then
            echo "Generated code is out of date. Run 'make generate' and commit the result."
            git diff --stat
            exit 1
          fi
        # Expected: no diff. If diff exists, the PR is missing regenerated files.

  test:
    name: Test
    runs-on: ubuntu-latest
    needs: [lint, vuln, generate-verify]
    services:
      postgres:
        image: postgres:16
        env:
          POSTGRES_DB: screenspace_test
          POSTGRES_USER: test
          POSTGRES_PASSWORD: test
        ports: ["5432:5432"]
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5

      minio:
        image: minio/minio
        env:
          MINIO_ROOT_USER: minioadmin
          MINIO_ROOT_PASSWORD: minioadmin
        ports: ["9000:9000"]
        options: --entrypoint "minio server /data"

    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: "1.26"
          cache: true
          cache-dependency-path: server/go.sum

      - name: Run tests with race detector
        working-directory: server
        env:
          DATABASE_URL: postgres://test:test@localhost:5432/screenspace_test?sslmode=disable
          S3_ENDPOINT: http://localhost:9000
          S3_ACCESS_KEY: minioadmin
          S3_SECRET_KEY: minioadmin
          S3_BUCKET: screenspace-test
          JWT_SECRET: test-secret-minimum-32-characters-long
        run: go test -race -coverprofile=coverage.out ./...
        # Expected: all tests pass, -race finds no data races

      - name: Check coverage threshold
        working-directory: server
        run: |
          COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
          echo "Coverage: $COVERAGE%"
          if (( $(echo "$COVERAGE < 95" | bc -l) )); then
            echo "FAIL: coverage $COVERAGE% is below 95% threshold"
            exit 1
          fi

      - name: Upload coverage report
        uses: actions/upload-artifact@v4
        if: always()
        with:
          name: coverage-report
          path: server/coverage.out

  docker:
    name: Docker Build
    needs: test
    if: github.ref == 'refs/heads/main'
    runs-on: ubuntu-latest
    permissions:
      packages: write
    steps:
      - uses: actions/checkout@v4

      - uses: docker/login-action@v3
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}

      - uses: docker/build-push-action@v5
        with:
          context: server
          push: true
          tags: ghcr.io/0x63616c/screenspace-server:latest
```

- [ ] Commit: `ci(server): update to Go 1.26, add lint/vuln/generate-verify steps`

### C3b: app.yml

**Current state:** `build` job with `swift build` + `swift test`, no lint or format check. Release job on version tags.

**Target state:** Add `lint` job that runs SwiftLint and SwiftFormat checks before the build. `build` job depends on `lint`.

- [ ] Replace `.github/workflows/app.yml` with the full contents below.

**Full file contents — `.github/workflows/app.yml`:**

```yaml
name: macOS App CI

on:
  push:
    paths: ["app/**", ".github/workflows/app.yml"]
    tags: ["v*"]
  pull_request:
    paths: ["app/**", ".github/workflows/app.yml"]

jobs:
  lint:
    name: Lint & Format Check
    runs-on: macos-15
    steps:
      - uses: actions/checkout@v4

      - name: Install SwiftLint
        run: brew install swiftlint

      - name: Install SwiftFormat
        run: brew install swiftformat

      - name: SwiftLint (strict)
        working-directory: app
        run: swiftlint lint --strict
        # Expected: exits 0. Any warning or error is a failure in strict mode.

      - name: SwiftFormat (lint check, no write)
        working-directory: app
        run: swiftformat --lint .
        # Expected: exits 0. Any formatting violation is a failure.

  build:
    name: Build & Test
    runs-on: macos-15
    needs: lint
    steps:
      - uses: actions/checkout@v4

      - name: Build
        working-directory: app
        run: swift build

      - name: Run tests
        working-directory: app
        run: swift test

  release:
    name: Release
    needs: build
    if: startsWith(github.ref, 'refs/tags/v')
    runs-on: macos-15
    steps:
      - uses: actions/checkout@v4

      - name: Build for release
        working-directory: app
        run: swift build -c release

      - name: Upload release asset
        uses: softprops/action-gh-release@v1
        with:
          files: app/.build/release/ScreenSpace
```

- [ ] Commit: `ci(app): add swiftlint and swiftformat check steps`

---

## C4: Dockerfile Updates

**Files:**
- `server/Dockerfile` (Modify)

### Tasks

**Current state:** The Dockerfile already uses `golang:1.26-alpine` and `alpine:3.21`. It installs `ffmpeg`. The only gap is the build command targets `.` (root package) instead of `./cmd/server` (correct after the server refactor moves main to `cmd/server/`), and there is no health check.

- [ ] Update `server/Dockerfile` to the target state below.

**Full file contents — `server/Dockerfile`:**

```dockerfile
FROM golang:1.26-alpine AS builder
WORKDIR /app

# Cache dependency download layer separately from source
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o server ./cmd/server

FROM alpine:3.21
RUN apk add --no-cache ffmpeg ca-certificates tzdata

COPY --from=builder /app/server /usr/local/bin/server

# Non-root user for runtime
RUN addgroup -S screenspace && adduser -S screenspace -G screenspace
USER screenspace

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD wget -qO- http://localhost:8080/api/v1/health || exit 1

ENTRYPOINT ["server"]
```

**Changes from current state:**
- Build target: `./cmd/server` (was `.` -- correct path after main moves to `cmd/server/`)
- Added `-ldflags="-w -s"` to strip debug info, reduces binary size
- Added `ca-certificates` and `tzdata` (needed for HTTPS calls and time zone handling)
- Added non-root user `screenspace` for runtime security (gosec G204 compliance)
- Added `EXPOSE 8080` for documentation and container tooling
- Added `HEALTHCHECK` that pings the deep health endpoint

- [ ] Build and verify the image locally:

```bash
cd server && docker build -t screenspace-server:local .
# Expected: build succeeds, two-stage build, final image ~50-80MB

docker run --rm screenspace-server:local server --help 2>/dev/null || true
# Expected: binary runs (will fail without env vars, that's fine)

docker image inspect screenspace-server:local --format '{{.Config.User}}'
# Expected: screenspace
```

- [ ] Commit: `chore(server): harden Dockerfile with non-root user, health check, stripped binary`

---

## Verification Checklist

Run this after all four tasks are complete to confirm the full Track C is wired correctly.

- [ ] `cd server && golangci-lint run --config .golangci.yml` -- exits 0 (or known violations listed, not suppressed)
- [ ] `lefthook run pre-commit` at project root -- both go-imports and go-lint hooks fire
- [ ] `docker build -t screenspace-server:local server/` -- succeeds
- [ ] `docker image inspect screenspace-server:local --format '{{.Config.Healthcheck}}'` -- non-empty
- [ ] Push a branch with a staged `.go` file -- CI server.yml lint job runs and passes
- [ ] Push a branch with a staged `.swift` file -- CI app.yml lint job runs SwiftLint + SwiftFormat
