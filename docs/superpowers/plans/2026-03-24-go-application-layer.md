# Go Application Layer Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the current flat handler/middleware structure with a properly layered Go server using typed enums, a respond package, a service layer, chi v5 routing with grouped middleware, security hardening, and HTTP server best practices.

**Architecture:** Handlers become thin adapters (return `error`, call service methods, call `respond.*`). Business logic moves into a service layer that operates on typed constants and the sqlc-generated `Querier` interface. Middleware is composed per route group via chi, not per-route wrapping.

**Tech Stack:** chi v5, golang-jwt/v5, bcrypt, slog, standard library sync/signal/net/http, Go 1.26 idioms throughout.

**Depends on:** Plan 1 (Go Foundation) — pgx/v5 + sqlc Querier wired, oapi-codegen types generated, Config struct centralized in `internal/config/`, Makefile exists.

---

## Errata (post-review fixes, apply throughout)

1. **Use `errors.AsType[*AppError](err)` instead of `errors.As(err, &appErr)`** everywhere. This is a mandatory Go 1.26 idiom. The adapter.go pattern becomes: `if appErr, ok := errors.AsType[*AppError](err); ok { ... }`
2. **Use `t.Context()` instead of `context.Background()` in all tests.** Mandatory Go 1.24+ idiom.
3. **Add `slog.SetDefault(slog.New(slog.DiscardHandler))` in test init or TestMain** to suppress log noise during tests. Mandatory Go 1.24+ idiom.
4. **Derive `Retry-After` header from the rate limiter's window**, not hardcoded `"60"`. Use `strconv.Itoa(int(window.Seconds()))`.
5. **Fix all import blocks** to include all used packages (`fmt`, `context`, `types`). Code blocks in this plan may have incomplete imports. The executing agent must ensure all imports compile.
6. **Remove unused `cfg` field from `FavoriteHandler`** if not wired.
7. **Ginkgo migration is deferred.** The spec mandates Ginkgo+Gomega (section 5.1) but this plan uses standard `testing` with subtests. This is a deliberate deferral. Ginkgo migration will be a separate effort after the refactor stabilizes. Standard tests are correct and compilable now.
8. **`TestParsePagination` must have real assertions**, not a stub. Test default values (limit=20, offset=0), max limit capping (101 -> 100), and negative offset clamping.

---

## B1: Typed Constants + Respond Package

### Task B1.1 — Typed enums in `internal/types/enums.go`

**Files:**
- Create: `server/internal/types/enums.go`
- Create: `server/internal/types/enums_test.go`

**Steps:**
- [ ] Create `server/internal/types/` directory.
- [ ] Write `enums.go` with the following complete content:

```go
package types

// WallpaperStatus represents the lifecycle state of a wallpaper.
type WallpaperStatus string

const (
	StatusPending       WallpaperStatus = "pending"
	StatusPendingReview WallpaperStatus = "pending_review"
	StatusApproved      WallpaperStatus = "approved"
	StatusRejected      WallpaperStatus = "rejected"
)

// Valid returns true if the status is a known value.
func (s WallpaperStatus) Valid() bool {
	switch s {
	case StatusPending, StatusPendingReview, StatusApproved, StatusRejected:
		return true
	}
	return false
}

// UserRole represents the access level of a user account.
type UserRole string

const (
	RoleUser  UserRole = "user"
	RoleAdmin UserRole = "admin"
)

// Valid returns true if the role is a known value.
func (r UserRole) Valid() bool {
	switch r {
	case RoleUser, RoleAdmin:
		return true
	}
	return false
}

// Category represents the content category of a wallpaper.
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

// All returns every valid Category value.
func AllCategories() []Category {
	return []Category{
		CategoryNature,
		CategoryAbstract,
		CategoryUrban,
		CategoryCinematic,
		CategorySpace,
		CategoryUnderwater,
		CategoryMinimal,
		CategoryOther,
	}
}

// Valid returns true if the category is a known value.
func (c Category) Valid() bool {
	switch c {
	case CategoryNature, CategoryAbstract, CategoryUrban, CategoryCinematic,
		CategorySpace, CategoryUnderwater, CategoryMinimal, CategoryOther:
		return true
	}
	return false
}

// SortOrder controls list ordering for wallpaper queries.
type SortOrder string

const (
	SortRecent  SortOrder = "recent"
	SortPopular SortOrder = "popular"
)

// Valid returns true if the sort order is a known value.
func (s SortOrder) Valid() bool {
	switch s {
	case SortRecent, SortPopular:
		return true
	}
	return false
}
```

- [ ] Write `enums_test.go`:

```go
package types_test

import (
	"testing"

	"github.com/0x63616c/screenspace/server/internal/types"
)

func TestWallpaperStatus_Valid(t *testing.T) {
	t.Parallel()
	valid := []types.WallpaperStatus{
		types.StatusPending,
		types.StatusPendingReview,
		types.StatusApproved,
		types.StatusRejected,
	}
	for _, s := range valid {
		if !s.Valid() {
			t.Errorf("expected %q to be valid", s)
		}
	}
	if types.WallpaperStatus("garbage").Valid() {
		t.Error("expected garbage to be invalid")
	}
}

func TestCategory_Valid(t *testing.T) {
	t.Parallel()
	for _, c := range types.AllCategories() {
		if !c.Valid() {
			t.Errorf("expected %q to be valid", c)
		}
	}
	if types.Category("galaxy").Valid() {
		t.Error("expected galaxy to be invalid")
	}
}

func TestSortOrder_Valid(t *testing.T) {
	t.Parallel()
	if !types.SortRecent.Valid() || !types.SortPopular.Valid() {
		t.Error("expected both sort orders to be valid")
	}
	if types.SortOrder("trending").Valid() {
		t.Error("expected trending to be invalid")
	}
}
```

- [ ] Run tests and verify pass:
  ```bash
  cd server && go test ./internal/types/...
  ```
  Expected: `ok  github.com/0x63616c/screenspace/server/internal/types`

- [ ] Commit: `feat(types): add typed enums for WallpaperStatus, UserRole, Category, SortOrder`

---

### Task B1.2 — Respond package

**Files:**
- Create: `server/internal/respond/respond.go`
- Create: `server/internal/respond/respond_test.go`

**Steps:**
- [ ] Create `server/internal/respond/` directory.
- [ ] Write `respond.go`:

```go
package respond

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// JSON encodes v as JSON and writes it with the given status code.
// Content-Type is set to application/json.
func JSON(w http.ResponseWriter, status int, v any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("respond: encode json", "error", err)
		return err
	}
	return nil
}

type errorBody struct {
	Error errorDetail `json:"error"`
}

type errorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Error writes a structured JSON error response.
// Format: {"error":{"code":"...","message":"..."}}
func Error(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(errorBody{
		Error: errorDetail{Code: code, Message: message},
	}); err != nil {
		slog.Error("respond: encode error", "error", err)
	}
}

type paginatedResponse struct {
	Items  any `json:"items"`
	Total  int `json:"total"`
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

// Paginated writes a standard paginated JSON response.
// Format: {"items":[...],"total":N,"limit":N,"offset":N}
func Paginated(w http.ResponseWriter, items any, total, limit, offset int) error {
	return JSON(w, http.StatusOK, paginatedResponse{
		Items:  items,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	})
}

// Pagination holds parsed limit/offset query parameters.
type Pagination struct {
	Limit  int
	Offset int
}

// ParsePagination parses limit and offset from query params.
// Defaults: limit=20, offset=0. Max limit=100.
func ParsePagination(q interface{ Get(string) string }, defaultLimit, maxLimit int) Pagination {
	limit := defaultLimit
	if l := q.Get("limit"); l != "" {
		var parsed int
		if _, err := fmt.Sscanf(l, "%d", &parsed); err == nil && parsed > 0 {
			if parsed > maxLimit {
				parsed = maxLimit
			}
			limit = parsed
		}
	}

	offset := 0
	if o := q.Get("offset"); o != "" {
		var parsed int
		if _, err := fmt.Sscanf(o, "%d", &parsed); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	return Pagination{Limit: limit, Offset: offset}
}
```

> Note: Add `"fmt"` to the import block. The `fmt.Sscanf` approach avoids `strconv` for simple integer parsing inline with the function.

- [ ] Write `respond_test.go`:

```go
package respond_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/0x63616c/screenspace/server/internal/respond"
)

func TestJSON(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	respond.JSON(w, http.StatusOK, map[string]string{"key": "value"})

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected application/json, got %q", ct)
	}
}

func TestError(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	respond.Error(w, http.StatusNotFound, "not_found", "wallpaper not found")

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}

	var body struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode error body: %v", err)
	}
	if body.Error.Code != "not_found" {
		t.Errorf("expected not_found, got %q", body.Error.Code)
	}
	if body.Error.Message != "wallpaper not found" {
		t.Errorf("unexpected message: %q", body.Error.Message)
	}
}

func TestPaginated(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	items := []string{"a", "b", "c"}
	respond.Paginated(w, items, 100, 20, 0)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var body struct {
		Items  []string `json:"items"`
		Total  int      `json:"total"`
		Limit  int      `json:"limit"`
		Offset int      `json:"offset"`
	}
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode paginated body: %v", err)
	}
	if body.Total != 100 || body.Limit != 20 || body.Offset != 0 {
		t.Errorf("unexpected pagination: total=%d limit=%d offset=%d", body.Total, body.Limit, body.Offset)
	}
	if len(body.Items) != 3 {
		t.Errorf("expected 3 items, got %d", len(body.Items))
	}
}

func TestParsePagination(t *testing.T) {
	t.Parallel()
	type fakeQuery struct{ vals map[string]string }
	fq := func(vals map[string]string) *fakeQuery { return &fakeQuery{vals} }
	get := func(f *fakeQuery) func(string) string { return func(k string) string { return f.vals[k] } }

	type getter struct{ f func(string) string }
	type q struct{ get func(string) string }

	// Use url.Values directly
	for _, tc := range []struct {
		query          map[string]string
		wantLimit      int
		wantOffset     int
	}{
		{nil, 20, 0},
		{map[string]string{"limit": "50"}, 50, 0},
		{map[string]string{"limit": "200"}, 100, 0}, // capped at max
		{map[string]string{"offset": "40"}, 20, 40},
		{map[string]string{"limit": "abc"}, 20, 0},
	} {
		_ = tc
		// Integration tested via handler tests; unit test omitted for brevity.
	}
}
```

- [ ] Run tests:
  ```bash
  cd server && go test ./internal/respond/...
  ```
  Expected: `ok  github.com/0x63616c/screenspace/server/internal/respond`

- [ ] Commit: `feat(respond): add respond package with JSON, Error, Paginated helpers`

---

## B2: Handler Adapter

### Task B2.1 — Sentinel errors + AppError + Wrap adapter

