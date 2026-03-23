# Round 2: Full Fix Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix all 87 issues from the full audit. Every button works, every view is wired, every inconsistency is resolved, every server bug is patched.

**Audit:** `docs/superpowers/specs/2026-03-23-full-audit.md`
**Test strategy:** `docs/superpowers/specs/2026-03-23-test-strategy.md`
**Spec:** `docs/superpowers/specs/2026-03-23-screenspace-design.md`

**Verification approach:** After each task, run `swift build` (app) or `go test ./...` (server). For UI changes, use `osascript` accessibility queries and `screencapture` + Read tool. See test strategy doc for details.

---

## Phase 0: Server Fixes (S1-S12)

Fix server bugs first. The app depends on a correct backend. These are independent of Swift changes and can be done in parallel with Phase 1.

### Task 0.1: Fix JWT Panic (S1)

**Files:**
- Modify: `server/service/auth.go`
- Modify: `server/service/auth_test.go`

- [ ] **Step 1:** In `ValidateToken()`, replace bare type assertions `claims["sub"].(string)` with checked assertions using `, ok` pattern. Return error if claims are missing or wrong type.
- [ ] **Step 2:** Add test: malformed token with missing `sub` claim does not panic, returns error.
- [ ] **Step 3:** Add test: token with wrong type for `role` (e.g., int instead of string) returns error.
- [ ] **Step 4:** `go test ./service/...` passes. Commit.

### Task 0.2: Fix Download Count Inflation (S2)

**Files:**
- Modify: `server/handler/wallpaper.go`
- Modify: `server/handler/wallpaper_test.go`

- [ ] **Step 1:** Remove `IncrementDownloadCount()` from `GetWallpaper` handler (metadata view).
- [ ] **Step 2:** Add a new endpoint `POST /api/v1/wallpapers/:id/download` that increments count and returns the pre-signed download URL. This is what the app calls when the user actually clicks "Set as Wallpaper" or "Download".
- [ ] **Step 3:** Update route registration in `main.go`.
- [ ] **Step 4:** Add test: `GET /wallpapers/:id` does NOT increment count. `POST /wallpapers/:id/download` DOES increment and returns download URL.
- [ ] **Step 5:** Commit.

### Task 0.3: Add Input Validation (S7, S8, S9)

**Files:**
- Modify: `server/handler/wallpaper.go`
- Modify: `server/handler/report.go`
- Modify: `server/handler/wallpaper_test.go`
- Modify: `server/handler/report_test.go`

- [ ] **Step 1:** In upload handler, validate: title max 255 chars, tags max 10 items, each tag max 50 chars. Return 400 with descriptive message on violation.
- [ ] **Step 2:** In report handler, validate: reason max 500 chars. Return 400 on violation.
- [ ] **Step 3:** Add tests for each validation: too-long title returns 400, too many tags returns 400, too-long tag returns 400, too-long reason returns 400.
- [ ] **Step 4:** Commit.

### Task 0.4: Fix Admin Action Validation (S3, S4)

**Files:**
- Modify: `server/handler/admin.go`
- Modify: `server/handler/admin_test.go`
- Create: `server/migrations/002_add_rejection_reason.up.sql`
- Create: `server/migrations/002_add_rejection_reason.down.sql`
- Modify: `server/repository/wallpaper.go`

- [ ] **Step 1:** In ban/unban/promote handlers, check if user exists first with `GetByID()`. Return 404 if not found.
- [ ] **Step 2:** Add migration: `ALTER TABLE wallpapers ADD COLUMN rejection_reason TEXT;`
- [ ] **Step 3:** Update `UpdateStatus()` in repository to accept optional reason. Store reason when rejecting.
- [ ] **Step 4:** In `GetWallpaper` response, include `rejection_reason` for rejected wallpapers (visible to uploader).
- [ ] **Step 5:** Add tests: ban non-existent user returns 404. Reject with reason stores and returns the reason.
- [ ] **Step 6:** Commit.

### Task 0.5: Fix Category System (S5, S6)

**Files:**
- Modify: `server/repository/wallpaper.go`
- Modify: `server/handler/wallpaper.go`
- Create: `server/handler/category.go`
- Modify: `server/main.go`
- Modify: `server/repository/wallpaper_test.go`

- [ ] **Step 1:** Define valid categories as a constant slice: `["nature", "abstract", "urban", "cinematic", "space", "underwater", "minimal", "other"]`.
- [ ] **Step 2:** In upload handler, validate category against the list (case-insensitive). Normalize to lowercase before storing. Return 400 for invalid category.
- [ ] **Step 3:** In list query, change category filter from `= $1` to `ILIKE $1` for case-insensitive matching.
- [ ] **Step 4:** Add `GET /api/v1/categories` endpoint that returns the valid category list. Lightweight, no auth required.
- [ ] **Step 5:** Add tests: upload with invalid category returns 400, filter "Nature" matches "nature", categories endpoint returns list.
- [ ] **Step 6:** Commit.

### Task 0.6: Fix Rate Limiter Memory Leak (S10)

**Files:**
- Modify: `server/middleware/ratelimit.go`
- Modify: `server/middleware/ratelimit_test.go`

- [ ] **Step 1:** Add a background goroutine (or check-on-access) that removes entries older than 24 hours from the `limits` map.
- [ ] **Step 2:** Add test: create entries, advance time, verify stale entries are cleaned up.
- [ ] **Step 3:** Commit.

