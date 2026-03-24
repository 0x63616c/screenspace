# ScreenSpace

macOS menu bar app for live video wallpapers with a community server for sharing.

## Architecture

- **App:** Swift 6 / SwiftUI, macOS 15+ (targeting macOS 26+ in future)
- **Server:** Go 1.26, PostgreSQL, S3-compatible storage
- **Distribution:** Direct (Sparkle auto-update), not Mac App Store
- **Dev environment:** Tilt (docker-compose for Postgres + MinIO + server)

## Code Style

### Go
- chi v5 for routing with route groups (public, auth, admin)
- pgx/v5 + pgxpool for PostgreSQL (not lib/pq)
- sqlc generates the repository layer from `db/queries/*.sql`
- oapi-codegen generates server interface + types from `api/openapi.yaml`
- Service layer between handlers and repositories for business logic
- Handlers return `error` via adapter pattern, never write HTTP errors directly
- `respond.JSON()`, `respond.Error()`, `respond.Paginated()` for all responses
- `slog` everywhere, never `log`. Enforced by depguard.
- Typed string constants for statuses, roles, categories, sort orders (never magic strings)
- `errors.Is`/`errors.As` for all error comparisons, never `==` or string matching
- All config from env vars, centralized in `internal/config/config.go`
- golangci-lint with errorlint, sloglint, bodyclose, depguard, gosec, exhaustive

### Swift
- `@Observable` everywhere (not ObservableObject/Combine)
- Platform protocol layer: all OS interactions behind protocols (WallpaperProviding, FileSystemProviding, KeychainProviding, etc.) with Live + Mock implementations
- ViewModels for every feature view, testable without UI
- Enums with raw values for all statuses, roles, categories (compiler-enforced exhaustive switches)
- `Sendable` conformance on all value types
- async/await over Combine, `Task { @MainActor in }` over `DispatchQueue.main.async`
- SwiftFormat (Lockwood) for formatting, SwiftLint for linting
- Swift Testing (`@Test`/`@Suite`) for unit tests, swift-snapshot-testing for visual regression
- EventLog (JSONL) for structured app event logging

### Both
- No magic strings. Typed constants/enums for everything.
- Single source of truth. OpenAPI spec is the API contract. Code is the documentation. No comments that duplicate what code defines.
- No lying comments. Don't write "10 endpoints" in a comment if the code defines 11.
- Lefthook for pre-commit hooks (goimports + golangci-lint for Go, SwiftFormat + SwiftLint for Swift)

## Project Structure

### Server (`server/`)
- `api/openapi.yaml` - API spec (source of truth)
- `cmd/server/main.go` - wiring only
- `internal/config/` - centralized config from env vars
- `internal/handler/` - thin HTTP adapters (parse request, call service, return)
- `internal/service/` - business logic
- `internal/respond/` - JSON responses, errors, pagination
- `internal/middleware/` - auth, admin, rate limiting, logging, security headers
- `internal/storage/` - S3 interface + implementation
- `internal/types/` - typed enums
- `db/migrations/` - SQL migrations
- `db/queries/` - sqlc query files
- `db/generated/` - sqlc output (never hand-edit)
- `generated/` - oapi-codegen output (never hand-edit)

### App (`app/`)
- `Sources/ScreenSpace/App/` - entry point, AppState, DI
- `Sources/ScreenSpace/Features/{Feature}/` - View + ViewModel per feature
- `Sources/ScreenSpace/Platform/Protocols/` - OS abstraction interfaces
- `Sources/ScreenSpace/Platform/Live/` - real OS implementations
- `Sources/ScreenSpace/Platform/Mock/` - test/preview implementations
- `Sources/ScreenSpace/Engine/` - wallpaper engine, pause controller
- `Sources/ScreenSpace/Core/` - API client, config, cache, event log, types
- `Sources/ScreenSpace/UI/Components/` - shared UI components
- `Sources/ScreenSpace/UI/Design/` - spacing, typography, button styles

## Testing

- Go: Ginkgo + Gomega (BDD), service tests with mock repos, handler tests with mock services, integration tests with real DB in CI
- Swift: Swift Testing for unit tests on ViewModels, swift-snapshot-testing for visual regression, mock platform implementations for all OS interactions
- Always execute test strategies. "It compiles" is not tested.

## Dev Commands

### Server
```bash
make build          # build binary
make run            # build and run
make test           # go test -race ./...
make lint           # golangci-lint run
make generate       # sqlc generate + oapi-codegen
make migrate/up     # run migrations
make audit          # all quality checks
```

## Conventions

- Conventional commits (feat:, fix:, chore:, etc.)
- Commit often, small focused changes, push after every commit
- Standardized pagination: `{"items":[...],"total":42,"limit":20,"offset":0}`
- Standardized errors: `{"error":{"code":"not_found","message":"wallpaper not found"}}`
- Button hierarchy: primary (.borderedProminent), secondary (.bordered), destructive (.bordered + red), plain (.plain)
- Typography tokens: pageTitle, sectionTitle, cardTitle, cardMeta, body, meta, label
- Spacing tokens: xs(4), sm(8), md(12), lg(16), xl(24), xxl(32)

## Security

- JWT auth with Bearer tokens, validated per-request (including banned check)
- bcrypt for passwords, minimum 8 characters
- Rate limiting on all endpoint groups
- Request body size limits (1MB for JSON)
- HTTP server timeouts (read 15s, write 30s, idle 120s)
- Security headers (X-Content-Type-Options, X-Frame-Options, CSP)
- Presigned upload URLs scoped with content-length and content-type
- Keychain with kSecAttrAccessibleWhenUnlockedThisDeviceOnly
- Never read .env, .pem, .key, credentials files

## Specs & Plans

- Refactor spec: `docs/superpowers/specs/2026-03-23-refactor-design.md`
- Full audit: `docs/superpowers/specs/2026-03-23-full-audit.md`
- Test findings: `docs/testing/2026-03-23-findings.md`