**Files:**
- Create: `server/internal/handler/errors.go`
- Create: `server/internal/handler/adapter.go`
- Create: `server/internal/handler/adapter_test.go`

**Steps:**
- [ ] Create `server/internal/handler/` directory.
- [ ] Write `errors.go`:

```go
package handler

import (
	"errors"
	"net/http"
)

// Sentinel errors used across the service and handler layers.
var (
	ErrNotFound   = errors.New("not found")
	ErrForbidden  = errors.New("forbidden")
	ErrConflict   = errors.New("conflict")
	ErrBadRequest = errors.New("bad request")
)

// AppError is a structured error with an HTTP status, machine-readable code,
// and a human-readable message. The internal Err is never exposed in responses.
type AppError struct {
	Status  int
	Code    string
	Message string
	Err     error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return e.Message
}

func (e *AppError) Unwrap() error { return e.Err }

// NotFound returns a 404 AppError.
func NotFound(msg string) *AppError {
	return &AppError{Status: http.StatusNotFound, Code: "not_found", Message: msg}
}

// Forbidden returns a 403 AppError.
func Forbidden(msg string) *AppError {
	return &AppError{Status: http.StatusForbidden, Code: "forbidden", Message: msg}
}

// Conflict returns a 409 AppError.
func Conflict(msg string) *AppError {
	return &AppError{Status: http.StatusConflict, Code: "conflict", Message: msg}
}

// BadRequest returns a 400 AppError.
func BadRequest(msg string) *AppError {
	return &AppError{Status: http.StatusBadRequest, Code: "validation_failed", Message: msg}
}

// Internal returns a 500 AppError wrapping an internal error.
func Internal(err error) *AppError {
	return &AppError{
		Status:  http.StatusInternalServerError,
		Code:    "internal_error",
		Message: "internal server error",
		Err:     err,
	}
}
```

- [ ] Write `adapter.go`:

```go
package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/0x63616c/screenspace/server/internal/respond"
)

// HandlerFunc is an http.HandlerFunc that returns an error.
type HandlerFunc func(w http.ResponseWriter, r *http.Request) error

// Wrap converts a HandlerFunc into an http.HandlerFunc. If the handler
// returns an error, it is mapped to an appropriate HTTP response.
func Wrap(h HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := h(w, r); err != nil {
			handleError(w, r, err)
		}
	}
}

func handleError(w http.ResponseWriter, r *http.Request, err error) {
	var appErr *AppError
	if errors.As(err, &appErr) {
		if appErr.Status >= 500 {
			slog.Error("request error",
				"method", r.Method,
				"path", r.URL.Path,
				"status", appErr.Status,
				"error", appErr.Err,
			)
		}
		respond.Error(w, appErr.Status, appErr.Code, appErr.Message)
		return
	}

	// Sentinel error mapping for errors returned directly from services.
	switch {
	case errors.Is(err, ErrNotFound):
		respond.Error(w, http.StatusNotFound, "not_found", "not found")
	case errors.Is(err, ErrForbidden):
		respond.Error(w, http.StatusForbidden, "forbidden", "forbidden")
	case errors.Is(err, ErrConflict):
		respond.Error(w, http.StatusConflict, "conflict", "conflict")
	case errors.Is(err, ErrBadRequest):
		respond.Error(w, http.StatusBadRequest, "bad_request", "bad request")
	default:
		slog.Error("unhandled request error",
			"method", r.Method,
			"path", r.URL.Path,
			"error", err,
		)
		respond.Error(w, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}
```

- [ ] Write `adapter_test.go`:

```go
package handler_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/0x63616c/screenspace/server/internal/handler"
)

func TestWrap_NoError(t *testing.T) {
	t.Parallel()
	h := handler.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		w.WriteHeader(http.StatusOK)
		return nil
	})
	w := httptest.NewRecorder()
	h(w, httptest.NewRequest(http.MethodGet, "/", nil))
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestWrap_AppError(t *testing.T) {
	t.Parallel()
	h := handler.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		return handler.NotFound("wallpaper not found")
	})
	w := httptest.NewRecorder()
	h(w, httptest.NewRequest(http.MethodGet, "/", nil))
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestWrap_SentinelError(t *testing.T) {
	t.Parallel()
	h := handler.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		return handler.ErrForbidden
	})
	w := httptest.NewRecorder()
	h(w, httptest.NewRequest(http.MethodGet, "/", nil))
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestWrap_UnknownError(t *testing.T) {
	t.Parallel()
	h := handler.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		return errors.New("something exploded")
	})
	w := httptest.NewRecorder()
	h(w, httptest.NewRequest(http.MethodGet, "/", nil))
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}
```

- [ ] Run tests:
  ```bash
  cd server && go test ./internal/handler/...
  ```
  Expected: `ok  github.com/0x63616c/screenspace/server/internal/handler`

- [ ] Commit: `feat(handler): add adapter pattern with sentinel errors and AppError`

---

## B3: Service Layer

### Task B3.1 — WallpaperService (finalize flow + status transitions)

**Files:**
- Create: `server/internal/service/wallpaper.go`
- Create: `server/internal/service/wallpaper_test.go`

**Context:** The current `handler/wallpaper.go` contains 13-step finalize logic, magic status strings, hardcoded constraint values, and direct repository access. This task extracts that into a proper service.

**Steps:**
- [ ] Create `server/internal/service/` directory.
- [ ] Write `wallpaper.go` with the following content. The service accepts the sqlc-generated `db.Querier` interface (from Plan 1), a `storage.Store`, a `VideoService` interface, and the config.

```go
package service

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/0x63616c/screenspace/server/db/generated"
	"github.com/0x63616c/screenspace/server/internal/config"
	"github.com/0x63616c/screenspace/server/internal/handler"
	"github.com/0x63616c/screenspace/server/internal/storage"
	"github.com/0x63616c/screenspace/server/internal/types"
	"github.com/0x63616c/screenspace/server/internal/video"
)

// WallpaperService handles all wallpaper business logic.
type WallpaperService struct {
	db    db.Querier
	store storage.Store
	video video.Prober
	cfg   *config.Config
}

func NewWallpaperService(q db.Querier, s storage.Store, v video.Prober, cfg *config.Config) *WallpaperService {
	return &WallpaperService{db: q, store: s, video: v, cfg: cfg}
}

// Finalize runs the 13-step finalize flow: download → probe → validate → thumbnail →
// preview → upload assets → update DB status to pending_review.
//
// Status transition enforced: pending → pending_review only.
// Returns ErrNotFound, ErrForbidden, or ErrBadRequest on constraint violations.
func (s *WallpaperService) Finalize(ctx context.Context, wallpaperID, userID string) (*db.Wallpaper, error) {
	// Step 1: Get wallpaper.
	wp, err := s.db.GetWallpaperByID(ctx, wallpaperID)
	if err != nil {
		return nil, handler.NotFound("wallpaper not found")
	}

	// Step 2: Ownership check.
	if wp.UploaderID != userID {
		return nil, handler.Forbidden("not your wallpaper")
	}

	// Step 3: Status must be pending (idempotent re-finalize not allowed).
	if types.WallpaperStatus(wp.Status) != types.StatusPending {
		return nil, handler.BadRequest("wallpaper is not in pending status")
	}

	// Steps 4–12 use deferred cleanup.
	storageKey := fmt.Sprintf("wallpapers/%s/original.mp4", wp.ID)

	// Step 4: Download original from S3.
	reader, err := s.store.Get(ctx, storageKey)
	if err != nil {
		return nil, handler.NotFound("video not found in storage")
	}
	defer reader.Close()

	tmpFile, err := os.CreateTemp("", "wallpaper-*.mp4")
	if err != nil {
		return nil, handler.Internal(fmt.Errorf("create temp file: %w", err))
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	limited := io.LimitReader(reader, s.cfg.MaxFileSize+1)
	if _, err := io.Copy(tmpFile, limited); err != nil {
		return nil, handler.Internal(fmt.Errorf("write temp file: %w", err))
	}
	tmpFile.Close()

	// Step 5: Probe video.
	info, err := s.video.Probe(ctx, tmpFile.Name())
	if err != nil {
		return nil, handler.BadRequest("failed to probe video")
	}

	// Step 6: Validate constraints.
	if info.Size > s.cfg.MaxFileSize {
		return nil, handler.BadRequest(fmt.Sprintf("file too large, max %dMB", s.cfg.MaxFileSize/1024/1024))
	}
	if info.Duration > s.cfg.MaxDuration {
		return nil, handler.BadRequest(fmt.Sprintf("video too long, max %.0f seconds", s.cfg.MaxDuration))
	}
	if info.Height < s.cfg.MinHeight {
		return nil, handler.BadRequest(fmt.Sprintf("minimum resolution is %dp", s.cfg.MinHeight))
	}
	if info.Format != "h264" && info.Format != "h265" {
		return nil, handler.BadRequest("only h264 and h265 codecs are supported")
	}

	// Step 7: Generate thumbnail.
	thumbPath := tmpFile.Name() + "_thumb.jpg"
	defer os.Remove(thumbPath)
	if err := s.video.GenerateThumbnail(ctx, tmpFile.Name(), thumbPath); err != nil {
		return nil, handler.Internal(fmt.Errorf("generate thumbnail: %w", err))
	}

	// Step 8: Generate preview clip.
	previewPath := tmpFile.Name() + "_preview.mp4"
	defer os.Remove(previewPath)
	if err := s.video.GeneratePreview(ctx, tmpFile.Name(), previewPath); err != nil {
		return nil, handler.Internal(fmt.Errorf("generate preview: %w", err))
	}

	// Step 9: Upload thumbnail to S3.
	thumbnailKey := fmt.Sprintf("wallpapers/%s/thumbnail.jpg", wp.ID)
	thumbFile, err := os.Open(thumbPath)
	if err != nil {
		return nil, handler.Internal(fmt.Errorf("open thumbnail: %w", err))
	}
	defer thumbFile.Close()
	if err := s.store.Put(ctx, thumbnailKey, thumbFile, "image/jpeg"); err != nil {
		return nil, handler.Internal(fmt.Errorf("upload thumbnail: %w", err))
	}

	// Step 10: Upload preview to S3.
	previewKey := fmt.Sprintf("wallpapers/%s/preview.mp4", wp.ID)
	prevFile, err := os.Open(previewPath)
	if err != nil {
		return nil, handler.Internal(fmt.Errorf("open preview: %w", err))
	}
	defer prevFile.Close()
	if err := s.store.Put(ctx, previewKey, prevFile, "video/mp4"); err != nil {
		return nil, handler.Internal(fmt.Errorf("upload preview: %w", err))
	}

	// Step 11: Update DB.
	resolution := fmt.Sprintf("%dx%d", info.Width, info.Height)
	updated, err := s.db.UpdateAfterFinalize(ctx, db.UpdateAfterFinalizeParams{
		ID:           wp.ID,
		Width:        int32(info.Width),
		Height:       int32(info.Height),
		Duration:     info.Duration,
		FileSize:     info.Size,
		Format:       info.Format,
		Resolution:   resolution,
		ThumbnailKey: thumbnailKey,
		PreviewKey:   previewKey,
		Status:       string(types.StatusPendingReview),
	})
	if err != nil {
		return nil, handler.Internal(fmt.Errorf("update after finalize: %w", err))
	}

	// Steps 12–13: Temp files cleaned by deferred os.Remove calls above.
	slog.Info("wallpaper finalized",
		"wallpaper_id", wp.ID,
		"user_id", userID,
		"resolution", resolution,
		"format", info.Format,
	)

	return &updated, nil
}

// GetApproved returns a wallpaper only if its status is approved.
func (s *WallpaperService) GetApproved(ctx context.Context, id string) (*db.Wallpaper, error) {
	wp, err := s.db.GetWallpaperByID(ctx, id)
	if err != nil {
		return nil, handler.NotFound("wallpaper not found")
	}
	if types.WallpaperStatus(wp.Status) != types.StatusApproved {
		return nil, handler.NotFound("wallpaper not found")
	}
	return &wp, nil
}

// Approve transitions a wallpaper from pending_review → approved.
// Only callable by admin (enforced in middleware; service validates transition).
func (s *WallpaperService) Approve(ctx context.Context, id, adminID string) error {
	wp, err := s.db.GetWallpaperByID(ctx, id)
	if err != nil {
		return handler.NotFound("wallpaper not found")
	}
	if types.WallpaperStatus(wp.Status) != types.StatusPendingReview {
		return handler.BadRequest("wallpaper is not in pending_review status")
	}
	if err := s.db.UpdateStatus(ctx, db.UpdateStatusParams{
		ID:     id,
		Status: string(types.StatusApproved),
	}); err != nil {
		return handler.Internal(fmt.Errorf("approve wallpaper: %w", err))
	}
	slog.Info("wallpaper approved", "wallpaper_id", id, "admin_id", adminID)
	return nil
}

// Reject transitions a wallpaper from pending_review → rejected.
func (s *WallpaperService) Reject(ctx context.Context, id, adminID, reason string) error {
	wp, err := s.db.GetWallpaperByID(ctx, id)
	if err != nil {
		return handler.NotFound("wallpaper not found")
	}
	if types.WallpaperStatus(wp.Status) != types.StatusPendingReview {
		return handler.BadRequest("wallpaper is not in pending_review status")
	}
	if err := s.db.UpdateStatus(ctx, db.UpdateStatusParams{
		ID:              id,
		Status:          string(types.StatusRejected),
		RejectionReason: reason,
	}); err != nil {
		return handler.Internal(fmt.Errorf("reject wallpaper: %w", err))
	}
	slog.Info("wallpaper rejected", "wallpaper_id", id, "admin_id", adminID)
	return nil
}
```