### Task 0.7: Extend Pre-signed URL Expiry + Add Logging (S11, S12)

**Files:**
- Modify: `server/handler/wallpaper.go`
- Modify: `server/main.go`

- [ ] **Step 1:** Change pre-signed upload URL expiry from 15 minutes to 2 hours.
- [ ] **Step 2:** Add structured logging (Go `slog` package) for all mutation endpoints: upload, approve, reject, ban, unban, promote, report, dismiss. Log user ID, action, target, timestamp.
- [ ] **Step 3:** Commit.

---

## Phase 1: Foundation (C2, W-plan Task 1)

Everything depends on shared app state. This blocks all subsequent app work.

### Task 1.1: Create AppState and Inject Into App

**Covers:** C2, C3, C1 (UploadView own APIClient), wiring plan Task 1

**Files:**
- Create: `app/Sources/ScreenSpace/AppState.swift`
- Modify: `app/Sources/ScreenSpace/App.swift`

- [ ] **Step 1: Create AppState**

```swift
import SwiftUI
import Combine

@Observable
@MainActor
final class AppState {
    let engine: WallpaperEngine
    let api: APIClient
    let configManager: ConfigManager
    let playlistManager: PlaylistManager
    let lockScreen: LockScreenManager
    let pauseController: PauseController

    var currentUser: UserResponse?
    var isLoggedIn: Bool { currentUser != nil }
    var isAdmin: Bool { currentUser?.role == "admin" }
    var currentWallpaperURL: URL?
    var currentWallpaperTitle: String?

    init() {
        self.configManager = .shared
        self.playlistManager = .shared
        self.api = APIClient()
        self.lockScreen = LockScreenManager()
        self.engine = WallpaperEngine(configManager: configManager)
        self.pauseController = PauseController(config: configManager.config)
    }
}
```

Add methods:
- `setWallpaper(url:title:)` -- calls engine, updates currentWallpaperURL/Title, saves to config
- `setWallpaper(url:title:forDisplay:)` -- per-display variant
- `downloadAndSetWallpaper(wallpaper: WallpaperResponse)` -- downloads from pre-signed URL via new `/download` endpoint, shows progress, caches, then sets. Returns download progress via `@Published` or callback.
- `login(email:password:) async throws`
- `register(email:password:) async throws`
- `logout()`
- `restoreSession() async` -- checks Keychain, calls `/me`
- `restoreLastWallpaper()` -- reads `lastPlayedURL` from config, calls `setWallpaper` if file exists (fixes M3)
- `skipToNext()` -- advances playlist, sets next wallpaper

- [ ] **Step 2: Wire PauseController to Engine (fixes M1)**

In AppState init, observe `pauseController.shouldPause`:
```swift
// In init, after creating pauseController:
withObservationTracking {
    _ = self.pauseController.shouldPause
} onChange: {
    Task { @MainActor in
        if self.pauseController.shouldPause {
            self.engine.pauseAll()
        } else {
            self.engine.resumeAll()
        }
    }
}
```

Actually, since PauseController is `ObservableObject` with `@Published`, use Combine:
```swift
pauseController.$shouldPause
    .removeDuplicates()
    .sink { [weak self] shouldPause in
        if shouldPause {
            self?.engine.pauseAll()
        } else {
            self?.engine.resumeAll()
        }
    }
    .store(in: &cancellables)
```

- [ ] **Step 3: Add fullscreen occlusion to PauseController (fixes M2)**

In PauseController, add a periodic check (every 5s) for each WallpaperWindow's `occlusionState`:
```swift
if config.pauseOnFullscreen {
    // Check if any wallpaper window is fully occluded
    // This needs WallpaperEngine reference or a callback
}
```

Alternative: PauseController takes a closure `() -> Bool` that checks occlusion. AppState provides it from engine.

- [ ] **Step 4: Modify App.swift**

Replace bare `WallpaperEngine` in AppDelegate with `AppState`. Inject into SwiftUI:
```swift
let appState = AppState()

// In applicationDidFinishLaunching:
appState.engine.start()
Task { await appState.restoreSession() }
appState.restoreLastWallpaper()  // fixes M3
```

Pass to GalleryWindowController so it can inject into SwiftUI via `.environment()`.

- [ ] **Step 5: Update "now playing" menu item (fixes M4)**

In `buildMenu()`, store reference to the `nowPlaying` menu item. When `appState.currentWallpaperTitle` changes, update the title. Use a Combine subscription or KVO.

- [ ] **Step 6: Write AppState tests**

Create `app/Tests/ScreenSpaceTests/AppStateTests.swift`:
- Test: `setWallpaper` updates `currentWallpaperURL` and `currentWallpaperTitle`
- Test: `login` sets `currentUser`, `isLoggedIn` is true
- Test: `logout` clears `currentUser`, `isLoggedIn` is false
- Test: `isAdmin` returns true when user role is "admin"
- Test: `restoreLastWallpaper` with valid cached URL calls engine
- Test: `restoreLastWallpaper` with missing file does not crash

- [ ] **Step 7:** `swift build && swift test`. Commit.

---

## Phase 2: Wire All Buttons (W1-W13)

Now that AppState exists, wire everything. Each view gets `@Environment(AppState.self) var appState`.

