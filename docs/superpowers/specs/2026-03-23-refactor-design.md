# ScreenSpace Refactor & Hardening Spec

**Date:** 2026-03-23
**Updated:** 2026-03-24 (post-review)
**Scope:** Full-stack refactor of server (Go) and app (Swift) for production-grade quality, testability, security, and vibe-coding ergonomics.

---

## Toolchain

- **Go:** 1.26 (managed via mise, `mise.toml` in project root)
- **Swift:** 6.0+ (swift-tools-version: 6.0, Xcode 26+)
- **Generated code** (sqlc output in `db/generated/`, oapi-codegen output in `generated/`) is committed to git. CI verifies `make generate` produces no diff.

### Go 1.24-1.26 Idioms (mandatory)

**From 1.24:**
- `omitzero` over `omitempty` in all JSON struct tags. `omitempty` is broken for `time.Time` and zero-value numerics.
- `t.Context()` in all tests instead of `context.Background()`.
- `slog.DiscardHandler` in test setup for no-op logging.

**From 1.25:**
- `sync.WaitGroup.Go()` instead of manual Add(1)/go/Done() patterns.
- `http.CrossOriginProtection` for CSRF if/when browser-based clients are added. Not needed for native app API-only usage (no cookies, no forms). Add to security headers middleware as a no-op placeholder with a comment explaining when to enable.
- `runtime/trace.FlightRecorder` for production ring-buffer tracing (replaces pprof for intermittent debugging).
- `testing/synctest` for deterministic concurrent test patterns.

**From 1.26:**
- `errors.AsType[T]()` instead of declaring a target variable for `errors.As()`. Cleaner, type-safe error handling.
- `slog.NewMultiHandler()` for fanning logs to multiple destinations.
- Green Tea GC is now default (10-40% less GC overhead, no action needed).
- `os/signal.NotifyContext()` now reports which signal cancelled (better graceful shutdown logging).
- `io.ReadAll()` is ~2x faster with ~50% less allocation (benefits request body reading).

---

## Principles

1. **Single source of truth.** OpenAPI spec defines the API contract. Enums define categories/statuses/roles. Code is the documentation. No comments that duplicate what code defines. No counts, no lists that can drift.
2. **Compile-time safety over runtime checks.** Typed enums, exhaustive switches, generated code, interface contracts. If it can break, the compiler should catch it.
3. **Testable without the world.** Every OS interaction behind a protocol. Every service behind an interface. Tests run with mocks, no database, no S3, no screen, no network.
4. **Small files, one concern.** AI context windows work best with focused files. One responsibility per file. Features grouped by screen.
5. **No magic strings.** Every status, role, category, sort order, and format is a typed constant or enum.
6. **No lying comments.** If the code defines 11 endpoints, don't write "10 endpoints" anywhere. The code IS the documentation.

---

## Phase 1: Go Server Architecture

### 1.1 Router: chi v5

Replace `net/http.ServeMux` with chi. Route groups for public, authenticated, and admin.

**What changes:**
- `main.go` rewritten with chi route groups
- Middleware composed per-group (not per-route wrapping)
- `chi.URLParam(r, "id")` replaces `r.PathValue("id")` in all handlers
- Built-in middleware added: Logger, Recoverer, RequestID, RealIP, Timeout, CleanPath

**Route structure:**
```
/api/v1
  Public (ipLimiter: 120/min per IP)
    GET  /health
    GET  /categories          (not paginated, returns flat array of all categories)
    GET  /wallpapers
    GET  /wallpapers/popular
    GET  /wallpapers/recent
    GET  /wallpapers/{id}

  Auth (authLimiter: 10/min per IP)
    POST /auth/register
    POST /auth/login

  Authenticated (authMw + userLimiter: 30/min per user)
    GET  /auth/me
    POST /wallpapers/{id}/download  (downloadLimiter: 60/hour per user)
    POST /wallpapers/{id}/favorite
    POST /wallpapers/{id}/report
    POST /wallpapers (uploadLimiter: 5/day)
    POST /wallpapers/{id}/finalize

  Admin (authMw + adminMw)
    GET  /admin/queue
    POST /admin/queue/{id}/approve
    POST /admin/queue/{id}/reject
    GET  /admin/wallpapers
    PATCH /admin/wallpapers/{id}
    GET  /admin/users
    POST /admin/users/{id}/ban
    POST /admin/users/{id}/unban
    POST /admin/users/{id}/promote
    GET  /admin/reports
    POST /admin/reports/{id}/dismiss
```

Delete `requireAdmin()` function. Admin middleware handles it.

### 1.2 Database: pgx/v5 + pgxpool

Replace `lib/pq` + `database/sql` with `pgx/v5`.

**Migration strategy:** Atomic swap in a single commit. lib/pq and pgx cannot safely coexist (different driver registration, different error types). One commit replaces all `database/sql` calls with pgxpool, removes `lib/pq` from go.mod, runs `go mod tidy`. All repository code must be rewritten to sqlc simultaneously (Phase 1.2 + 1.3 are one unit of work).

**What changes:**
- `sql.Open("postgres", ...)` becomes `pgxpool.New(ctx, connString)`
- `pq.Array()` becomes native pgx array support
- Connection pool configured with explicit limits from config
- `pool.Ping(ctx)` on startup
- `golang-migrate` switches to pgx-compatible driver
- `depguard` rule blocks `"github.com/lib/pq"` import after migration

**Pool config (from env):**
- `DB_MAX_CONNS` (default 25)
- `DB_MIN_CONNS` (default 5)
- `DB_MAX_CONN_LIFETIME` (default 5m)
- `DB_HEALTH_CHECK_PERIOD` (default 30s)

### 1.3 Repository: sqlc

Replace hand-written repositories with sqlc-generated code.