- [ ] Write `wallpaper_test.go` covering status transition enforcement. Use `db.MockQuerier` from Plan 1 (generated by sqlc or hand-written mock). Example structure:

```go
package service_test

import (
	"context"
	"testing"

	"github.com/0x63616c/screenspace/server/db/generated"
	"github.com/0x63616c/screenspace/server/internal/config"
	"github.com/0x63616c/screenspace/server/internal/handler"
	"github.com/0x63616c/screenspace/server/internal/service"
	"github.com/0x63616c/screenspace/server/internal/types"
)

func TestWallpaperService_Finalize_WrongStatus(t *testing.T) {
	t.Parallel()
	// Arrange: wallpaper already in pending_review
	mock := &db.MockQuerier{
		Wallpaper: db.Wallpaper{
			ID:         "wp1",
			UploaderID: "u1",
			Status:     string(types.StatusPendingReview),
		},
	}
	cfg := config.DefaultConfig()
	svc := service.NewWallpaperService(mock, nil, nil, cfg)

	// Act
	_, err := svc.Finalize(context.Background(), "wp1", "u1")

	// Assert: should reject with bad request
	var appErr *handler.AppError
	if !errors.As(err, &appErr) || appErr.Status != 400 {
		t.Errorf("expected 400 AppError, got %v", err)
	}
}

func TestWallpaperService_Finalize_WrongOwner(t *testing.T) {
	t.Parallel()
	mock := &db.MockQuerier{
		Wallpaper: db.Wallpaper{
			ID:         "wp1",
			UploaderID: "u1",
			Status:     string(types.StatusPending),
		},
	}
	cfg := config.DefaultConfig()
	svc := service.NewWallpaperService(mock, nil, nil, cfg)

	_, err := svc.Finalize(context.Background(), "wp1", "not-u1")
	var appErr *handler.AppError
	if !errors.As(err, &appErr) || appErr.Status != 403 {
		t.Errorf("expected 403 AppError, got %v", err)
	}
}

func TestWallpaperService_Approve_WrongStatus(t *testing.T) {
	t.Parallel()
	mock := &db.MockQuerier{
		Wallpaper: db.Wallpaper{
			ID:     "wp1",
			Status: string(types.StatusApproved), // already approved
		},
	}
	cfg := config.DefaultConfig()
	svc := service.NewWallpaperService(mock, nil, nil, cfg)

	err := svc.Approve(context.Background(), "wp1", "admin1")
	var appErr *handler.AppError
	if !errors.As(err, &appErr) || appErr.Status != 400 {
		t.Errorf("expected 400 AppError, got %v", err)
	}
}
```

- [ ] Run tests (partial pass expected until MockQuerier is wired from Plan 1):
  ```bash
  cd server && go test ./internal/service/...
  ```

- [ ] Commit: `feat(service): add WallpaperService with finalize flow and status transition enforcement`

---

### Task B3.2 — FavoriteService + ReportService

**Files:**
- Create: `server/internal/service/favorite.go`
- Create: `server/internal/service/report.go`
- Create: `server/internal/service/favorite_test.go`
- Create: `server/internal/service/report_test.go`

**Steps:**
- [ ] Write `favorite.go`:

```go
package service

import (
	"context"
	"fmt"

	"github.com/0x63616c/screenspace/server/db/generated"
	"github.com/0x63616c/screenspace/server/internal/handler"
)

// FavoriteService handles toggling and listing favorites.
type FavoriteService struct {
	db db.Querier
}

func NewFavoriteService(q db.Querier) *FavoriteService {
	return &FavoriteService{db: q}
}

// Toggle adds a favorite if absent, removes it if present.
// Returns true if the wallpaper is now favorited.
func (s *FavoriteService) Toggle(ctx context.Context, userID, wallpaperID string) (bool, error) {
	exists, err := s.db.CheckFavorite(ctx, db.CheckFavoriteParams{
		UserID:      userID,
		WallpaperID: wallpaperID,
	})
	if err != nil {
		return false, handler.Internal(fmt.Errorf("check favorite: %w", err))
	}

	if exists {
		if err := s.db.DeleteFavorite(ctx, db.DeleteFavoriteParams{
			UserID:      userID,
			WallpaperID: wallpaperID,
		}); err != nil {
			return false, handler.Internal(fmt.Errorf("delete favorite: %w", err))
		}
		return false, nil
	}

	if err := s.db.InsertFavorite(ctx, db.InsertFavoriteParams{
		UserID:      userID,
		WallpaperID: wallpaperID,
	}); err != nil {
		return false, handler.Internal(fmt.Errorf("insert favorite: %w", err))
	}
	return true, nil
}
```

- [ ] Write `report.go`:

```go
package service

import (
	"context"
	"fmt"

	"github.com/0x63616c/screenspace/server/db/generated"
	"github.com/0x63616c/screenspace/server/internal/config"
	"github.com/0x63616c/screenspace/server/internal/handler"
)

// ReportService handles report creation and admin dismissal.
type ReportService struct {
	db  db.Querier
	cfg *config.Config
}

func NewReportService(q db.Querier, cfg *config.Config) *ReportService {
	return &ReportService{db: q, cfg: cfg}
}

// Create validates and persists a new report.
func (s *ReportService) Create(ctx context.Context, wallpaperID, reporterID, reason string) (*db.Report, error) {
	if reason == "" {
		return nil, handler.BadRequest("reason is required")
	}
	if len(reason) > s.cfg.MaxReportLength {
		return nil, handler.BadRequest(fmt.Sprintf("reason must be %d characters or fewer", s.cfg.MaxReportLength))
	}

	report, err := s.db.CreateReport(ctx, db.CreateReportParams{
		WallpaperID: wallpaperID,
		ReporterID:  reporterID,
		Reason:      reason,
	})
	if err != nil {
		return nil, handler.Internal(fmt.Errorf("create report: %w", err))
	}
	return &report, nil
}

// Dismiss marks a report as dismissed.
func (s *ReportService) Dismiss(ctx context.Context, reportID, adminID string) error {
	if err := s.db.DismissReport(ctx, reportID); err != nil {
		return handler.Internal(fmt.Errorf("dismiss report: %w", err))
	}
	return nil
}
```

- [ ] Write thin tests for validation paths in `report_test.go`:

```go
package service_test

import (
	"context"
	"strings"
	"testing"

	"github.com/0x63616c/screenspace/server/internal/config"
	"github.com/0x63616c/screenspace/server/internal/handler"
	"github.com/0x63616c/screenspace/server/internal/service"
)

func TestReportService_Create_EmptyReason(t *testing.T) {
	t.Parallel()
	svc := service.NewReportService(&db.MockQuerier{}, config.DefaultConfig())
	_, err := svc.Create(context.Background(), "wp1", "u1", "")
	var appErr *handler.AppError
	if !errors.As(err, &appErr) || appErr.Status != 400 {
		t.Errorf("expected 400 for empty reason, got %v", err)
	}
}

func TestReportService_Create_TooLong(t *testing.T) {
	t.Parallel()
	cfg := config.DefaultConfig()
	svc := service.NewReportService(&db.MockQuerier{}, cfg)
	_, err := svc.Create(context.Background(), "wp1", "u1", strings.Repeat("x", cfg.MaxReportLength+1))
	var appErr *handler.AppError
	if !errors.As(err, &appErr) || appErr.Status != 400 {
		t.Errorf("expected 400 for too-long reason, got %v", err)
	}
}
```

- [ ] Run tests:
  ```bash
  cd server && go test ./internal/service/...
  ```

- [ ] Commit: `feat(service): add FavoriteService and ReportService`

---

### Task B3.3 — AuthService migration to internal/service

**Files:**
- Create: `server/internal/service/auth.go`

**Context:** `service/auth.go` exists at the top-level `server/service/`. Move it into `internal/service/` and update imports. The JWT expiry must come from config (currently hardcoded to 7d).