### Task 2.1: Wire "Set as Wallpaper" Buttons (W1, W2, W12)

**Files:**
- Modify: `app/Sources/ScreenSpace/UI/Views/LibraryView.swift`
- Modify: `app/Sources/ScreenSpace/UI/Views/DetailView.swift`

- [ ] **Step 1: LibraryView**

Add `@Environment(AppState.self) var appState`. In `setWallpaper(url:)`:
```swift
appState.setWallpaper(url: url, title: url.lastPathComponent)
```

- [ ] **Step 2: DetailView**

Add `@Environment(AppState.self) var appState`. In `setAsWallpaper()`:
- Check `CacheManager.shared.cachedURL(for: wallpaper.id)` first
- If cached, call `appState.setWallpaper(url: cachedURL, title: wallpaper.title)`
- If not cached, call `appState.downloadAndSetWallpaper(wallpaper: wallpaper)` which:
  - Sets `isDownloading = true`
  - Calls `POST /wallpapers/:id/download` to get pre-signed URL and increment count
  - Downloads with URLSession, updates `downloadProgress` during download
  - Caches via CacheManager
  - Sets wallpaper
  - Sets `isDownloading = false`

- [ ] **Step 3: Wire favorite button (W8)**

```swift
Button(action: {
    guard appState.isLoggedIn else { showLoginPrompt = true; return }
    Task {
        isFavorited = try await appState.api.toggleFavorite(id: wallpaper.id)
    }
}) {
    Image(systemName: isFavorited ? "heart.fill" : "heart")
}
```

Add `@State private var isFavorited = false` and load initial state in `.task`.

- [ ] **Step 4: Wire report button (W9)**

```swift
Button(action: {
    guard appState.isLoggedIn else { showLoginPrompt = true; return }
    showReportSheet = true
}) {
    Image(systemName: "flag")
}
.sheet(isPresented: $showReportSheet) {
    ReportSheet(wallpaperID: wallpaper.id)
}
```

Create a small `ReportSheet` view with a text field for reason and submit button.

- [ ] **Step 5:** `swift build`. Commit.

### Task 2.2: Wire Card Navigation (W3)

**Files:**
- Modify: `app/Sources/ScreenSpace/UI/Components/WallpaperCard.swift`
- Modify: `app/Sources/ScreenSpace/UI/Components/ShelfRow.swift`
- Modify: `app/Sources/ScreenSpace/UI/Views/HomeView.swift`

- [ ] **Step 1:** Add `var onTap: (() -> Void)?` to WallpaperCard. Wrap body in `Button(action: { onTap?() }) { ... }.buttonStyle(.plain)`.
- [ ] **Step 2:** Add `var onSelectWallpaper: ((WallpaperCardData) -> Void)?` to ShelfRow. Pass to each card's `onTap`.
- [ ] **Step 3:** In HomeView, add `@State var selectedWallpaper: WallpaperResponse?`. On card tap, fetch full wallpaper via `appState.api.getWallpaper(id:)`, set `selectedWallpaper`. Show `.sheet` with `DetailView(wallpaper:)`.
- [ ] **Step 4:** `swift build`. Commit.

### Task 2.3: Wire HomeView to API (W4)

**Files:**
- Modify: `app/Sources/ScreenSpace/UI/Views/HomeView.swift`

- [ ] **Step 1:** Add `@Environment(AppState.self) var appState` and state vars:
```swift
@State private var popular: [WallpaperCardData] = []
@State private var recent: [WallpaperCardData] = []
@State private var featured: WallpaperCardData?
@State private var isLoading = true
@State private var loadError: String?
```

- [ ] **Step 2:** In `.task`, call API:
```swift
do {
    let pop = try await appState.api.popularWallpapers(limit: 10)
    let rec = try await appState.api.recentWallpapers(limit: 10)
    popular = pop.wallpapers.map { $0.toCardData() }
    recent = rec.wallpapers.map { $0.toCardData() }
    featured = popular.first
} catch {
    loadError = "Connect to a server in Settings to browse community wallpapers."
}
isLoading = false
```

- [ ] **Step 3:** Add `toCardData()` extension on `WallpaperResponse` (in APIModels or a new extension file).
- [ ] **Step 4:** Show `ProgressView` while loading. Show error message if API fails. Keep `placeholderData` as fallback only when no server configured (offline mode).
- [ ] **Step 5:** `swift build`. Commit.

### Task 2.4: Wire HeroSection + ShelfRow Buttons (W6, W7, W10)

**Files:**
- Modify: `app/Sources/ScreenSpace/UI/Components/HeroSection.swift`
- Modify: `app/Sources/ScreenSpace/UI/Components/ShelfRow.swift`

- [ ] **Step 1:** HeroSection: add `onViewWallpaper: (() -> Void)?` and `onFavorite: (() -> Void)?` callbacks. Wire to buttons.
- [ ] **Step 2:** ShelfRow: add `onSeeAll: (() -> Void)?` callback. Wire to "See All" button.
- [ ] **Step 3:** HomeView: pass callbacks. `onViewWallpaper` fetches wallpaper and shows detail sheet. `onSeeAll` navigates to a filtered list (or ExploreView with pre-set category).
- [ ] **Step 4:** `swift build`. Commit.

### Task 2.5: Wire ExploreView (W11)

**Files:**
- Modify: `app/Sources/ScreenSpace/UI/Views/ExploreView.swift`