**Configuration (`db/sqlc.yaml`):**
```yaml
version: "2"
sql:
  - engine: "postgresql"
    queries: "queries/"
    schema: "migrations/"
    gen:
      go:
        package: "db"
        out: "generated"
        sql_package: "pgx/v5"
        emit_interface: true
```

**Query files:**
- `db/queries/user.sql` - CreateUser, GetUserByID, GetUserByEmail, ListUsers, CountUsers, ListUsersWithSearch, SetBanned, SetRole
- `db/queries/wallpaper.sql` - CreateWallpaper, GetWallpaperByID, ListWallpapers (with sqlc.narg for optional filters), CountWallpapers, UpdateStatus, UpdateMetadata, UpdateAfterFinalize, IncrementDownloadCount, DeleteWallpaper
- `db/queries/favorite.sql` - CheckFavorite, InsertFavorite, DeleteFavorite, ListFavoritesByUser, CountFavoritesByUser
- `db/queries/report.sql` - CreateReport, ListPendingReports, CountPendingReports, DismissReport

Use separate queries per sort order instead of dynamic CASE WHEN (cleaner, avoids NULL ordering ambiguity):
```sql
-- name: ListWallpapersRecent :many
SELECT ... FROM wallpapers
WHERE status = sqlc.arg('status')
  AND (sqlc.narg('category')::text IS NULL OR category ILIKE sqlc.narg('category'))
  AND (sqlc.narg('query')::text IS NULL OR title ILIKE sqlc.narg('query'))
ORDER BY created_at DESC
LIMIT sqlc.arg('lim') OFFSET sqlc.arg('off');

-- name: ListWallpapersPopular :many
SELECT ... FROM wallpapers
WHERE status = sqlc.arg('status')
  AND (sqlc.narg('category')::text IS NULL OR category ILIKE sqlc.narg('category'))
  AND (sqlc.narg('query')::text IS NULL OR title ILIKE sqlc.narg('query'))
ORDER BY download_count DESC, created_at DESC
LIMIT sqlc.arg('lim') OFFSET sqlc.arg('off');
```

### 1.4 API Contract: oapi-codegen

Single OpenAPI 3.0 spec generates server interface, request/response types, and request validation middleware.

**File:** `api/openapi.yaml`

**Generated output:**
- `generated/server.gen.go` - chi server interface (compile error if handler missing)
- `generated/types.gen.go` - request/response structs with validation tags
- Request validation middleware (title maxLength, tag maxItems, category enum, etc.)

**What this replaces:**
- All hand-written request/response structs in handlers
- All inline validation (title length, tag count, category whitelist, reason length)
- `APIModels.swift` on the app side (generate Swift client from same spec)

**oapi-codegen config:**
```yaml
package: generated
output: generated/server.gen.go
generate:
  chi-server: true
  models: true
```

> **Note:** Do NOT use `strict-server: true`. Strict mode generates typed response unions that conflict with the handler adapter pattern (1.6) where handlers return `error` and use `respond.JSON()`. Regular `chi-server` mode generates the route interface while allowing custom response handling.

**Swift client generation:**