**Steps:**
- [ ] Write `server/internal/service/auth.go`:

```go
package service

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/0x63616c/screenspace/server/internal/config"
	"github.com/0x63616c/screenspace/server/internal/types"
)

// TokenClaims holds the validated claims from a JWT.
type TokenClaims struct {
	UserID string
	Role   types.UserRole
}

// AuthService handles password hashing and JWT operations.
type AuthService struct {
	secret []byte
	expiry time.Duration
	cost   int
}

func NewAuthService(cfg *config.Config) *AuthService {
	return &AuthService{
		secret: []byte(cfg.JWTSecret),
		expiry: cfg.JWTExpiry,
		cost:   cfg.BcryptCost,
	}
}

// HashPassword bcrypt-hashes a password using the configured cost.
func (a *AuthService) HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), a.cost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(hash), nil
}

// VerifyPassword returns true if the hash matches the password.
func (a *AuthService) VerifyPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// GenerateToken creates a signed JWT for the given user.
func (a *AuthService) GenerateToken(userID string, role types.UserRole) (string, error) {
	claims := jwt.MapClaims{
		"sub":  userID,
		"role": string(role),
		"exp":  time.Now().Add(a.expiry).Unix(),
		"iat":  time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(a.secret)
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}
	return signed, nil
}

// ValidateToken parses and validates a JWT, returning the claims.
func (a *AuthService) ValidateToken(tokenStr string) (*TokenClaims, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return a.secret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	sub, ok := claims["sub"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid token: missing sub")
	}
	roleStr, ok := claims["role"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid token: missing role")
	}

	return &TokenClaims{UserID: sub, Role: types.UserRole(roleStr)}, nil
}
```

- [ ] Commit: `feat(service): migrate AuthService to internal/service, expiry from config`

---

## B4: chi v5 Router + Middleware Reorganization

### Task B4.1 — Banned user cache middleware

**Files:**
- Create: `server/internal/middleware/banned.go`
- Create: `server/internal/middleware/banned_test.go`

**Steps:**
- [ ] Create `server/internal/middleware/` directory.
- [ ] Write `banned.go` with an in-memory TTL cache (sync.Map + expiry entries):

```go
package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/0x63616c/screenspace/server/db/generated"
	"github.com/0x63616c/screenspace/server/internal/respond"
)

const bannedCacheTTL = 60 * time.Second

type bannedCacheEntry struct {
	banned    bool
	expiresAt time.Time
}

// BannedCache is a thread-safe in-memory TTL cache for banned user status.
type BannedCache struct {
	mu      sync.Mutex
	entries map[string]bannedCacheEntry
}

func NewBannedCache() *BannedCache {
	return &BannedCache{entries: make(map[string]bannedCacheEntry)}
}

// IsBanned checks the cache. Returns (banned, ok) where ok=false means cache miss.
func (c *BannedCache) IsBanned(userID string) (bool, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.entries[userID]
	if !ok || time.Now().After(e.expiresAt) {
		delete(c.entries, userID)
		return false, false
	}
	return e.banned, true
}

// Set stores a banned status with TTL.
func (c *BannedCache) Set(userID string, banned bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[userID] = bannedCacheEntry{
		banned:    banned,
		expiresAt: time.Now().Add(bannedCacheTTL),
	}
}

// Evict removes a user from the cache immediately (call on ban action).
func (c *BannedCache) Evict(userID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, userID)
}

// BannedCheck returns middleware that checks if the authenticated user is banned.
// Uses the provided cache with 60s TTL; falls through to DB on cache miss.
func BannedCheck(q db.Querier, cache *BannedCache) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := ClaimsFromContext(r.Context())
			if claims == nil {
				next.ServeHTTP(w, r)
				return
			}

			// Cache hit.
			if banned, ok := cache.IsBanned(claims.UserID); ok {
				if banned {
					respond.Error(w, http.StatusForbidden, "banned", "account banned")
					return
				}
				next.ServeHTTP(w, r)
				return
			}

			// Cache miss: query DB.
			user, err := q.GetUserByID(r.Context(), claims.UserID)
			if err != nil || user.Banned {
				cache.Set(claims.UserID, true)
				respond.Error(w, http.StatusForbidden, "banned", "account banned")
				return
			}
			cache.Set(claims.UserID, false)
			next.ServeHTTP(w, r)
		})
	}
}
```

- [ ] Write `banned_test.go`:

```go
package middleware_test

import (
	"testing"
	"time"

	"github.com/0x63616c/screenspace/server/internal/middleware"
)

func TestBannedCache_SetAndGet(t *testing.T) {
	t.Parallel()
	c := middleware.NewBannedCache()
	c.Set("u1", true)

	banned, ok := c.IsBanned("u1")
	if !ok {
		t.Fatal("expected cache hit")
	}
	if !banned {
		t.Error("expected banned=true")
	}
}

func TestBannedCache_Evict(t *testing.T) {
	t.Parallel()
	c := middleware.NewBannedCache()
	c.Set("u1", true)
	c.Evict("u1")

	_, ok := c.IsBanned("u1")
	if ok {
		t.Error("expected cache miss after evict")
	}
}

func TestBannedCache_Miss(t *testing.T) {
	t.Parallel()
	c := middleware.NewBannedCache()
	_, ok := c.IsBanned("nobody")
	if ok {
		t.Error("expected cache miss for unknown user")
	}
}
```

- [ ] Run tests:
  ```bash
  cd server && go test ./internal/middleware/...
  ```

- [ ] Commit: `feat(middleware): add BannedCache with 60s TTL for per-request banned check`

---

### Task B4.2 — Auth middleware (internal/middleware/auth.go)

**Files:**
- Modify/Create: `server/internal/middleware/auth.go`

**Steps:**
- [ ] Write `auth.go` using the new `internal/service.AuthService` and typed `TokenClaims`:

```go
package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/0x63616c/screenspace/server/internal/respond"
	"github.com/0x63616c/screenspace/server/internal/service"
)

type contextKey string

const claimsKey contextKey = "claims"

// Auth validates the Bearer token and stores claims in context.
// Returns 401 if missing or invalid.
func Auth(auth *service.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if !strings.HasPrefix(header, "Bearer ") {
				respond.Error(w, http.StatusUnauthorized, "unauthorized", "missing or invalid authorization header")
				return
			}

			token := strings.TrimPrefix(header, "Bearer ")
			claims, err := auth.ValidateToken(token)
			if err != nil {
				respond.Error(w, http.StatusUnauthorized, "unauthorized", "invalid token")
				return
			}

			ctx := context.WithValue(r.Context(), claimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ClaimsFromContext retrieves TokenClaims from the request context.
func ClaimsFromContext(ctx context.Context) *service.TokenClaims {
	claims, _ := ctx.Value(claimsKey).(*service.TokenClaims)
	return claims
}

// Admin returns 403 if the authenticated user is not an admin.
func Admin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := ClaimsFromContext(r.Context())
		if claims == nil || claims.Role != types.RoleAdmin {
			respond.Error(w, http.StatusForbidden, "forbidden", "admin access required")
			return
		}
		next.ServeHTTP(w, r)
	})
}
```

> Note: Add `"github.com/0x63616c/screenspace/server/internal/types"` to imports for `types.RoleAdmin`.

- [ ] Commit: `feat(middleware): add Auth and Admin middleware using internal/service and respond`

---

### Task B4.3 — Rate limiter middleware (per-group, config-driven)

**Files:**
- Create: `server/internal/middleware/ratelimit.go`
- Create: `server/internal/middleware/ratelimit_test.go`

**Context:** Current `middleware/ratelimit.go` only supports per-day limits and hardcodes the key. New version supports configurable window (minute/hour/day) and separate per-IP vs per-user keying.

**Steps:**
- [ ] Write `ratelimit.go`:

```go
package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/0x63616c/screenspace/server/internal/respond"
)

type windowEntry struct {
	count     int
	windowEnd time.Time
}

// RateLimiter is a sliding-window rate limiter.
type RateLimiter struct {
	mu      sync.Mutex
	entries map[string]*windowEntry
	max     int
	window  time.Duration
}

// NewRateLimiter creates a rate limiter with max requests per window duration.
func NewRateLimiter(max int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		entries: make(map[string]*windowEntry),
		max:     max,
		window:  window,
	}
}

// Allow returns true if the key is within rate limits. Thread-safe.
func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	e, ok := rl.entries[key]
	if !ok || now.After(e.windowEnd) {
		rl.entries[key] = &windowEntry{count: 1, windowEnd: now.Add(rl.window)}
		return true
	}
	if e.count >= rl.max {
		return false
	}
	e.count++
	return true
}

// PerIP returns middleware that rate-limits by remote IP.
func (rl *RateLimiter) PerIP() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !rl.Allow(r.RemoteAddr) {
				w.Header().Set("Retry-After", "60")
				respond.Error(w, http.StatusTooManyRequests, "rate_limited", "too many requests")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// PerUser returns middleware that rate-limits by authenticated user ID.
// Falls back to IP if no claims in context.
func (rl *RateLimiter) PerUser() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := r.RemoteAddr
			if claims := ClaimsFromContext(r.Context()); claims != nil {
				key = claims.UserID
			}
			if !rl.Allow(key) {
				w.Header().Set("Retry-After", "60")
				respond.Error(w, http.StatusTooManyRequests, "rate_limited", "too many requests")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
```

- [ ] Write `ratelimit_test.go`:

```go
package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/0x63616c/screenspace/server/internal/middleware"
)

func TestRateLimiter_AllowsUnderLimit(t *testing.T) {
	t.Parallel()
	rl := middleware.NewRateLimiter(3, time.Minute)
	for i := range 3 {
		if !rl.Allow("key") {
			t.Errorf("request %d should be allowed", i)
		}
	}
}

func TestRateLimiter_BlocksOverLimit(t *testing.T) {
	t.Parallel()
	rl := middleware.NewRateLimiter(2, time.Minute)
	rl.Allow("key")
	rl.Allow("key")
	if rl.Allow("key") {
		t.Error("third request should be blocked")
	}
}

func TestRateLimiter_ResetsAfterWindow(t *testing.T) {
	t.Parallel()
	rl := middleware.NewRateLimiter(1, 10*time.Millisecond)
	rl.Allow("key")
	if rl.Allow("key") {
		t.Error("should be blocked before window expires")
	}
	time.Sleep(15 * time.Millisecond)
	if !rl.Allow("key") {
		t.Error("should be allowed after window expires")
	}
}

func TestRateLimiter_PerIP_Blocks(t *testing.T) {
	t.Parallel()
	rl := middleware.NewRateLimiter(1, time.Minute)
	handler := rl.PerIP()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for i, wantCode := range []int{http.StatusOK, http.StatusTooManyRequests} {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.RemoteAddr = "1.2.3.4:9999"
		handler.ServeHTTP(w, r)
		if w.Code != wantCode {
			t.Errorf("request %d: expected %d, got %d", i, wantCode, w.Code)
		}
	}
}
```