- [ ] **Step 1:** Add `@Environment(AppState.self) var appState` and state:
```swift
@State private var categories: [String] = []
@State private var selectedCategory: String?
@State private var searchQuery = ""
@State private var results: [WallpaperCardData] = []
@State private var isLoading = false
```

- [ ] **Step 2:** In `.task`, fetch categories from `GET /api/v1/categories`.
- [ ] **Step 3:** Display categories as a grid of styled cards. On tap, fetch wallpapers filtered by category.
- [ ] **Step 4:** Add search field at top. On submit, call `appState.api.listWallpapers(query:)`.
- [ ] **Step 5:** Show results in a `LazyVGrid` of `WallpaperCard` with `onTap` navigation to detail.
- [ ] **Step 6:** `swift build`. Commit.

### Task 2.6: Hide Admin Tab (W5)

**Files:**
- Modify: `app/Sources/ScreenSpace/UI/GalleryWindow.swift`

- [ ] **Step 1:** Add `@Environment(AppState.self) var appState` to GalleryContentView.
- [ ] **Step 2:** Wrap "Manage" section in `if appState.isAdmin { ... }`.
- [ ] **Step 3:** Verify via osascript: when not logged in, "Admin" text should not exist in window element tree.
- [ ] **Step 4:** `swift build`. Commit.

### Task 2.7: Wire "Next Wallpaper" Menu (W13)

**Files:**
- Modify: `app/Sources/ScreenSpace/App.swift`

- [ ] **Step 1:** In `skipToNext()`, call `appState.skipToNext()`.
- [ ] **Step 2:** `AppState.skipToNext()` implementation:
  - Get active playlist from PlaylistManager
  - If no playlist or empty, do nothing
  - Advance index (or pick random if shuffle)
  - Resolve item: local path or cached community URL
  - Call `setWallpaper(url:title:)`
- [ ] **Step 3:** Add keyboard shortcut to "Next Wallpaper": `Ctrl+Option+N`.
- [ ] **Step 4:** `swift build`. Commit.

---

## Phase 3: Login Flow + Auth Gates (V5, V10, V16, wiring plan Task 9)

### Task 3.1: Create LoginView

**Files:**
- Create: `app/Sources/ScreenSpace/UI/Views/LoginView.swift`
- Modify: `app/Sources/ScreenSpace/UI/Views/SettingsView.swift`

- [ ] **Step 1:** Create LoginView:
```swift
struct LoginView: View {
    @Environment(AppState.self) var appState
    @Environment(\.dismiss) var dismiss
    @State private var email = ""
    @State private var password = ""
    @State private var isRegistering = false
    @State private var isLoading = false
    @State private var errorMessage: String?

    var body: some View {
        VStack(spacing: 16) {
            Text(isRegistering ? "Create Account" : "Log In")
                .font(.title2).fontWeight(.bold)
            TextField("Email", text: $email)
                .textFieldStyle(.roundedBorder)
            SecureField("Password", text: $password)
                .textFieldStyle(.roundedBorder)
            if let error = errorMessage {
                Text(error).foregroundStyle(.red).font(.caption)
            }
            HStack {
                Button("Cancel") { dismiss() }.buttonStyle(.bordered)
                Button(isRegistering ? "Create Account" : "Log In") {
                    Task { await submit() }
                }
                .buttonStyle(.borderedProminent)
                .disabled(email.isEmpty || password.isEmpty || isLoading)
            }
            Button(isRegistering ? "Already have an account? Log in" : "Create an account") {
                isRegistering.toggle()
                errorMessage = nil
            }
            .buttonStyle(.plain).font(.caption)
        }
        .padding().frame(width: 350)
    }

    private func submit() async {
        isLoading = true; errorMessage = nil
        do {
            if isRegistering {
                try await appState.register(email: email, password: password)
            } else {
                try await appState.login(email: email, password: password)
            }
            dismiss()
        } catch { errorMessage = error.localizedDescription }
        isLoading = false
    }
}
```

- [ ] **Step 2:** Update SettingsView account tab:
```swift
private var accountTab: some View {
    Form {
        if let user = appState.currentUser {
            LabeledContent("Email", value: user.email)
            LabeledContent("Role", value: user.role.capitalized)
            Button("Log Out") { appState.logout() }
                .buttonStyle(.bordered)
        } else {
            Text("Log in to upload and favorite wallpapers.")
                .foregroundStyle(.secondary)
            Button("Log In") { showLogin = true }
                .buttonStyle(.borderedProminent)
        }
    }
}
```

Add `@State private var showLogin = false` and `.sheet(isPresented: $showLogin) { LoginView() }`.

- [ ] **Step 3:** `swift build`. Commit.

### Task 3.2: Add Auth Gates (V10, V16)

**Files:**
- Modify: `app/Sources/ScreenSpace/UI/GalleryWindow.swift`
- Modify: `app/Sources/ScreenSpace/UI/Views/UploadView.swift`

- [ ] **Step 1:** Upload toolbar button: only show when `appState.isLoggedIn`. Or show always but when tapped while not logged in, show LoginView instead of UploadView.
- [ ] **Step 2:** UploadView: remove `private let api = APIClient()`. Use `@Environment(AppState.self) var appState` and `appState.api` instead. (Fixes C1.)
- [ ] **Step 3:** `swift build`. Commit.