Use [swift-openapi-generator](https://github.com/apple/swift-openapi-generator) to generate Swift API types from the same `api/openapi.yaml`. This replaces `APIModels.swift`. Add as a SwiftPM build plugin in `Package.swift`.

### 1.5 Service Layer

New layer between handlers and repositories for business logic.

**Files:**
- `internal/service/wallpaper.go` - create, finalize, download, delete orchestration
- `internal/service/auth.go` - existing, moves here
- `internal/service/video.go` - existing, moves here
- `internal/service/favorite.go` - toggle logic (transaction)
- `internal/service/report.go` - create with validation

**What moves out of handlers:**
- All validation logic (now partially handled by oapi-codegen, remainder in services)
- S3 orchestration (presigned URLs, upload/download coordination)
- Business rules (ownership checks, status transitions)
- The entire Finalize flow (see below)

**Finalize service flow** (`WallpaperService.Finalize(ctx, wallpaperID, userID)`):
1. Get wallpaper by ID. Return `ErrNotFound` if missing.
2. Verify `uploaderID == userID`. Return `ErrForbidden` if not owner.
3. Verify status is `pending`. Return `ErrBadRequest` if not (idempotent re-finalize not allowed).
4. Download original video from S3 (`wallpapers/{id}/original.mp4`) to temp file. Use `io.LimitReader(reader, cfg.MaxFileSize+1)` to cap download size.
5. Probe video via `VideoService.Probe()`. Returns width, height, duration, file size, codec.
6. Validate constraints from config:
   - `info.Size <= cfg.MaxFileSize` (default 200MB)
   - `info.Duration <= cfg.MaxDuration` (default 60s)
   - `info.Height >= cfg.MinHeight` (default 1080)
   - `info.Format` is "h264" or "h265"
   - Return `ErrBadRequest` with specific message for each violation.
7. Generate thumbnail via `VideoService.GenerateThumbnail()` -> temp file.
8. Generate preview clip via `VideoService.GeneratePreview()` -> temp file.
9. Upload thumbnail to S3 (`wallpapers/{id}/thumbnail.jpg`, content-type `image/jpeg`).
10. Upload preview to S3 (`wallpapers/{id}/preview.mp4`, content-type `video/mp4`).
11. Update DB: width, height, duration, file size, format, resolution string, thumbnail key, preview key, status = `pending_review`.
12. Clean up all temp files (deferred).
13. Return updated wallpaper.

**Error handling:** Steps 4-10 can fail independently. On S3 upload failure, the wallpaper stays in `pending` status and user can retry. Temp file cleanup must happen regardless (defer).

**Admin promotion** (`POST /admin/users/{id}/promote`):
- Sets `role = admin`. Idempotent (promoting an existing admin is a no-op).
- Cannot promote banned users (return `ErrBadRequest`).
- `ADMIN_EMAIL` auto-promotes at registration only (bootstrap). After that, admins use this endpoint.
- No demote endpoint. Add later if needed.

Handlers become:
```go
func (h *WallpaperHandler) Get(w http.ResponseWriter, r *http.Request) error {
    id := chi.URLParam(r, "id")
    wp, err := h.svc.GetApproved(r.Context(), id)
    if err != nil {
        return err
    }
    return respond.JSON(w, http.StatusOK, wp)
}
```

### 1.6 Handler Adapter + Respond Package

**Handler adapter** (`internal/handler/adapter.go`):
Handlers return `error`. Wrapper maps errors to HTTP responses.

```go
type HandlerFunc func(w http.ResponseWriter, r *http.Request) error

func Wrap(h HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        if err := h(w, r); err != nil {
            handleError(w, r, err)
        }
    }
}
```

**Sentinel errors:**
```go
var (
    ErrNotFound    = errors.New("not found")
    ErrForbidden   = errors.New("forbidden")
    ErrConflict    = errors.New("conflict")
    ErrBadRequest  = errors.New("bad request")
)
```

**AppError type:**
```go
type AppError struct {
    Status  int
    Code    string // machine-readable: "not_found", "rate_limited", "validation_failed"
    Message string // human-readable
    Err     error  // internal, never exposed
}
```

**Respond helpers:**
- `respond.JSON(w, status, data)` - sets Content-Type, encodes JSON
- `respond.Error(w, status, code, message)` - structured error response
- `respond.Paginated(w, items, total, limit, offset)` - standard pagination response

**Standardized pagination response:**
```json
{
  "items": [...],
  "total": 42,
  "limit": 20,
  "offset": 0
}
```

All paginated endpoints use the same format. No more `"wallpapers"`, `"users"`, `"reports"` as array keys.

**Pagination parsing:**
```go
type Pagination struct {
    Limit  int
    Offset int
}

func ParsePagination(q url.Values) Pagination {
    // defaults: limit=20, offset=0, max limit=100
}
```

### 1.7 Typed Constants

```go
type WallpaperStatus string
const (
    StatusPending       WallpaperStatus = "pending"
    StatusPendingReview WallpaperStatus = "pending_review"
    StatusApproved      WallpaperStatus = "approved"
    StatusRejected      WallpaperStatus = "rejected"
)
```

**Wallpaper status transitions (enforced in service layer):**
```
pending --> pending_review  (on finalize: video validated, thumbnail generated)
pending_review --> approved (admin approve)
pending_review --> rejected (admin reject)
rejected --> (terminal, no further transitions)
approved --> (terminal, no further transitions)
```

No other transitions are legal. Service methods must validate the current status before updating.

```go

type UserRole string
const (
    RoleUser  UserRole = "user"
    RoleAdmin UserRole = "admin"
)

type Category string
const (
    CategoryNature     Category = "nature"
    CategoryAbstract   Category = "abstract"
    CategoryUrban      Category = "urban"
    CategoryCinematic  Category = "cinematic"
    CategorySpace      Category = "space"
    CategoryUnderwater Category = "underwater"
    CategoryMinimal    Category = "minimal"
    CategoryOther      Category = "other"
)

type SortOrder string
const (
    SortRecent  SortOrder = "recent"
    SortPopular SortOrder = "popular"
)
```

Enforced by `exhaustive` linter on all switch statements.

### 1.8 Config Centralization

Single `Config` struct, all from env vars:

```go
type Config struct {
    // Server
    Port            string // PORT (default "8080")
    ShutdownTimeout time.Duration // SHUTDOWN_TIMEOUT (default 25s)
    LogLevel        string // LOG_LEVEL (default "info")

    // Database
    DatabaseURL        string // DATABASE_URL (required)
    DBMaxConns         int    // DB_MAX_CONNS (default 25)
    DBMinConns         int    // DB_MIN_CONNS (default 5)
    DBMaxConnLifetime  time.Duration // DB_MAX_CONN_LIFETIME (default 5m)
    DBHealthCheckPeriod time.Duration // DB_HEALTH_CHECK_PERIOD (default 30s)

    // S3
    S3Endpoint  string // S3_ENDPOINT (required)
    S3Bucket    string // S3_BUCKET (default "screenspace")
    S3AccessKey string // S3_ACCESS_KEY (required)
    S3SecretKey string // S3_SECRET_KEY (required)

    // Auth
    JWTSecret       string // JWT_SECRET (required, min 32 chars)
    JWTExpiry        time.Duration // JWT_EXPIRY (default 7d)
    AdminEmail      string // ADMIN_EMAIL
    BcryptCost      int    // BCRYPT_COST (default 10)
    MinPasswordLen  int    // MIN_PASSWORD_LENGTH (default 8)

    // Rate Limits
    AuthRateLimit    int // AUTH_RATE_LIMIT (default 10/min)
    PublicRateLimit  int // PUBLIC_RATE_LIMIT (default 120/min)
    UserRateLimit    int // USER_RATE_LIMIT (default 30/min)
    UploadRateLimit  int // UPLOAD_RATE_LIMIT (default 5/day)
    DownloadRateLimit int // DOWNLOAD_RATE_LIMIT (default 60/hour)

    // Upload Constraints
    MaxFileSize     int64   // MAX_FILE_SIZE (default 200MB)
    MaxDuration     float64 // MAX_DURATION (default 60s)
    MinHeight       int     // MIN_HEIGHT (default 1080)
    MaxTitleLength  int     // MAX_TITLE_LENGTH (default 255)
    MaxTagCount     int     // MAX_TAG_COUNT (default 10)
    MaxTagLength    int     // MAX_TAG_LENGTH (default 50)
    MaxReportLength int     // MAX_REPORT_LENGTH (default 500)

    // Pagination
    DefaultPageSize int // DEFAULT_PAGE_SIZE (default 20)
    MaxPageSize     int // MAX_PAGE_SIZE (default 100)

    // URLs
    PresignedDownloadExpiry time.Duration // PRESIGNED_DOWNLOAD_EXPIRY (default 1h)
    PresignedUploadExpiry   time.Duration // PRESIGNED_UPLOAD_EXPIRY (default 2h)
}
```

Validation at startup: required fields present, JWT secret >= 32 chars, numeric bounds sane.

### 1.9 Logging

Consistent `slog` everywhere. `depguard` blocks `"log"` import.

**Request logger middleware:**
```go
// Logs: method, path, status, duration, request_id, user_id (if auth'd)
slog.Info("request",
    "method", r.Method,
    "path", r.URL.Path,
    "status", status,
    "duration_ms", dur.Milliseconds(),
    "request_id", middleware.GetReqID(r.Context()),
    "user_id", userID,
)
```

**Log level** configurable via `LOG_LEVEL` env var.

### 1.10 HTTP Server Hardening

```go
srv := &http.Server{
    Addr:         ":" + cfg.Port,
    Handler:      router,
    ReadTimeout:  15 * time.Second,
    WriteTimeout: 30 * time.Second,
    IdleTimeout:  120 * time.Second,
}
```

**Graceful shutdown:**
```go
ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
defer stop()
go srv.ListenAndServe()
<-ctx.Done()
shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
defer cancel()
srv.Shutdown(shutdownCtx)
```

**Request body size limits:**
```go
r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1MB for JSON endpoints
```

**Security headers middleware:**
```go
w.Header().Set("X-Content-Type-Options", "nosniff")
w.Header().Set("X-Frame-Options", "DENY")
w.Header().Set("Content-Security-Policy", "default-src 'none'")
```

**Deep health check:**
```go
// GET /health
// Pings database, returns component status
{"status":"ok","db":"ok","uptime":"2h30m"}
```

### 1.11 Error Handling Fixes

- `errors.Is(err, migrate.ErrNoChange)` instead of `err != migrate.ErrNoChange`
- Postgres unique violation: check `pgx` error code `23505` via `errors.As` instead of `strings.Contains(err.Error(), "duplicate")`
- All storage methods wrap errors with context: `fmt.Errorf("put object %s: %w", key, err)`
- `json.NewEncoder(w).Encode()` errors logged (not critical but clean)
- `IncrementDownloadCount` error logged
- `Popular`/`Recent` handlers refactored to not mutate request URL

### 1.12 Debug Server

```go
// Internal-only debug server, bound to localhost only
go func() {
    debugMux := http.NewServeMux()
    debugMux.HandleFunc("/debug/pprof/", pprof.Index)
    debugMux.HandleFunc("/debug/pprof/profile", pprof.Profile)
    http.ListenAndServe("127.0.0.1:6060", debugMux)
}()
```

Enabled via `DEBUG_ENABLED=true` env var. Must bind to `127.0.0.1`, never `0.0.0.0`.

For intermittent production issues, prefer `runtime/trace.FlightRecorder` (Go 1.25+) over pprof. FlightRecorder maintains a ring buffer of trace data that can be dumped on-demand when something goes wrong, without the overhead of continuous profiling.

For goroutine leak detection in development, use the standard `/debug/pprof/goroutine` endpoint with `?debug=1` to inspect blocked goroutines.

---

## Phase 2: Swift App Architecture

### 2.1 Platform Protocol Layer

Every OS interaction behind a protocol with Live + Mock implementations.

**Protocols:**

| Protocol | Methods | Live impl | Mock impl |
|---|---|---|---|
| `WallpaperProviding` | setWallpaper(url:display:), currentWallpaper(display:), availableDisplays() | NSWorkspace, CGDisplay | Records calls |
| `FileSystemProviding` | fileExists(at:), write(data:to:), remove(at:), contentsOfDirectory(at:), fileSize(at:) | FileManager | In-memory dictionary |
| `KeychainProviding` | save(key:data:), load(key:), delete(key:) | Security framework | In-memory dictionary |
| `NetworkProviding` | data(for:) async throws -> (Data, URLResponse) | URLSession | Returns canned responses |
| `PlayerProviding` | play(url:), pause(), resume(), seek(to:), stop() | AVQueuePlayer | Records calls |
| `ConfigStoring` | load() -> AppConfig, save(AppConfig) | JSON file | In-memory |

**File structure:**
```
Platform/
  Protocols/
    WallpaperProviding.swift
    FileSystemProviding.swift
    KeychainProviding.swift
    NetworkProviding.swift
    PlayerProviding.swift
    ConfigStoring.swift
  Live/
    LiveWallpaperProvider.swift
    LiveFileSystem.swift
    LiveKeychain.swift
    LiveNetwork.swift
    LivePlayer.swift
    LiveConfigStore.swift
  Mock/
    MockWallpaperProvider.swift
    MockFileSystem.swift
    MockKeychain.swift
    MockNetwork.swift
    MockPlayer.swift
    MockConfigStore.swift
```

### 2.2 ViewModels

Every feature view gets a ViewModel. Pure logic, testable without UI.

| View | ViewModel | What it owns |
|---|---|---|
| HomeView | HomeViewModel | Load popular/recent, select wallpaper |
| ExploreView | ExploreViewModel | Category browsing, search, filtering |
| LibraryView | LibraryViewModel | Local video list, import, delete |
| FavoritesView | FavoritesViewModel | Load/paginate favorites |
| DetailView | DetailViewModel | Download, set wallpaper, favorite, report |
| UploadView | UploadViewModel | File selection, validation, upload flow |
| SettingsView | SettingsViewModel | Config read/write, cache management |
| AdminView | AdminViewModel | Queue, approve/reject, users, reports |
| LoginView | LoginViewModel | Login, register, validation |
| PlaylistsView | PlaylistsViewModel | CRUD playlists, assign to displays |

ViewModels take protocol dependencies via init:
```swift
@Observable
@MainActor
final class HomeViewModel {
    private let api: NetworkProviding
    private let eventLog: EventLogging

    var popular: [WallpaperCardData] = []
    var recent: [WallpaperCardData] = []
    var isLoading = true
    var error: String?

    init(api: NetworkProviding, eventLog: EventLogging) { ... }

    func load() async { ... }
}
```

### 2.3 EventLog

JSONL log of all app actions.

**Protocol:**
```swift
protocol EventLogging {
    func log(_ event: String, data: [String: Any])
}
```

**Entry format:**
```json
{"ts":"2026-03-23T14:32:01Z","sid":"a1b2c3","v":"0.2.0","event":"wallpaper_set","data":{"display":"built-in","source":"community"}}
```

**Fields on every entry:**
- `ts` - ISO 8601 timestamp
- `sid` - session UUID (generated on launch)
- `v` - app version from bundle
- `event` - event name
- `data` - event-specific payload

**Events logged:**
- Lifecycle: `app_launched`, `app_terminated`, `session_restored`
- Wallpaper: `wallpaper_set`, `wallpaper_downloaded`, `wallpaper_cached`
- Playback: `paused`, `resumed`, `playlist_advanced`
- Auth: `logged_in`, `logged_out`, `registered`
- Social: `favorite_toggled`, `reported`
- Config: `config_changed`
- Cache: `cache_evicted`
- Errors: `error`

**Storage:** `~/Library/Application Support/ScreenSpace/logs/events.jsonl`
**Rotation:** At 5MB, rename to `events.1.jsonl`, keep max 3.
**No PII** in log entries.

**Mock implementation** collects events in-memory for test assertions:
```swift
let log = MockEventLog()
let vm = HomeViewModel(api: mockAPI, eventLog: log)
await vm.load()
XCTAssertEqual(log.events.last?.event, "wallpapers_loaded")
```

### 2.4 Swift Enums

```swift
enum WallpaperStatus: String, Codable, Sendable {
    case pending
    case pendingReview = "pending_review"
    case approved
    case rejected
}

enum UserRole: String, Codable, Sendable {
    case user
    case admin
}

enum Category: String, Codable, CaseIterable, Sendable {
    case nature, abstract, urban, cinematic
    case space, underwater, minimal, other
}

enum SortOrder: String, Codable, Sendable {
    case recent, popular
}
```

Compiler enforces exhaustive switches. No more `"admin"` string comparisons.

### 2.5 Sendable Conformances

Add `Sendable` to all value types:
- `AppConfig`
- `WallpaperCardData`
- `Playlist`, `PlaylistItem`
- All API model structs in `APIModels.swift`
- `WallpaperResponse`, `AuthResponse`, `UserResponse`, etc.

### 2.6 Observable Migration

- Migrate `PauseController` from `ObservableObject`/`@Published` to `@Observable`
- Remove Combine import from `AppState`
- Replace `sink` pipeline with direct observation or `.onChange` in view layer
- Replace `NotificationCenter.addObserver` with `NotificationCenter.notifications(named:)` async sequences

### 2.7 Standardized UI Components

**Button hierarchy:**

| Level | Style | Usage | Accessibility trait |
|---|---|---|---|
| Primary | `.borderedProminent` | One per screen, main action | `.isButton` |
| Secondary | `.bordered` | Supporting actions | `.isButton` |
| Destructive | `.bordered` + `.red` tint | Delete, ban, reject | `.isButton` |
| Plain | `.plain` | Navigation, "See All" | `.isButton` + `.isLink` |

**Typography scale (semantic names):**

| Token | Font | Usage |
|---|---|---|
| `Typography.pageTitle` | `.title2.bold()` | View titles ("Your Library", "Explore") |
| `Typography.sectionTitle` | `.title3.bold()` | Section headers in shelves |
| `Typography.cardTitle` | `.subheadline` | Wallpaper card title |
| `Typography.cardMeta` | `.caption2` | Duration, resolution on cards |
| `Typography.body` | `.body` | General content |
| `Typography.meta` | `.caption` | Metadata, timestamps, descriptions |
| `Typography.label` | `.headline` | Form labels, list item titles |

**Spacing tokens** (already exist):
```swift
enum Spacing {
    static let xs: CGFloat = 4
    static let sm: CGFloat = 8
    static let md: CGFloat = 12
    static let lg: CGFloat = 16
    static let xl: CGFloat = 24
    static let xxl: CGFloat = 32
}
```

### 2.8 Accessibility Pass

Every interactive element gets:
- `.accessibilityLabel()` - what it is
- `.accessibilityHint()` - what it does when activated
- `.accessibilityAddTraits()` - role (button, link, image, header)
- `.accessibilityValue()` - current state where applicable

**Specific additions:**
- All sidebar navigation items: labels
- All action buttons in DetailView: labels + hints
- Admin action buttons: `.isButton` trait, destructive actions flagged
- Settings form controls: labels bound to controls
- Upload form fields: labels
- Card grids: identified as collections
- "Now Playing" menu item: live region for VoiceOver announcements

### 2.9 Info.plist & Entitlements

**Info.plist:**
```xml
CFBundleIdentifier: co.worldwidewebb.screenspace
CFBundleName: ScreenSpace
CFBundleDisplayName: ScreenSpace
CFBundleShortVersionString: (from git tag)
CFBundleVersion: (build number)
LSUIElement: true (menu bar app)
LSMinimumSystemVersion: 15.0
NSAppTransportSecurity:
  NSAllowsLocalNetworking: true (for localhost dev)
SUFeedURL: (Sparkle appcast URL)
```

**Entitlements (hardened runtime, not sandboxed):**
```xml
com.apple.security.network.client: true
com.apple.security.files.user-selected.read-write: true
com.apple.security.cs.allow-unsigned-executable-memory: false
```

**Keychain fix:**
Add `kSecAttrAccessible: kSecAttrAccessibleWhenUnlockedThisDeviceOnly` to all Keychain operations.

### 2.10 Deprecation Fixes

- `NSApp.activate(ignoringOtherApps: true)` -> `NSApp.activate()`
- `NSItemProvider.loadItem(forTypeIdentifier:)` -> `Transferable` protocol / `.dropDestination`
- `AdminView` creating its own `APIClient()` -> use `@Environment(AppState.self)`
- `LockScreenManager` AppleScript privilege escalation -> investigate `desktopimages` framework or document as requiring user grant

### 2.11 Concurrency Fixes

- Remove `@unchecked Sendable` from `ConfigManager` and `PlaylistManager`. Make them actors or add proper synchronization.
- Add `Task.isCancelled` checks between sequential async calls in `.task` blocks.
- `VideoPreview` `updateNSView` handles URL changes properly.
- `SettingsView` observes config reactively instead of snapshot at init.
- Download progress tracked via `URLSessionDownloadDelegate` or `AsyncBytes`.
- `DispatchQueue.main.async` -> `Task { @MainActor in }` for consistency.

---

## Phase 3: Tooling & Standards

### 3.1 Lefthook

```yaml
# lefthook.yml (project root)
pre-commit:
  parallel: true
  commands:
    go-imports:
      root: server/
      glob: "*.go"
      run: goimports -w {staged_files} && git add {staged_files}
    go-lint:
      root: server/
      glob: "*.go"
      run: golangci-lint run
    swift-format:
      root: app/
      glob: "*.swift"
      run: swiftformat --config .swiftformat {staged_files}
      stage_fixed: true
    swift-lint:
      root: app/
      glob: "*.swift"
      run: swiftlint lint --strict {staged_files}
```

### 3.2 golangci-lint

`.golangci.yml` in `server/`:

**Enabled linters:**
- Defaults: `errcheck`, `govet`, `staticcheck`, `unused`, `ineffassign`, `gosimple`
- Error handling: `errorlint` (NOT `wrapcheck`, it conflicts with sentinel error returns in the handler adapter pattern)
- Security: `gosec`
- Performance: `bodyclose`, `perfsprint`
- Style: `revive`, `misspell`, `unconvert`, `nakedret`
- Complexity: `cyclop`, `gocyclo`
- Modern Go: `modernize`, `sloglint`
- Completeness: `exhaustive`, `noctx`
- Import control: `depguard` (block `"log"`, block `"github.com/lib/pq"`)

### 3.3 SwiftFormat

`.swiftformat` in `app/`:
- `--indent 4`
- `--maxwidth 120`
- `--wraparguments before-first`
- `--wrapcollections before-first`

### 3.4 SwiftLint

`.swiftlint.yml` in `app/`:

**Enabled opt-in rules:** `empty_count`, `closure_spacing`, `first_where`, `overridden_super_call`, `fatal_error_message`, `redundant_nil_coalescing`

**Disabled rules:** (formatting rules that conflict with SwiftFormat)

**Custom rules:** `force_unwrapping` warning (not error, for Application Support paths)

### 3.5 Makefile

`server/Makefile`:
```makefile
.PHONY: help build run test lint fmt tidy audit generate migrate docker clean

help:           ## Show this help
build:          ## Build the server binary
run:            ## Build and run
test:           ## Run tests with race detector
test/cover:     ## Run tests with coverage report
lint:           ## Run golangci-lint
fmt:            ## Run goimports
tidy:           ## Run go mod tidy
audit:          ## Run all quality checks (tidy, vet, lint, vuln, test)
generate:       ## Run sqlc generate + oapi-codegen
migrate/up:     ## Run migrations up
migrate/down:   ## Run migrations down one step
migrate/create: ## Create new migration (usage: make migrate/create name=add_foo)
docker:         ## Build Docker image
clean:          ## Remove build artifacts
tools:          ## Install required dev tools
```

### 3.6 CI Updates

**server.yml:**
- Go version updated to match go.mod
- Add lint step: `golangci-lint run`
- Add vuln step: `govulncheck ./...`
- Add generate step: verify `sqlc generate` produces no diff

**app.yml:**
- Add lint step: `swiftlint lint --strict`
- Add format check: `swiftformat --lint .`

### 3.7 Dockerfile

```dockerfile
FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o server ./cmd/server

FROM alpine:3.21
RUN apk add --no-cache ffmpeg
COPY --from=builder /app/server /usr/local/bin/server
ENTRYPOINT ["server"]
```

---

## Phase 4: Security Hardening

### 4.1 Rate Limiting

| Endpoint group | Limit | Key |
|---|---|---|
| Auth (login/register) | 10/min | Per IP |
| Public reads | 120/min | Per IP |
| Authenticated writes | 30/min | Per user |
| Upload | 5/day | Per user |
| Download | 60/hour | Per user |

Rate limit responses include `Retry-After` header and structured error:
```json
{"error":{"code":"rate_limited","message":"too many requests"}}
```

### 4.2 Banned User Check

Add banned check to auth middleware (per-request, not just login):
```go
// In Auth middleware, after validating token:
user, err := userRepo.GetByID(ctx, claims.UserID)
if err != nil || user.Banned {
    respond.Error(w, 403, "banned", "account banned")
    return
}
```

**Performance:** Use an in-memory TTL cache (e.g., `sync.Map` with expiry or a small LRU) with a 60-second TTL to avoid a DB query on every authenticated request. Cache miss falls through to DB. On ban, the cache entry is evicted immediately for that user.

**Token invalidation tradeoff (documented, accepted):** Banning a user does NOT revoke their JWT. The per-request banned check catches them on their very next API call, so the effective window is near-zero. We accept this rather than adding token refresh complexity. If token refresh rotation is needed later, it's defined in Phase 7.2 (Auth Evolution). No action needed now.

### 4.3 Upload Security

- Presigned upload URL includes `Content-Length` max (from `cfg.MaxFileSize`) and `Content-Type: video/mp4`
- Finalize uses `io.LimitReader(reader, cfg.MaxFileSize+1)` instead of unbounded `io.Copy`
- If limit exceeded, return error before writing full file to disk

### 4.4 Password & Auth

- Minimum password length: 8 characters (configurable)
- Email format validation (basic regex, not full RFC 5322)
- Registration returns generic 409 "registration failed" instead of "email already exists"
- JWT secret minimum 32 characters, validated at startup

### 4.5 Error Response Consistency

All error responses use `Content-Type: application/json`:
```json
{"error":{"code":"not_found","message":"wallpaper not found"}}
```

No more `http.Error()` with JSON strings in `text/plain`.

---

## Phase 5: Testing

### 5.1 Go: Ginkgo + Gomega

Replace all `*_test.go` files with BDD-style specs.

```go
var _ = Describe("Wallpapers", func() {
    var svc *service.WallpaperService
    var mockDB *db.MockQuerier

    BeforeEach(func() {
        mockDB = db.NewMockQuerier()
        svc = service.NewWallpaperService(mockDB, mockStore, mockVideo, cfg)
    })

    Describe("GetApproved", func() {
        It("returns approved wallpapers", func() { ... })
        It("returns not found for pending wallpapers", func() { ... })
        It("returns not found for nonexistent IDs", func() { ... })
    })
})
```

**Test layers:**
- Service tests: mock the sqlc-generated `Querier` interface + mock Store
- Handler tests: mock the service interface, test HTTP request/response
- Integration tests: real database (existing pattern, kept for CI)

### 5.2 Swift: Swift Testing

Replace all `XCTestCase` with `@Test`/`@Suite`:

```swift
@Suite("HomeViewModel")
struct HomeViewModelTests {
    @Test("loads popular and recent wallpapers")
    func loadsWallpapers() async {
        let api = MockNetwork(responses: [...])
        let log = MockEventLog()
        let vm = HomeViewModel(api: api, eventLog: log)

        await vm.load()

        #expect(vm.popular.count == 5)
        #expect(vm.recent.count == 10)
        #expect(log.events.contains { $0.event == "wallpapers_loaded" })
    }
}
```

### 5.3 Swift: Snapshot Testing (post-refactor)

> **Deferred until after the refactor stabilizes.** Snapshot tests break on every UI change (typography, spacing, accessibility labels). During an active refactor they create CI noise. Add snapshot tests after Phases 1-4 are complete and UI components are stable.

```swift
@Test("home view matches snapshot")
func homeViewSnapshot() {
    let vm = HomeViewModel(api: mockAPI, eventLog: MockEventLog())
    vm.popular = [.preview, .preview, .preview]
    let view = HomeView(viewModel: vm)
    assertSnapshot(of: view, as: .image(size: CGSize(width: 1200, height: 800)))
}
```

Reference images stored in `Tests/ScreenSpaceTests/__Snapshots__/`.

### 5.4 Golden File Tests (Go)

For API response shape verification:
- Record actual HTTP responses as JSON files
- Future test runs compare against golden files
- Explicit update command when response shape changes intentionally

### 5.5 Dead Code Detection

Run Periphery on Swift app to find unreferenced declarations:
```bash
periphery scan --project app/Package.swift
```

---

## Phase 6: Bug Fixes

### 6.1 Open bugs from findings doc

| ID | Issue | Fix approach |
|---|---|---|
| BUG-1 | Hover scaling clipped | Verify if commit `6a662df` fixed it, otherwise add padding |
| BUG-2 | No refresh on HomeView | Add retry button on error state + re-trigger on window focus |
| UX-1 | Empty state when no wallpapers | EmptyStateView with "No wallpapers yet" message |
| UX-2 | No "no results" in search | Empty state in ExploreView search results |
| UX-3/4 | Favorite/report visible when logged out | Hide buttons or show login prompt |
| M4 | "Now Playing" menu hardcoded | Wire to AppState.currentWallpaperTitle |
| M9 | Screensaver never installed | Add install button or document manual install |
| M10 | No hover preview on cards | Future phase (complex, low priority) |
| C20 | No VoiceOver roles | Covered by accessibility pass (Phase 2.8) |
| C21 | selectedSection not persisted | Save to config, restore on launch |
| PROD-4 | Sparkle not integrated | Wire SUFeedURL + SPUStandardUpdaterController |

### 6.2 Server fixes from Go audit

| Issue | Fix |
|---|---|
| `go 1.25.0` in go.mod | Already updated to 1.26 (see Toolchain section) |
| `ListByUser` scans different columns | Unify with sqlc-generated code |
| `Popular`/`Recent` mutate request URL | Refactor to pass params directly |
| Missing error wrapping in storage | Add context to all error returns |
| `json.Encode` errors ignored | Log on failure |
| Seed script hardcoded passwords | Add env var override, warn in output |

### 6.3 Swift fixes from Swift audit

| Issue | Fix |
|---|---|
| `AdminView` creates own `APIClient` | Use `@Environment(AppState.self)` |
| `LockScreenManager` AppleScript escalation | Investigate safe alternative or document limitation |
| `NSApp.activate(ignoringOtherApps:)` deprecated | Use `NSApp.activate()` |
| `SettingsView` config snapshot | Observe ConfigManager reactively |
| Download progress never updates | Implement URLSession delegate |
| `.task` blocks missing cancellation checks | Add `Task.isCancelled` checks |
| `VideoPreview` doesn't handle URL changes | Implement `updateNSView` |
| Force unwraps on Application Support paths | `guard let` + `fatalError("message")` |

---

## Phase 7: Future Considerations

### 7.1 macOS 26+ Only

When macOS 26 ships, drop macOS 15 support. Benefits:
- Latest SwiftUI APIs (window management, new modifiers)
- Latest Swift Testing improvements
- Smaller testing matrix
- Can use any new wallpaper/desktop APIs Apple introduces

### 7.2 Auth Evolution

- **Sign in with Apple / Passkeys** - eliminate password management
- **Token refresh rotation** - short-lived access tokens (15min) + refresh tokens
- **MFA/2FA** - TOTP via authenticator app
- **Email verification** - prevent spam accounts
- **Password reset flow** - doesn't exist at all

### 7.3 Scaling

- **Redis-backed rate limiting** - for multi-instance deployments
- **CDN** (CloudFront/Bunny) - for wallpaper delivery instead of presigned S3 URLs
- **Background job queue** (River or Postgres-backed) - Finalize should be async
- **Postgres read replicas** - if browse traffic grows

### 7.4 App Features

- **Sparkle integration** - blocking for real distribution, should be soon after refactor
- **Crash reporting** - Sentry or similar
- **Analytics** - privacy-respecting, opt-in usage stats
- **iCloud sync** - favorites, playlists, config across Macs
- **iOS companion app** - browse/favorite from phone
- **Desktop widget** - current wallpaper / quick switch

### 7.5 Content & Moderation

- **NSFW detection** - automated content moderation on upload
- **Video transcoding** - accept any format, transcode server-side
- **Collections/curations** - admin-curated sets
- **Creator profiles** - public uploader pages

### 7.6 Infrastructure

- **OpenTelemetry/tracing** - distributed request tracing
- **Alerting** - error rate monitoring
- **Database backups** - automated with PITR
- **Staging environment** - currently just dev and prod
- **Certificate pinning** - on the Swift app

### 7.7 Tart VM Integration

For visual integration testing of wallpaper changes and UI rendering.

**Setup:**
1. `brew install tart`
2. `tart create --from-ipsw latest screenspace-test` (~25GB base image)
3. Boot, grant accessibility permissions, save as base
4. Clone for each test run (copy-on-write, instant)

**Test workflow:**
1. `tart clone screenspace-test test-run`
2. Build app on host: `swift build -c release`
3. Copy binary: `tart copy-to test-run .build/release/ScreenSpace /Applications/`
4. SSH into VM, launch app, trigger action
5. `screencapture` inside VM
6. `tart copy-from test-run /tmp/screenshot.png ./`
7. Read screenshot, verify visually
8. `tart delete test-run`

**Resource requirements:** ~25GB disk, 4GB RAM, 2 cores. Feasible on M2 Pro 32GB.

**Phase:** After main refactor is complete. Not blocking.

---

## Project Structure (Target State)

### Go Server

```
server/
  api/
    openapi.yaml
  cmd/
    server/main.go          # wiring only
    seed/main.go
  internal/
    config/
      config.go             # single struct, all env vars
    handler/
      wallpaper.go          # thin HTTP adapters
      auth.go
      favorite.go
      report.go
      admin.go
    service/
      wallpaper.go          # business logic
      auth.go
      video.go
      favorite.go
      report.go
    respond/
      json.go               # respond.JSON, respond.Paginated
      errors.go             # AppError, sentinel errors, Wrap()
      pagination.go         # parse + respond
    middleware/
      auth.go
      admin.go
      ratelimit.go
      logger.go
      requestid.go
      security.go           # security headers
      bodysize.go           # MaxBytesReader
    storage/
      store.go              # interface
      s3.go
    types/
      enums.go              # WallpaperStatus, UserRole, Category, SortOrder
  db/
    migrations/
    queries/
      wallpaper.sql
      user.sql
      favorite.sql
      report.sql
    generated/              # sqlc output
    sqlc.yaml
  generated/                # oapi-codegen output
    server.gen.go
    types.gen.go
  Makefile
  Dockerfile
  .golangci.yml
```

### Swift App

```
app/Sources/ScreenSpace/
  App/
    App.swift
    AppState.swift
    Environment.swift       # DI container
  Features/
    Home/
      HomeView.swift
      HomeViewModel.swift
    Explore/
      ExploreView.swift
      ExploreViewModel.swift
    Library/
      LibraryView.swift
      LibraryViewModel.swift
    Favorites/
      FavoritesView.swift
      FavoritesViewModel.swift
    Upload/
      UploadView.swift
      UploadViewModel.swift
    Settings/
      SettingsView.swift
      SettingsViewModel.swift
    Admin/
      AdminView.swift
      AdminViewModel.swift
    Detail/
      DetailView.swift
      DetailViewModel.swift
    Login/
      LoginView.swift
      LoginViewModel.swift
    Playlists/
      PlaylistsView.swift
      PlaylistsViewModel.swift
  Platform/
    Protocols/
    Live/
    Mock/
  Engine/
    WallpaperEngine.swift
    PauseController.swift
    DisplayIdentifier.swift
  Core/
    API/
      APIClient.swift       # generated types come from swift-openapi-generator
    Config/
      AppConfig.swift
    Cache/
      CacheManager.swift
    EventLog/
      EventLog.swift
      EventLogEntry.swift
    Types/
      Enums.swift           # WallpaperStatus, UserRole, Category
  UI/
    Components/
      WallpaperCard.swift
      HeroSection.swift
      ShelfRow.swift
      EmptyStateView.swift
      GlassCard.swift
      ResolutionBadge.swift
    Design/
      DesignTokens.swift    # Spacing
      Typography.swift      # semantic font tokens
      ButtonStyles.swift    # standardized button hierarchy
    Modifiers/
      ErrorAlert.swift
    Windows/
      GalleryWindow.swift
      WallpaperWindow.swift

app/Tests/ScreenSpaceTests/
  Features/
    HomeViewModelTests.swift
    ExploreViewModelTests.swift
    ...
  Snapshots/
    HomeViewSnapshotTests.swift
    ...
    __Snapshots__/          # reference images
  Platform/
    MockTests.swift
  Core/
    APIClientTests.swift
    ConfigTests.swift
    EventLogTests.swift
```

### Root

```
screenspace/
  .github/workflows/
    app.yml
    server.yml
    appcast.yml
  app/
  server/
  docs/
  lefthook.yml
  .gitignore
```