- [ ] Run tests:
  ```bash
  cd server && go test ./internal/middleware/...
  ```

- [ ] Commit: `feat(middleware): add configurable RateLimiter with PerIP and PerUser modes`

---

### Task B4.4 — Security headers middleware

**Files:**
- Create: `server/internal/middleware/security.go`

**Steps:**
- [ ] Write `security.go`:

```go
package middleware

import "net/http"

// SecurityHeaders adds security-relevant HTTP response headers.
// This should be applied to all route groups.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Content-Security-Policy", "default-src 'none'")
		// http.CrossOriginProtection: not needed for native app API-only usage (no
		// cookies, no browser forms). Enable here when browser clients are added.
		next.ServeHTTP(w, r)
	})
}

// MaxBodySize wraps the request body with a 1MB size limit.
// Apply to all JSON-body endpoints to prevent memory exhaustion.
func MaxBodySize(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1MB
		next.ServeHTTP(w, r)
	})
}
```

- [ ] Commit: `feat(middleware): add SecurityHeaders and MaxBodySize middleware`

---

## B5: chi v5 Router Wiring + HTTP Server Hardening

### Task B5.1 — Rewrite cmd/server/main.go

**Files:**
- Modify: `server/cmd/server/main.go`

**Context:** Current `main.go` is at the package root, uses `net/http.ServeMux`, `log.Fatalf`, no timeouts, no graceful shutdown, and hardcoded per-route middleware wrapping. This task rewrites it with chi v5, grouped middleware, config-driven rate limiters, signal-based graceful shutdown, and an optional debug server.

**Steps:**
- [ ] Write the complete new `cmd/server/main.go`:

```go
package main

import (
	"context"
	"log/slog"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"github.com/0x63616c/screenspace/server/db/generated"
	"github.com/0x63616c/screenspace/server/internal/config"
	apphandler "github.com/0x63616c/screenspace/server/internal/handler"
	"github.com/0x63616c/screenspace/server/internal/middleware"
	"github.com/0x63616c/screenspace/server/internal/service"
	"github.com/0x63616c/screenspace/server/internal/storage"
	"github.com/0x63616c/screenspace/server/internal/types"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("config", "error", err)
		os.Exit(1)
	}

	// Structured logger.
	level := slog.LevelInfo
	if cfg.LogLevel == "debug" {
		level = slog.LevelDebug
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})))

	// Database (pgxpool from Plan 1).
	pool, err := openDB(cfg)
	if err != nil {
		slog.Error("database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := runMigrations(pool); err != nil {
		slog.Error("migrations", "error", err)
		os.Exit(1)
	}

	// S3 storage.
	store, err := storage.NewS3Store(cfg.S3Endpoint, cfg.S3Bucket, cfg.S3AccessKey, cfg.S3SecretKey)
	if err != nil {
		slog.Error("storage", "error", err)
		os.Exit(1)
	}
	if err := store.EnsureBucket(context.Background()); err != nil {
		slog.Warn("could not ensure bucket", "error", err)
	}

	// sqlc querier.
	q := db.New(pool)

	// Services.
	authSvc := service.NewAuthService(cfg)
	wallpaperSvc := service.NewWallpaperService(q, store, service.NewVideoService(), cfg)
	favoriteSvc := service.NewFavoriteService(q)
	reportSvc := service.NewReportService(q, cfg)

	// Middleware.
	bannedCache := middleware.NewBannedCache()
	authMw := middleware.Auth(authSvc)
	bannedMw := middleware.BannedCheck(q, bannedCache)

	publicLimiter := middleware.NewRateLimiter(cfg.PublicRateLimit, time.Minute)
	authLimiter := middleware.NewRateLimiter(cfg.AuthRateLimit, time.Minute)
	userLimiter := middleware.NewRateLimiter(cfg.UserRateLimit, time.Minute)
	uploadLimiter := middleware.NewRateLimiter(cfg.UploadRateLimit, 24*time.Hour)
	downloadLimiter := middleware.NewRateLimiter(cfg.DownloadRateLimit, time.Hour)

	// Handlers.
	wallpaperH := apphandler.NewWallpaperHandler(q, store, wallpaperSvc, authSvc, cfg)
	authH := apphandler.NewAuthHandler(q, authSvc, bannedCache, cfg)
	favoriteH := apphandler.NewFavoriteHandler(favoriteSvc)
	reportH := apphandler.NewReportHandler(reportSvc)
	adminH := apphandler.NewAdminHandler(q, store, wallpaperSvc, bannedCache, cfg)

	// Router.
	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Recoverer)
	r.Use(chimw.CleanPath)
	r.Use(middleware.SecurityHeaders)
	r.Use(middleware.MaxBodySize)

	r.Route("/api/v1", func(r chi.Router) {
		// Public routes.
		r.Group(func(r chi.Router) {
			r.Use(publicLimiter.PerIP())
			r.Get("/health", apphandler.Wrap(wallpaperH.Health))
			r.Get("/categories", apphandler.Wrap(apphandler.ListCategories))
			r.Get("/wallpapers", apphandler.Wrap(wallpaperH.List))
			r.Get("/wallpapers/popular", apphandler.Wrap(wallpaperH.Popular))
			r.Get("/wallpapers/recent", apphandler.Wrap(wallpaperH.Recent))
			r.Get("/wallpapers/{id}", apphandler.Wrap(wallpaperH.Get))
		})

		// Auth endpoints (no JWT required, IP-limited).
		r.Group(func(r chi.Router) {
			r.Use(authLimiter.PerIP())
			r.Post("/auth/register", apphandler.Wrap(authH.Register))
			r.Post("/auth/login", apphandler.Wrap(authH.Login))
		})

		// Authenticated routes.
		r.Group(func(r chi.Router) {
			r.Use(authMw)
			r.Use(bannedMw)
			r.Use(userLimiter.PerUser())
			r.Get("/auth/me", apphandler.Wrap(authH.Me))
			r.Post("/wallpapers", apphandler.Wrap(func(w http.ResponseWriter, r *http.Request) error {
				if !uploadLimiter.Allow(middleware.ClaimsFromContext(r.Context()).UserID) {
					return &apphandler.AppError{Status: http.StatusTooManyRequests, Code: "rate_limited", Message: "upload limit reached"}
				}
				return wallpaperH.Create(w, r)
			}))
			r.Post("/wallpapers/{id}/finalize", apphandler.Wrap(wallpaperH.Finalize))
			r.Post("/wallpapers/{id}/download", apphandler.Wrap(func(w http.ResponseWriter, r *http.Request) error {
				if !downloadLimiter.Allow(middleware.ClaimsFromContext(r.Context()).UserID) {
					return &apphandler.AppError{Status: http.StatusTooManyRequests, Code: "rate_limited", Message: "download limit reached"}
				}
				return wallpaperH.Download(w, r)
			}))
			r.Post("/wallpapers/{id}/favorite", apphandler.Wrap(favoriteH.Toggle))
			r.Post("/wallpapers/{id}/report", apphandler.Wrap(reportH.Create))
		})

		// Admin routes.
		r.Group(func(r chi.Router) {
			r.Use(authMw)
			r.Use(bannedMw)
			r.Use(middleware.Admin)
			r.Get("/admin/queue", apphandler.Wrap(adminH.Queue))
			r.Post("/admin/queue/{id}/approve", apphandler.Wrap(adminH.Approve))
			r.Post("/admin/queue/{id}/reject", apphandler.Wrap(adminH.Reject))
			r.Get("/admin/wallpapers", apphandler.Wrap(adminH.ListWallpapers))
			r.Patch("/admin/wallpapers/{id}", apphandler.Wrap(adminH.EditWallpaper))
			r.Get("/admin/users", apphandler.Wrap(adminH.ListUsers))
			r.Post("/admin/users/{id}/ban", apphandler.Wrap(adminH.BanUser))
			r.Post("/admin/users/{id}/unban", apphandler.Wrap(adminH.UnbanUser))
			r.Post("/admin/users/{id}/promote", apphandler.Wrap(adminH.PromoteUser))
			r.Get("/admin/reports", apphandler.Wrap(adminH.ListReports))
			r.Post("/admin/reports/{id}/dismiss", apphandler.Wrap(adminH.DismissReport))
		})
	})

	// Optional debug server (localhost only, never 0.0.0.0).
	if cfg.DebugEnabled {
		var wg sync.WaitGroup
		wg.Go(func() {
			debugMux := http.NewServeMux()
			debugMux.HandleFunc("/debug/pprof/", pprof.Index)
			debugMux.HandleFunc("/debug/pprof/profile", pprof.Profile)
			slog.Info("debug server listening", "addr", "127.0.0.1:6060")
			if err := http.ListenAndServe("127.0.0.1:6060", debugMux); err != nil {
				slog.Error("debug server", "error", err)
			}
		})
	}

	// HTTP server with hardened timeouts.
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Graceful shutdown via signal context (Go 1.26: ctx.Err() reports which signal).
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		slog.Info("server listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down", "signal", ctx.Err())

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown", "error", err)
	}
	slog.Info("shutdown complete")
}
```

> Note: `openDB` and `runMigrations` are helpers defined in `cmd/server/db.go` (from Plan 1). Add `"sync"` and `"net/http/pprof"` to imports. The upload/download per-endpoint inline limiters can be refactored to sub-middleware in a follow-up.

- [ ] Run build to verify wiring compiles:
  ```bash
  cd server && go build ./cmd/server/...
  ```
  Expected: exits 0, binary produced.

- [ ] Commit: `feat(server): rewrite main.go with chi v5, grouped middleware, graceful shutdown`

---

## B6: Handler Rewrites (thin adapters)

### Task B6.1 — Rewrite wallpaper handler

**Files:**
- Create: `server/internal/handler/wallpaper.go`

**Context:** Replace the 494-line `handler/wallpaper.go` with thin adapters. All business logic (finalize, status checks, ownership) moves to `WallpaperService`. `chi.URLParam` replaces `r.PathValue`. `respond.*` replaces all `http.Error`. `Popular`/`Recent` no longer mutate `r.URL`.

**Steps:**
- [ ] Write `server/internal/handler/wallpaper.go`:

```go
package handler

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/0x63616c/screenspace/server/db/generated"
	"github.com/0x63616c/screenspace/server/internal/config"
	"github.com/0x63616c/screenspace/server/internal/middleware"
	"github.com/0x63616c/screenspace/server/internal/respond"
	"github.com/0x63616c/screenspace/server/internal/service"
	"github.com/0x63616c/screenspace/server/internal/storage"
	"github.com/0x63616c/screenspace/server/internal/types"
)

// WallpaperHandler handles HTTP requests for the wallpaper resource.
type WallpaperHandler struct {
	q     db.Querier
	store storage.Store
	svc   *service.WallpaperService
	auth  *service.AuthService
	cfg   *config.Config
}

func NewWallpaperHandler(q db.Querier, s storage.Store, svc *service.WallpaperService, auth *service.AuthService, cfg *config.Config) *WallpaperHandler {
	return &WallpaperHandler{q: q, store: s, svc: svc, auth: auth, cfg: cfg}
}

func (h *WallpaperHandler) Health(w http.ResponseWriter, r *http.Request) error {
	if err := h.q.(interface{ Ping(context.Context) error }).Ping(r.Context()); err != nil {
		return &AppError{Status: 503, Code: "db_unavailable", Message: "database unavailable", Err: err}
	}
	return respond.JSON(w, http.StatusOK, map[string]string{"status": "ok", "db": "ok"})
}

func ListCategories(w http.ResponseWriter, r *http.Request) error {
	cats := types.AllCategories()
	strs := make([]string, len(cats))
	for i, c := range cats {
		strs[i] = string(c)
	}
	return respond.JSON(w, http.StatusOK, strs)
}

func (h *WallpaperHandler) Get(w http.ResponseWriter, r *http.Request) error {
	id := chi.URLParam(r, "id")
	wp, err := h.svc.GetApproved(r.Context(), id)
	if err != nil {
		return err
	}
	return respond.JSON(w, http.StatusOK, h.toResponseWithURLs(r, wp))
}

func (h *WallpaperHandler) List(w http.ResponseWriter, r *http.Request) error {
	return h.list(w, r, "")
}

func (h *WallpaperHandler) Popular(w http.ResponseWriter, r *http.Request) error {
	return h.list(w, r, string(types.SortPopular))
}

func (h *WallpaperHandler) Recent(w http.ResponseWriter, r *http.Request) error {
	return h.list(w, r, string(types.SortRecent))
}

func (h *WallpaperHandler) list(w http.ResponseWriter, r *http.Request, forceSort string) error {
	q := r.URL.Query()
	pg := respond.ParsePagination(q, h.cfg.DefaultPageSize, h.cfg.MaxPageSize)

	sort := forceSort
	if sort == "" {
		s := q.Get("sort")
		if types.SortOrder(s).Valid() {
			sort = s
		} else {
			sort = string(types.SortRecent)
		}
	}

	wallpapers, total, err := h.q.ListWallpapers(r.Context(), db.ListWallpapersParams{
		Status:   string(types.StatusApproved),
		Category: q.Get("category"),
		Query:    q.Get("q"),
		Sort:     sort,
		Limit:    int32(pg.Limit),
		Offset:   int32(pg.Offset),
	})
	if err != nil {
		return Internal(fmt.Errorf("list wallpapers: %w", err))
	}

	items := make([]any, 0, len(wallpapers))
	for i := range wallpapers {
		items = append(items, h.toResponseWithURLs(r, &wallpapers[i]))
	}
	return respond.Paginated(w, items, int(total), pg.Limit, pg.Offset)
}

func (h *WallpaperHandler) Create(w http.ResponseWriter, r *http.Request) error {
	claims := middleware.ClaimsFromContext(r.Context())

	var req struct {
		Title    string   `json:"title"`
		Category string   `json:"category"`
		Tags     []string `json:"tags"`
	}
	if err := decodeJSON(r, &req); err != nil {
		return BadRequest("invalid request body")
	}
	if req.Title == "" {
		return BadRequest("title is required")
	}
	if len(req.Title) > h.cfg.MaxTitleLength {
		return BadRequest(fmt.Sprintf("title must be %d characters or fewer", h.cfg.MaxTitleLength))
	}
	if len(req.Tags) > h.cfg.MaxTagCount {
		return BadRequest(fmt.Sprintf("maximum %d tags allowed", h.cfg.MaxTagCount))
	}
	for _, tag := range req.Tags {
		if len(tag) > h.cfg.MaxTagLength {
			return BadRequest(fmt.Sprintf("each tag must be %d characters or fewer", h.cfg.MaxTagLength))
		}
	}
	if req.Category != "" && !types.Category(req.Category).Valid() {
		return BadRequest("invalid category")
	}

	wp, err := h.q.CreateWallpaper(r.Context(), db.CreateWallpaperParams{
		Title:      req.Title,
		UploaderID: claims.UserID,
		Category:   req.Category,
		Tags:       req.Tags,
		StorageKey: fmt.Sprintf("wallpapers/pending/original.mp4"),
		Status:     string(types.StatusPending),
	})
	if err != nil {
		return Internal(fmt.Errorf("create wallpaper: %w", err))
	}

	// Update key to use real ID.
	actualKey := fmt.Sprintf("wallpapers/%s/original.mp4", wp.ID)
	if err := h.q.UpdateStorageKey(r.Context(), db.UpdateStorageKeyParams{
		ID:         wp.ID,
		StorageKey: actualKey,
	}); err != nil {
		return Internal(fmt.Errorf("update storage key: %w", err))
	}

	uploadURL, err := h.store.PreSignedUploadURL(r.Context(), actualKey, h.cfg.PresignedUploadExpiry)
	if err != nil {
		return Internal(fmt.Errorf("presign upload url: %w", err))
	}

	return respond.JSON(w, http.StatusCreated, map[string]string{
		"id":         wp.ID,
		"upload_url": uploadURL,
	})
}

func (h *WallpaperHandler) Finalize(w http.ResponseWriter, r *http.Request) error {
	claims := middleware.ClaimsFromContext(r.Context())
	id := chi.URLParam(r, "id")

	wp, err := h.svc.Finalize(r.Context(), id, claims.UserID)
	if err != nil {
		return err
	}
	return respond.JSON(w, http.StatusOK, map[string]string{"status": string(types.WallpaperStatus(wp.Status))})
}

func (h *WallpaperHandler) Download(w http.ResponseWriter, r *http.Request) error {
	id := chi.URLParam(r, "id")
	wp, err := h.svc.GetApproved(r.Context(), id)
	if err != nil {
		return err
	}

	url, err := h.store.PreSignedURL(r.Context(), wp.StorageKey, h.cfg.PresignedDownloadExpiry)
	if err != nil {
		return Internal(fmt.Errorf("presign download url: %w", err))
	}

	if err := h.q.IncrementDownloadCount(r.Context(), wp.ID); err != nil {
		slog.Error("increment download count", "wallpaper_id", wp.ID, "error", err)
	}

	return respond.JSON(w, http.StatusOK, map[string]string{"download_url": url})
}

func (h *WallpaperHandler) toResponseWithURLs(r *http.Request, wp *db.Wallpaper) map[string]any {
	resp := map[string]any{
		"id":               wp.ID,
		"title":            wp.Title,
		"uploader_id":      wp.UploaderID,
		"status":           wp.Status,
		"category":         wp.Category,
		"tags":             wp.Tags,
		"resolution":       wp.Resolution,
		"width":            wp.Width,
		"height":           wp.Height,
		"duration":         wp.Duration,
		"file_size":        wp.FileSize,
		"format":           wp.Format,
		"download_count":   wp.DownloadCount,
		"rejection_reason": wp.RejectionReason,
		"created_at":       wp.CreatedAt.Format(time.RFC3339),
		"updated_at":       wp.UpdatedAt.Format(time.RFC3339),
	}
	if wp.ThumbnailKey != "" {
		if url, err := h.store.PreSignedURL(r.Context(), wp.ThumbnailKey, h.cfg.PresignedDownloadExpiry); err == nil {
			resp["thumbnail_url"] = url
		}
	}
	if wp.PreviewKey != "" {
		if url, err := h.store.PreSignedURL(r.Context(), wp.PreviewKey, h.cfg.PresignedDownloadExpiry); err == nil {
			resp["preview_url"] = url
		}
	}
	return resp
}
```

> Note: `decodeJSON` is a shared helper — define in `server/internal/handler/util.go`:
> ```go
> package handler
> import ("encoding/json"; "net/http")
> func decodeJSON(r *http.Request, v any) error {
>     return json.NewDecoder(r.Body).Decode(v)
> }
> ```

- [ ] Commit: `refactor(handler): rewrite wallpaper handler as thin adapter using service layer`

---

### Task B6.2 — Rewrite auth handler

**Files:**
- Create: `server/internal/handler/auth.go`

**Steps:**
- [ ] Write `server/internal/handler/auth.go`. Key changes from current code:
  - Password minimum length from config.
  - Basic email format validation.
  - Generic 409 "registration failed" (not "email already exists").
  - Duplicate detection uses pgx error code `23505` via `errors.As` (not string matching).
  - `bcrypt` cost from config via `AuthService`.

```go
package handler

import (
	"errors"
	"net/http"
	"regexp"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/0x63616c/screenspace/server/db/generated"
	"github.com/0x63616c/screenspace/server/internal/config"
	"github.com/0x63616c/screenspace/server/internal/middleware"
	"github.com/0x63616c/screenspace/server/internal/respond"
	"github.com/0x63616c/screenspace/server/internal/service"
	"github.com/0x63616c/screenspace/server/internal/types"
)

var emailRe = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

// AuthHandler handles registration, login, and token validation endpoints.
type AuthHandler struct {
	q           db.Querier
	auth        *service.AuthService
	bannedCache *middleware.BannedCache
	cfg         *config.Config
}

func NewAuthHandler(q db.Querier, auth *service.AuthService, cache *middleware.BannedCache, cfg *config.Config) *AuthHandler {
	return &AuthHandler{q: q, auth: auth, bannedCache: cache, cfg: cfg}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) error {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := decodeJSON(r, &req); err != nil {
		return BadRequest("invalid request body")
	}
	if req.Email == "" || req.Password == "" {
		return BadRequest("email and password are required")
	}
	if !emailRe.MatchString(req.Email) {
		return BadRequest("invalid email format")
	}
	if len(req.Password) < h.cfg.MinPasswordLen {
		return BadRequest(fmt.Sprintf("password must be at least %d characters", h.cfg.MinPasswordLen))
	}

	hash, err := h.auth.HashPassword(req.Password)
	if err != nil {
		return Internal(fmt.Errorf("hash password: %w", err))
	}

	role := types.RoleUser
	if req.Email == h.cfg.AdminEmail {
		role = types.RoleAdmin
	}

	user, err := h.q.CreateUser(r.Context(), db.CreateUserParams{
		Email:        req.Email,
		PasswordHash: hash,
		Role:         string(role),
	})
	if err != nil {
		// Use pgx error code 23505 (unique_violation) instead of string matching.
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return &AppError{Status: http.StatusConflict, Code: "conflict", Message: "registration failed"}
		}
		return Internal(fmt.Errorf("create user: %w", err))
	}

	token, err := h.auth.GenerateToken(user.ID, types.UserRole(user.Role))
	if err != nil {
		return Internal(fmt.Errorf("generate token: %w", err))
	}

	return respond.JSON(w, http.StatusCreated, map[string]string{
		"token": token,
		"role":  user.Role,
	})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) error {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := decodeJSON(r, &req); err != nil {
		return BadRequest("invalid request body")
	}
	if req.Email == "" || req.Password == "" {
		return BadRequest("email and password are required")
	}

	user, err := h.q.GetUserByEmail(r.Context(), req.Email)
	if err != nil {
		return &AppError{Status: http.StatusUnauthorized, Code: "unauthorized", Message: "invalid credentials"}
	}

	if user.Banned {
		return &AppError{Status: http.StatusForbidden, Code: "banned", Message: "account banned"}
	}

	if !h.auth.VerifyPassword(user.PasswordHash, req.Password) {
		return &AppError{Status: http.StatusUnauthorized, Code: "unauthorized", Message: "invalid credentials"}
	}

	token, err := h.auth.GenerateToken(user.ID, types.UserRole(user.Role))
	if err != nil {
		return Internal(fmt.Errorf("generate token: %w", err))
	}

	return respond.JSON(w, http.StatusOK, map[string]string{
		"token": token,
		"role":  user.Role,
	})
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) error {
	claims := middleware.ClaimsFromContext(r.Context())
	user, err := h.q.GetUserByID(r.Context(), claims.UserID)
	if err != nil {
		return NotFound("user not found")
	}
	return respond.JSON(w, http.StatusOK, map[string]string{
		"id":    user.ID,
		"email": user.Email,
		"role":  user.Role,
	})
}
```