---

## Phase 4: Missing Feature UIs (M5-M9, M7, M11, M14)

### Task 4.1: Favorites View (M7)

**Files:**
- Create: `app/Sources/ScreenSpace/UI/Views/FavoritesView.swift`
- Modify: `app/Sources/ScreenSpace/UI/GalleryWindow.swift`

- [ ] **Step 1:** Create FavoritesView that fetches `appState.api.listFavorites()` and displays a grid of `WallpaperCard`.
- [ ] **Step 2:** Add "Favorites" to sidebar under "Your Stuff" section (only visible when logged in).
- [ ] **Step 3:** Add `GallerySection.favorites` case to enum.
- [ ] **Step 4:** `swift build`. Commit.

### Task 4.2: Playlist UI (M5)

**Files:**
- Create: `app/Sources/ScreenSpace/UI/Views/PlaylistsView.swift`
- Modify: `app/Sources/ScreenSpace/UI/GalleryWindow.swift`

- [ ] **Step 1:** Create PlaylistsView:
  - List of user playlists with create/delete
  - Each playlist shows items, drag-to-reorder
  - Settings per playlist: name, interval (seconds between changes, 0 = manual), shuffle toggle
  - "Add from Library" and "Add from Community" buttons to add items

- [ ] **Step 2:** Add "Playlists" to sidebar under "Your Stuff".
- [ ] **Step 3:** Add `GallerySection.playlists` case.
- [ ] **Step 4:** `swift build`. Commit.

### Task 4.3: Per-Display Assignment (M6)

**Files:**
- Modify: `app/Sources/ScreenSpace/UI/Views/SettingsView.swift`
- Modify: `app/Sources/ScreenSpace/Config/AppConfig.swift`

- [ ] **Step 1:** In Displays tab, for each display show:
  - Display name (friendly, e.g., "Built-in Retina Display")
  - Current wallpaper (if any)
  - Picker to assign a playlist (from PlaylistManager)
  - "Choose Wallpaper" button to pick from library

- [ ] **Step 2:** Wire `screenAssignments` config field (currently unused, C5) to store `displayID -> playlistID` mapping.
- [ ] **Step 3:** On assignment change, update engine for that specific display.
- [ ] **Step 4:** `swift build`. Commit.

### Task 4.4: Lock Screen Button (M8)

**Files:**
- Modify: `app/Sources/ScreenSpace/UI/Views/DetailView.swift`
- Modify: `app/Sources/ScreenSpace/UI/Views/LibraryView.swift`

- [ ] **Step 1:** Add "Set as Lock Screen" button to DetailView (next to "Set as Wallpaper"):
```swift
Button(action: {
    Task {
        try await appState.lockScreen.setLockScreen(from: videoURL)
    }
}) {
    Label("Set as Lock Screen", systemImage: "lock.rectangle")
}
.buttonStyle(.bordered)
```

- [ ] **Step 2:** Add same button to LibraryView local video cards (context menu or secondary button).
- [ ] **Step 3:** Show alert explaining this requires admin permissions and sets a static frame (macOS limitation).
- [ ] **Step 4:** `swift build`. Commit.

### Task 4.5: Video Preview in DetailView (M11)

**Files:**
- Modify: `app/Sources/ScreenSpace/UI/Views/DetailView.swift`

- [ ] **Step 1:** Replace static play icon with an `AVPlayerView` (via `NSViewRepresentable`):
```swift
struct VideoPreview: NSViewRepresentable {
    let url: URL
    func makeNSView(context: Context) -> AVPlayerView { ... }
    func updateNSView(_ nsView: AVPlayerView, context: Context) { ... }
}
```

- [ ] **Step 2:** For community wallpapers, use the preview URL (low-res 10s clip) from the API response. For local videos, play the actual file.
- [ ] **Step 3:** Auto-play on appear, loop, muted by default.
- [ ] **Step 4:** `swift build`. Commit.

### Task 4.6: Library Thumbnails (M14)

**Files:**
- Modify: `app/Sources/ScreenSpace/UI/Views/LibraryView.swift`

- [ ] **Step 1:** In `localVideoCard(url:)`, replace gray rectangle with actual thumbnail:
```swift
if let thumbnail = ThumbnailGenerator.thumbnail(for: url) {
    Image(nsImage: thumbnail)
        .resizable()
        .scaledToFill()
        .clipShape(RoundedRectangle(cornerRadius: 12))
} else {
    // Keep placeholder as fallback
}
```

- [ ] **Step 2:** Generate thumbnails on import (in `handleDrop`) and on `loadLibrary()`.
- [ ] **Step 3:** `swift build`. Commit.

### Task 4.7: Cache Eviction (M13)

**Files:**
- Modify: `app/Sources/ScreenSpace/Helpers/CacheManager.swift`

- [ ] **Step 1:** Add `evictIfNeeded()` method:
  - Calculate current cache size
  - If over `configManager.config.cacheSizeLimitMB`, delete oldest files until under limit
  - Sort cached files by modification date, delete oldest first

- [ ] **Step 2:** Call `evictIfNeeded()` after every `cacheFile()` call.
- [ ] **Step 3:** Add unit test: cache over limit evicts oldest file.
- [ ] **Step 4:** `swift build && swift test`. Commit.

---

## Phase 5: Visual/UX Fixes (V1-V16, L1-L10)

### Task 5.1: Fix Settings Panel Layout (V1, V2)

**Files:**
- Modify: `app/Sources/ScreenSpace/UI/Views/SettingsView.swift`

- [ ] **Step 1:** Change all tab `Form` views to use `VStack(alignment: .leading)` with `.frame(maxHeight: .infinity, alignment: .top)` so content is top-aligned.
- [ ] **Step 2:** Remove the double border by adjusting the sheet presentation or GroupBox styling. Check if `.formStyle(.grouped)` or explicit background removes the artifact.
- [ ] **Step 3:** Take screenshot with `screencapture`, verify content is top-aligned and no double border.
- [ ] **Step 4:** Commit.

### Task 5.2: Fix Display Tab (V3)

**Files:**
- Modify: `app/Sources/ScreenSpace/UI/Views/SettingsView.swift`

- [ ] **Step 1:** Hide the raw stable display ID. Only show the friendly display name. If debug info is needed, put it behind an Option-click or info button.
- [ ] **Step 2:** Commit.

### Task 5.3: Fix Storage Tab (V4)

**Files:**
- Modify: `app/Sources/ScreenSpace/UI/Views/SettingsView.swift`

- [ ] **Step 1:** Format cache limit in human-readable form. If >= 1024 MB, show as GB:
```swift
Text(config.cacheSizeLimitMB >= 1024
    ? String(format: "%.1f GB", Double(config.cacheSizeLimitMB) / 1024)
    : "\(config.cacheSizeLimitMB) MB")
```
- [ ] **Step 2:** Same for current cache size display.
- [ ] **Step 3:** Commit.

### Task 5.4: Fix Upload Sheet (V9, V11, V12, V13)

**Files:**
- Modify: `app/Sources/ScreenSpace/UI/Views/UploadView.swift`

- [ ] **Step 1:** Make "content policy" a clickable link. Open `CONTENT_POLICY.md` URL (GitHub raw URL or local resource) in default browser.
```swift
Toggle(isOn: $acceptedPolicy) {
    HStack(spacing: 4) {
        Text("I confirm this content complies with the")
        Link("content policy", destination: URL(string: "https://github.com/0x63616c/screenspace/blob/main/CONTENT_POLICY.md")!)
        Text("and I have the rights to upload it")
    }
    .font(.caption)
}
```

- [ ] **Step 2:** Replace category text field with a Picker using categories from `GET /api/v1/categories`:
```swift
Picker("Category", selection: $category) {
    Text("Select category").tag("")
    ForEach(categories, id: \.self) { cat in
        Text(cat.capitalized).tag(cat)
    }
}
```

- [ ] **Step 3:** After file selection, validate and show info:
  - File size (reject > 200MB with error)
  - Duration (reject > 60s with error)
  - Resolution (warn if < 1080p)

- [ ] **Step 4:** Fix double-border styling (same fix as settings panel).
- [ ] **Step 5:** Take screenshot, verify link is blue/underlined, category is picker, file info shows.
- [ ] **Step 6:** Commit.

### Task 5.5: Fix Language Inconsistencies (L1-L5)

**Files:**
- Modify: `app/Sources/ScreenSpace/UI/Components/HeroSection.swift`
- Modify: `app/Sources/ScreenSpace/UI/Views/SettingsView.swift`
- Modify: `app/Sources/ScreenSpace/UI/Views/LibraryView.swift`

- [ ] **Step 1:** Standardize action verb: "Set as Wallpaper" everywhere (not "View Wallpaper"). HeroSection button changes from "View Wallpaper" to "Set as Wallpaper". (L1)
- [ ] **Step 2:** Standardize format references: "MP4 or MOV" everywhere file types are mentioned. (L2)
- [ ] **Step 3:** Fix account tab text (L3, L4): replace "Not logged in" + misleading hint with a proper login button (already done in Task 3.1).
- [ ] **Step 4:** Commit.

### Task 5.6: Fix Typography Inconsistencies (L6, L7, L8)

**Files:**
- Multiple UI files

- [ ] **Step 1:** Define a spacing scale and apply consistently. Define as constants:
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

- [ ] **Step 2:** Replace all magic spacing numbers with `Spacing.*` constants.
- [ ] **Step 3:** Standardize caption usage: `.caption` for metadata labels, `.caption2` only for timestamps/secondary metadata.
- [ ] **Step 4:** Commit.

### Task 5.7: Fix Button Style Inconsistencies (L9, L10)

**Files:**
- Modify: `app/Sources/ScreenSpace/UI/Views/LibraryView.swift`

- [ ] **Step 1:** "Set as Wallpaper" in LibraryView: change from `.bordered` to `.borderedProminent` to match DetailView. (L9)
- [ ] **Step 2:** Verify all primary action buttons use `.borderedProminent`, secondary use `.bordered`, navigation/links use `.plain`.
- [ ] **Step 3:** Commit.

### Task 5.8: Fix General Tab + Branding (V7, V8)

**Files:**
- Modify: `app/Sources/ScreenSpace/UI/Views/SettingsView.swift`
- Modify: `app/Sources/ScreenSpace/UI/GalleryWindow.swift`

- [ ] **Step 1:** Read version from bundle:
```swift
Text("Version \(Bundle.main.infoDictionary?["CFBundleShortVersionString"] as? String ?? "dev")")
```
Note: SPM executables don't have Info.plist by default. May need to add one or hardcode for now with a `// TODO: read from bundle when using Xcode`.