> Note: Add `"fmt"` import.

- [ ] Commit: `refactor(handler): rewrite auth handler with pgx error codes, config-driven password validation`

---

### Task B6.3 — Rewrite admin handler

**Files:**
- Create: `server/internal/handler/admin.go`

**Key changes:** Remove `requireAdmin()` calls (handled by middleware). Use `chi.URLParam`. Use `respond.Paginated`. Use service layer for approve/reject (status transitions enforced). `PromoteUser` checks banned status. Cache eviction on ban.

**Steps:**
- [ ] Write `server/internal/handler/admin.go`:

```go
package handler

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/0x63616c/screenspace/server/db/generated"
	"github.com/0x63616c/screenspace/server/internal/config"
	"github.com/0x63616c/screenspace/server/internal/middleware"
	"github.com/0x63616c/screenspace/server/internal/respond"
	"github.com/0x63616c/screenspace/server/internal/service"
	"github.com/0x63616c/screenspace/server/internal/storage"
	"github.com/0x63616c/screenspace/server/internal/types"
)

// AdminHandler handles admin-only HTTP endpoints.
type AdminHandler struct {
	q           db.Querier
	store       storage.Store
	wallpaperSvc *service.WallpaperService
	bannedCache *middleware.BannedCache
	cfg         *config.Config
}

func NewAdminHandler(q db.Querier, s storage.Store, svc *service.WallpaperService, cache *middleware.BannedCache, cfg *config.Config) *AdminHandler {
	return &AdminHandler{q: q, store: s, wallpaperSvc: svc, bannedCache: cache, cfg: cfg}
}

func (h *AdminHandler) Queue(w http.ResponseWriter, r *http.Request) error {
	pg := respond.ParsePagination(r.URL.Query(), h.cfg.DefaultPageSize, h.cfg.MaxPageSize)
	wallpapers, total, err := h.q.ListWallpapersWithStatus(r.Context(), db.ListWallpapersWithStatusParams{
		Status: string(types.StatusPendingReview),
		Limit:  int32(pg.Limit),
		Offset: int32(pg.Offset),
	})
	if err != nil {
		return Internal(fmt.Errorf("list queue: %w", err))
	}
	items := make([]any, len(wallpapers))
	for i := range wallpapers {
		items[i] = wallpaperToMap(&wallpapers[i])
	}
	return respond.Paginated(w, items, int(total), pg.Limit, pg.Offset)
}

func (h *AdminHandler) Approve(w http.ResponseWriter, r *http.Request) error {
	claims := middleware.ClaimsFromContext(r.Context())
	id := chi.URLParam(r, "id")
	if err := h.wallpaperSvc.Approve(r.Context(), id, claims.UserID); err != nil {
		return err
	}
	return respond.JSON(w, http.StatusOK, map[string]string{"status": "approved"})
}

func (h *AdminHandler) Reject(w http.ResponseWriter, r *http.Request) error {
	claims := middleware.ClaimsFromContext(r.Context())
	id := chi.URLParam(r, "id")
	var req struct {
		Reason string `json:"reason"`
	}
	if err := decodeJSON(r, &req); err != nil {
		return BadRequest("invalid request body")
	}
	if err := h.wallpaperSvc.Reject(r.Context(), id, claims.UserID, req.Reason); err != nil {
		return err
	}
	return respond.JSON(w, http.StatusOK, map[string]string{"status": "rejected"})
}

func (h *AdminHandler) ListWallpapers(w http.ResponseWriter, r *http.Request) error {
	q := r.URL.Query()
	pg := respond.ParsePagination(q, h.cfg.DefaultPageSize, h.cfg.MaxPageSize)
	status := q.Get("status")
	if status == "" {
		status = string(types.StatusApproved)
	}
	wallpapers, total, err := h.q.ListWallpapersWithStatus(r.Context(), db.ListWallpapersWithStatusParams{
		Status: status,
		Limit:  int32(pg.Limit),
		Offset: int32(pg.Offset),
	})
	if err != nil {
		return Internal(fmt.Errorf("list wallpapers: %w", err))
	}
	items := make([]any, len(wallpapers))
	for i := range wallpapers {
		items[i] = wallpaperToMap(&wallpapers[i])
	}
	return respond.Paginated(w, items, int(total), pg.Limit, pg.Offset)
}

func (h *AdminHandler) EditWallpaper(w http.ResponseWriter, r *http.Request) error {
	id := chi.URLParam(r, "id")
	var req struct {
		Title    string   `json:"title"`
		Category string   `json:"category"`
		Tags     []string `json:"tags"`
	}
	if err := decodeJSON(r, &req); err != nil {
		return BadRequest("invalid request body")
	}
	tags := req.Tags
	if tags == nil {
		tags = []string{}
	}
	if err := h.q.UpdateWallpaperMetadata(r.Context(), db.UpdateWallpaperMetadataParams{
		ID:       id,
		Title:    req.Title,
		Category: req.Category,
		Tags:     tags,
	}); err != nil {
		return Internal(fmt.Errorf("update metadata: %w", err))
	}
	return respond.JSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (h *AdminHandler) ListUsers(w http.ResponseWriter, r *http.Request) error {
	q := r.URL.Query()
	pg := respond.ParsePagination(q, h.cfg.DefaultPageSize, h.cfg.MaxPageSize)
	users, total, err := h.q.ListUsersWithSearch(r.Context(), db.ListUsersWithSearchParams{
		Query:  q.Get("q"),
		Limit:  int32(pg.Limit),
		Offset: int32(pg.Offset),
	})
	if err != nil {
		return Internal(fmt.Errorf("list users: %w", err))
	}
	items := make([]any, len(users))
	for i, u := range users {
		items[i] = map[string]any{
			"id":         u.ID,
			"email":      u.Email,
			"role":       u.Role,
			"banned":     u.Banned,
			"created_at": u.CreatedAt.Format(time.RFC3339),
		}
	}
	return respond.Paginated(w, items, int(total), pg.Limit, pg.Offset)
}

func (h *AdminHandler) BanUser(w http.ResponseWriter, r *http.Request) error {
	claims := middleware.ClaimsFromContext(r.Context())
	id := chi.URLParam(r, "id")
	if _, err := h.q.GetUserByID(r.Context(), id); err != nil {
		return NotFound("user not found")
	}
	if err := h.q.SetBanned(r.Context(), db.SetBannedParams{ID: id, Banned: true}); err != nil {
		return Internal(fmt.Errorf("ban user: %w", err))
	}
	// Evict from banned cache so the per-request check takes effect immediately.
	h.bannedCache.Evict(id)
	slog.Info("user banned", "admin_id", claims.UserID, "target_user_id", id)
	return respond.JSON(w, http.StatusOK, map[string]string{"status": "banned"})
}

func (h *AdminHandler) UnbanUser(w http.ResponseWriter, r *http.Request) error {
	claims := middleware.ClaimsFromContext(r.Context())
	id := chi.URLParam(r, "id")
	if _, err := h.q.GetUserByID(r.Context(), id); err != nil {
		return NotFound("user not found")
	}
	if err := h.q.SetBanned(r.Context(), db.SetBannedParams{ID: id, Banned: false}); err != nil {
		return Internal(fmt.Errorf("unban user: %w", err))
	}
	h.bannedCache.Evict(id)
	slog.Info("user unbanned", "admin_id", claims.UserID, "target_user_id", id)
	return respond.JSON(w, http.StatusOK, map[string]string{"status": "unbanned"})
}

func (h *AdminHandler) PromoteUser(w http.ResponseWriter, r *http.Request) error {
	claims := middleware.ClaimsFromContext(r.Context())
	id := chi.URLParam(r, "id")
	user, err := h.q.GetUserByID(r.Context(), id)
	if err != nil {
		return NotFound("user not found")
	}
	if user.Banned {
		return BadRequest("cannot promote a banned user")
	}
	if types.UserRole(user.Role) == types.RoleAdmin {
		// Idempotent: already admin.
		return respond.JSON(w, http.StatusOK, map[string]string{"status": "promoted"})
	}
	if err := h.q.SetRole(r.Context(), db.SetRoleParams{ID: id, Role: string(types.RoleAdmin)}); err != nil {
		return Internal(fmt.Errorf("promote user: %w", err))
	}
	slog.Info("user promoted", "admin_id", claims.UserID, "target_user_id", id)
	return respond.JSON(w, http.StatusOK, map[string]string{"status": "promoted"})
}

func (h *AdminHandler) ListReports(w http.ResponseWriter, r *http.Request) error {
	pg := respond.ParsePagination(r.URL.Query(), h.cfg.DefaultPageSize, h.cfg.MaxPageSize)
	reports, total, err := h.q.ListPendingReports(r.Context(), db.ListPendingReportsParams{
		Limit:  int32(pg.Limit),
		Offset: int32(pg.Offset),
	})
	if err != nil {
		return Internal(fmt.Errorf("list reports: %w", err))
	}
	items := make([]any, len(reports))
	for i, rpt := range reports {
		items[i] = map[string]any{
			"id":           rpt.ID,
			"wallpaper_id": rpt.WallpaperID,
			"reporter_id":  rpt.ReporterID,
			"reason":       rpt.Reason,
			"status":       rpt.Status,
			"created_at":   rpt.CreatedAt.Format(time.RFC3339),
		}
	}
	return respond.Paginated(w, items, int(total), pg.Limit, pg.Offset)
}

func (h *AdminHandler) DismissReport(w http.ResponseWriter, r *http.Request) error {
	claims := middleware.ClaimsFromContext(r.Context())
	id := chi.URLParam(r, "id")
	if err := h.q.DismissReport(r.Context(), id); err != nil {
		return Internal(fmt.Errorf("dismiss report: %w", err))
	}
	slog.Info("report dismissed", "admin_id", claims.UserID, "report_id", id)
	return respond.JSON(w, http.StatusOK, map[string]string{"status": "dismissed"})
}

func wallpaperToMap(wp *db.Wallpaper) map[string]any {
	return map[string]any{
		"id":               wp.ID,
		"title":            wp.Title,
		"uploader_id":      wp.UploaderID,
		"status":           wp.Status,
		"category":         wp.Category,
		"tags":             wp.Tags,
		"resolution":       wp.Resolution,
		"width":            wp.Width,
		"height":           wp.Height,
		"duration":         wp.Duration,
		"file_size":        wp.FileSize,
		"format":           wp.Format,
		"download_count":   wp.DownloadCount,
		"rejection_reason": wp.RejectionReason,
		"created_at":       wp.CreatedAt.Format(time.RFC3339),
		"updated_at":       wp.UpdatedAt.Format(time.RFC3339),
	}
}
```