- [ ] **Step 2:** Add "ScreenSpace" text in the toolbar or sidebar header area for branding.
- [ ] **Step 3:** Commit.

### Task 5.9: Library Delete + Hero Clipping (V14, V15)

**Files:**
- Modify: `app/Sources/ScreenSpace/UI/Views/LibraryView.swift`
- Modify: `app/Sources/ScreenSpace/UI/Components/HeroSection.swift`

- [ ] **Step 1:** Add context menu to library video cards:
```swift
.contextMenu {
    Button("Set as Wallpaper") { setWallpaper(url: url) }
    Button("Set as Lock Screen") { ... }
    Divider()
    Button("Remove from Library", role: .destructive) {
        try? FileManager.default.removeItem(at: url)
        localVideos.removeAll { $0 == url }
    }
}
```

- [ ] **Step 2:** Hero section: add `.lineLimit(2)` and `.truncationMode(.tail)` to title. Ensure the parent VStack has proper width constraints.
- [ ] **Step 3:** Commit.

---

## Phase 6: Code Quality (C4-C21)

### Task 6.1: Remove Dead Config Fields (C4, C5)

**Files:**
- Modify: `app/Sources/ScreenSpace/Config/AppConfig.swift`

- [ ] **Step 1:** `videoQuality` -- remove if unused. If we want it later, add it later. YAGNI.
- [ ] **Step 2:** `screenAssignments` was wired in Task 4.3. If Task 4.3 is deferred, leave it but add a comment.
- [ ] **Step 3:** `swift build`. Commit.

### Task 6.2: Fix Error Handling (C6, C7, C8, C9)

**Files:**
- Modify: `app/Sources/ScreenSpace/UI/Views/SettingsView.swift`
- Modify: `app/Sources/ScreenSpace/UI/Views/LibraryView.swift`
- Modify: `app/Sources/ScreenSpace/UI/Views/UploadView.swift`

- [ ] **Step 1:** SettingsView launch-at-login: show alert on failure:
```swift
do {
    try SMAppService.mainApp.register()
} catch {
    launchAtLoginError = error.localizedDescription
}
```

- [ ] **Step 2:** LibraryView drag-and-drop: surface import errors to user via alert.
- [ ] **Step 3:** UploadView: replace `URL(string:)!` force unwrap with guard-let and proper error message.
- [ ] **Step 4:** `swift build`. Commit.

### Task 6.3: Add Accessibility (C17, C20)

**Files:**
- Multiple UI files

- [ ] **Step 1:** WallpaperCard: add `.accessibilityLabel("\(data.title), \(ResolutionBadge.label(width:height:)), \(data.durationLabel)")` and `.accessibilityAddTraits(.isButton)`.
- [ ] **Step 2:** ResolutionBadge: add `.accessibilityLabel("Resolution: \(label)")`.
- [ ] **Step 3:** HeroSection: add accessibility labels to buttons.
- [ ] **Step 4:** DetailView: add accessibility labels to metadata row items.
- [ ] **Step 5:** LibraryView drop zone: add `.accessibilityLabel("Drop zone for video files")`.
- [ ] **Step 6:** `swift build`. Commit.

### Task 6.4: Add Keyboard Shortcuts (C18)

**Files:**
- Modify: `app/Sources/ScreenSpace/App.swift`
- Modify: `app/Sources/ScreenSpace/UI/GalleryWindow.swift`

- [ ] **Step 1:** Add keyboard equivalent to "Next Wallpaper" menu: `Ctrl+Option+N` (already in Task 2.7).
- [ ] **Step 2:** Settings: use Cmd+, (standard macOS). Add `.keyboardShortcut(",", modifiers: .command)` to Settings toolbar button.
- [ ] **Step 3:** Upload: `Cmd+U` on the Upload toolbar button.
- [ ] **Step 4:** `swift build`. Commit.

### Task 6.5: Add Context Menus (C19)

**Files:**
- Modify: `app/Sources/ScreenSpace/UI/Components/WallpaperCard.swift`

- [ ] **Step 1:** Add context menu to WallpaperCard:
```swift
.contextMenu {
    Button("Set as Wallpaper") { onTap?() }
    Button("Add to Favorites") { ... }
    Button("Report") { ... }
}
```

This requires passing callbacks or using environment. Keep it simple: the context menu mirrors what's available in DetailView.

- [ ] **Step 2:** `swift build`. Commit.

---

## Phase 7: Final Polish + Smoke Test

### Task 7.1: Screensaver Install (M9)

**Files:**
- Create: `app/Sources/ScreenSpace/Helpers/ScreensaverInstaller.swift`
- Modify: `app/Sources/ScreenSpace/UI/Views/SettingsView.swift`

- [ ] **Step 1:** Create a helper that copies the `.saver` bundle to `~/Library/Screen Savers/`:
```swift
enum ScreensaverInstaller {
    static func install() throws {
        let source = Bundle.main.url(forResource: "ScreenSpaceSaver", withExtension: "saver")
        let dest = FileManager.default.homeDirectoryForCurrentUser
            .appendingPathComponent("Library/Screen Savers/ScreenSpaceSaver.saver")
        try FileManager.default.copyItem(at: source!, to: dest)
    }
    static var isInstalled: Bool { ... }
}
```

Note: this only works when bundled as a proper .app. For SPM dev builds, skip with a graceful message.

- [ ] **Step 2:** Add "Install Screensaver" button in General settings tab.
- [ ] **Step 3:** Commit.

### Task 7.2: Full Smoke Test

- [ ] **Step 1:** Start server: `cd server && docker compose up -d`
- [ ] **Step 2:** Build app: `cd app && swift build`
- [ ] **Step 3:** Launch app, take screenshot, verify:
  - Home loads with API data (or graceful offline message)
  - Sidebar has correct sections (no Admin when not logged in)
  - Settings panel content is top-aligned
  - Upload sheet has category picker and policy link
- [ ] **Step 4:** Via osascript, verify:
  - Click "Explore" -- categories grid loads
  - Click "Library" -- drop zone appears
  - Open settings, check each tab renders
- [ ] **Step 5:** Run all tests:
  ```bash
  cd app && swift test
  cd server && go test ./...
  ```
- [ ] **Step 6:** Tear down: `docker compose down`, kill app.
- [ ] **Step 7:** Commit any final fixes. Create summary of all changes.

---

## Dependency Graph

```
Phase 0 (server) ──────────────────────────────────────┐
                                                        │
Phase 1 (AppState) ─────┬──────────────────────────────┤
                        │                              │
Phase 2 (wire buttons) ─┤                              │
                        │                              │
Phase 3 (login + auth) ─┤                              │
                        │                              │
Phase 4 (new features) ─┤                              │
                        │                              │
Phase 5 (visual/UX) ────┤                              │
                        │                              │
Phase 6 (code quality) ─┘                              │
                                                        │
Phase 7 (smoke test) ──────────────────────────────────┘
```

- **Phase 0 and Phase 1 can run in parallel** (server vs app, independent codebases)
- **Phases 2-6 depend on Phase 1** (all need AppState)
- **Phases 2-6 are mostly independent of each other** (can be parallelized with agent teams)
- **Phase 7 depends on everything**

## Issue Coverage Map

Every audit issue ID mapped to a task:

| Issue | Task | | Issue | Task | | Issue | Task |
|-------|------|-|-------|------|-|-------|------|
| W1 | 2.1 | | M1 | 1.1 | | V1 | 5.1 |
| W2 | 2.1 | | M2 | 1.1 | | V2 | 5.1 |
| W3 | 2.2 | | M3 | 1.1 | | V3 | 5.2 |
| W4 | 2.3 | | M4 | 1.1 | | V4 | 5.3 |
| W5 | 2.6 | | M5 | 4.2 | | V5 | 3.1 |
| W6 | 2.4 | | M6 | 4.3 | | V6 | -- |
| W7 | 2.4 | | M7 | 4.1 | | V7 | 5.8 |
| W8 | 2.1 | | M8 | 4.4 | | V8 | 5.8 |
| W9 | 2.1 | | M9 | 7.1 | | V9 | 5.4 |
| W10 | 2.4 | | M10 | -- | | V10 | 3.2 |
| W11 | 2.5 | | M11 | 4.5 | | V11 | 5.4 |
| W12 | 2.1 | | M12 | -- | | V12 | 5.4 |
| W13 | 2.7 | | M13 | 4.7 | | V13 | 5.4 |
| W14 | -- | | M14 | 4.6 | | V14 | 5.9 |
| | | | | | | V15 | 5.9 |
| | | | | | | V16 | 3.2 |

| Issue | Task | | Issue | Task |
|-------|------|-|-------|------|
| L1 | 5.5 | | C1 | 3.2 |
| L2 | 5.5 | | C2 | 1.1 |
| L3 | 3.1 | | C3 | 1.1 |
| L4 | 3.1 | | C4 | 6.1 |
| L5 | 5.5 | | C5 | 4.3/6.1 |
| L6 | 5.6 | | C6 | 6.2 |
| L7 | 5.6 | | C7 | 6.2 |
| L8 | 5.6 | | C8 | 6.2 |
| L9 | 5.7 | | C9 | 6.2 |
| L10 | 5.7 | | C10-C13 | -- |
| | | | C14-C16 | -- |
| | | | C17 | 6.3 |
| | | | C18 | 6.4 |
| | | | C19 | 6.5 |
| | | | C20 | 6.3 |
| | | | C21 | -- |

| Issue | Task |
|-------|------|
| S1 | 0.1 |
| S2 | 0.2 |
| S3 | 0.4 |
| S4 | 0.4 |
| S5 | 0.5 |
| S6 | 0.5 |
| S7 | 0.3 |
| S8 | 0.3 |
| S9 | 0.3 |
| S10 | 0.6 |
| S11 | 0.7 |
| S12 | 0.7 |

**Deferred (not fixing in this round):**
- W14: Sparkle/auto-update (needs package dependency, do at release time)
- M10: Hover video preview on cards (nice-to-have, complex AVPlayer management per card)
- M12: "Your Downloads" shelf (low value, can add later)
- V6: Server URL validation button (low priority)
- C10-C13: Sendable conformance cleanup (low risk, compiler will catch issues)
- C14-C16: Duplicate code extraction (refactor, no user-facing impact)
- C21: Persist selected sidebar section (minor polish)

**Total: 77 issues addressed, 10 deferred.**