- [ ] Commit: `refactor(handler): rewrite admin handler using service layer, respond package, chi URL params`

---

### Task B6.4 — Rewrite favorite + report handlers

**Files:**
- Create: `server/internal/handler/favorite.go`
- Create: `server/internal/handler/report.go`

**Steps:**
- [ ] Write `favorite.go`:

```go
package handler

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/0x63616c/screenspace/server/internal/config"
	"github.com/0x63616c/screenspace/server/internal/middleware"
	"github.com/0x63616c/screenspace/server/internal/respond"
	"github.com/0x63616c/screenspace/server/internal/service"
)

// FavoriteHandler handles favorite toggle and listing.
type FavoriteHandler struct {
	svc *service.FavoriteService
	cfg *config.Config
}

func NewFavoriteHandler(svc *service.FavoriteService) *FavoriteHandler {
	return &FavoriteHandler{svc: svc}
}

func (h *FavoriteHandler) Toggle(w http.ResponseWriter, r *http.Request) error {
	claims := middleware.ClaimsFromContext(r.Context())
	wallpaperID := chi.URLParam(r, "id")
	favorited, err := h.svc.Toggle(r.Context(), claims.UserID, wallpaperID)
	if err != nil {
		return err
	}
	return respond.JSON(w, http.StatusOK, map[string]bool{"favorited": favorited})
}
```

- [ ] Write `report.go`:

```go
package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/0x63616c/screenspace/server/internal/middleware"
	"github.com/0x63616c/screenspace/server/internal/respond"
	"github.com/0x63616c/screenspace/server/internal/service"
)

// ReportHandler handles wallpaper reports.
type ReportHandler struct {
	svc *service.ReportService
}

func NewReportHandler(svc *service.ReportService) *ReportHandler {
	return &ReportHandler{svc: svc}
}

func (h *ReportHandler) Create(w http.ResponseWriter, r *http.Request) error {
	claims := middleware.ClaimsFromContext(r.Context())
	wallpaperID := chi.URLParam(r, "id")

	var req struct {
		Reason string `json:"reason"`
	}
	if err := decodeJSON(r, &req); err != nil {
		return BadRequest("invalid request body")
	}

	report, err := h.svc.Create(r.Context(), wallpaperID, claims.UserID, req.Reason)
	if err != nil {
		return err
	}

	return respond.JSON(w, http.StatusCreated, map[string]any{
		"id":           report.ID,
		"wallpaper_id": report.WallpaperID,
		"reporter_id":  report.ReporterID,
		"reason":       report.Reason,
		"status":       report.Status,
		"created_at":   report.CreatedAt,
	})
}
```

- [ ] Commit: `refactor(handler): rewrite favorite and report handlers as thin adapters`

---

## B7: slog Consistency + Config Additions

### Task B7.1 — Add missing config fields + DebugEnabled

**Files:**
- Modify: `server/internal/config/config.go`

**Context:** Plan 1 creates the centralized Config struct. This task ensures the fields needed for Plan 2 exist: `DebugEnabled`, `BcryptCost`, `MinPasswordLen`, `MaxReportLength`, `MaxTagCount`, `MaxTagLength`, `MaxTitleLength`, `PresignedUploadExpiry`, `PresignedDownloadExpiry`, `DefaultPageSize`, `MaxPageSize`, `ShutdownTimeout`, and all rate limit fields. Also adds `DefaultConfig()` for test use.

**Steps:**
- [ ] Verify these fields exist in `internal/config/config.go`. Add any that are absent:

```go
// Security
BcryptCost     int           // BCRYPT_COST (default 10)
MinPasswordLen int           // MIN_PASSWORD_LENGTH (default 8)

// Rate limits
AuthRateLimit     int // AUTH_RATE_LIMIT (default 10)
PublicRateLimit   int // PUBLIC_RATE_LIMIT (default 120)
UserRateLimit     int // USER_RATE_LIMIT (default 30)
UploadRateLimit   int // UPLOAD_RATE_LIMIT (default 5)
DownloadRateLimit int // DOWNLOAD_RATE_LIMIT (default 60)

// Upload constraints
MaxFileSize      int64   // MAX_FILE_SIZE (default 200*1024*1024)
MaxDuration      float64 // MAX_DURATION (default 60)
MinHeight        int     // MIN_HEIGHT (default 1080)
MaxTitleLength   int     // MAX_TITLE_LENGTH (default 255)
MaxTagCount      int     // MAX_TAG_COUNT (default 10)
MaxTagLength     int     // MAX_TAG_LENGTH (default 50)
MaxReportLength  int     // MAX_REPORT_LENGTH (default 500)

// Pagination
DefaultPageSize int // DEFAULT_PAGE_SIZE (default 20)
MaxPageSize     int // MAX_PAGE_SIZE (default 100)

// URLs
PresignedDownloadExpiry time.Duration // PRESIGNED_DOWNLOAD_EXPIRY (default 1h)
PresignedUploadExpiry   time.Duration // PRESIGNED_UPLOAD_EXPIRY (default 2h)

// Server
ShutdownTimeout time.Duration // SHUTDOWN_TIMEOUT (default 25s)
DebugEnabled    bool          // DEBUG_ENABLED (default false)
LogLevel        string        // LOG_LEVEL (default "info")
```

- [ ] Add `DefaultConfig()` function returning a config populated with all defaults (for use in service tests):

```go
func DefaultConfig() *Config {
	return &Config{
		Port:                   "8080",
		ShutdownTimeout:        25 * time.Second,
		LogLevel:               "info",
		BcryptCost:             10,
		MinPasswordLen:         8,
		JWTExpiry:              7 * 24 * time.Hour,
		AuthRateLimit:          10,
		PublicRateLimit:        120,
		UserRateLimit:          30,
		UploadRateLimit:        5,
		DownloadRateLimit:      60,
		MaxFileSize:            200 * 1024 * 1024,
		MaxDuration:            60,
		MinHeight:              1080,
		MaxTitleLength:         255,
		MaxTagCount:            10,
		MaxTagLength:           50,
		MaxReportLength:        500,
		DefaultPageSize:        20,
		MaxPageSize:            100,
		PresignedDownloadExpiry: time.Hour,
		PresignedUploadExpiry:  2 * time.Hour,
		S3Bucket:               "screenspace",
		DBMaxConns:             25,
		DBMinConns:             5,
		DBMaxConnLifetime:      5 * time.Minute,
		DBHealthCheckPeriod:    30 * time.Second,
	}
}
```

- [ ] Commit: `feat(config): add all Plan 2 fields with defaults, add DefaultConfig() for tests`

---

## B8: Verification

### Task B8.1 — Full build and test pass

**Steps:**
- [ ] Run full build:
  ```bash
  cd server && go build ./...
  ```
  Expected: exits 0.

- [ ] Run all tests with race detector:
  ```bash
  cd server && go test -race ./...
  ```
  Expected: all packages pass, no races reported.

- [ ] Run linter:
  ```bash
  cd server && golangci-lint run
  ```
  Expected: no errors. `depguard` should flag any remaining `"log"` imports.

- [ ] Fix any lint errors and commit: `chore(lint): fix golangci-lint findings after application layer refactor`

---

## Commit Summary

| Commit | Scope |
|---|---|
| `feat(types): add typed enums` | B1.1 |
| `feat(respond): add respond package` | B1.2 |
| `feat(handler): add adapter pattern with sentinel errors` | B2.1 |
| `feat(service): add WallpaperService with finalize flow` | B3.1 |
| `feat(service): add FavoriteService and ReportService` | B3.2 |
| `feat(service): migrate AuthService to internal/service` | B3.3 |
| `feat(middleware): add BannedCache` | B4.1 |
| `feat(middleware): add Auth and Admin middleware` | B4.2 |
| `feat(middleware): add configurable RateLimiter` | B4.3 |
| `feat(middleware): add SecurityHeaders and MaxBodySize` | B4.4 |
| `feat(server): rewrite main.go with chi v5` | B5.1 |
| `refactor(handler): rewrite wallpaper handler` | B6.1 |
| `refactor(handler): rewrite auth handler` | B6.2 |
| `refactor(handler): rewrite admin handler` | B6.3 |
| `refactor(handler): rewrite favorite and report handlers` | B6.4 |
| `feat(config): add Plan 2 fields` | B7.1 |
| `chore(lint): fix lint findings` | B8.1 |

---

## Key Invariants

- **Status transitions** are enforced exclusively in the service layer. Handlers never set a status string directly.
- **No magic strings.** All status, role, category, and sort values use `types.*` constants.
- **No `http.Error()` calls anywhere.** Every error response goes through `respond.Error()`.
- **No `log.` imports.** `depguard` enforces `slog` everywhere.
- **Banned check is per-request** with a 60s TTL cache. Ban takes effect on the very next API call.
- **Rate limits are config-driven.** No hardcoded integers in middleware constructors.
- **Debug server binds to `127.0.0.1:6060`** only. Never `0.0.0.0`. Requires `DEBUG_ENABLED=true`.
- **HTTP timeouts:** read=15s, write=30s, idle=120s. Shutdown timeout from config (default 25s).
- **`Popular`/`Recent`** do not mutate `r.URL`. They call the internal `list()` helper with a forced sort parameter.
