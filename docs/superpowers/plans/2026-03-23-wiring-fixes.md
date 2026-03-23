# Wiring Fixes Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Wire all disconnected UI shells to their backends. Every button, view, and flow should work end-to-end.

**Architecture:** Add shared app state via `@Observable` class, pass WallpaperEngine and APIClient through SwiftUI environment. Connect views to real data.

**Spec:** `docs/superpowers/specs/2026-03-23-screenspace-design.md`

---

## Audit Results (25 issues across 9 files)

### Priority 1: Core Flow (must work for MVP)

| # | File | Issue |
|---|---|---|
| 1 | `LibraryView.swift:93` | "Set as Wallpaper" button is empty. Never calls WallpaperEngine. |
| 2 | `DetailView.swift:102` | "Set as Wallpaper" is empty. Never downloads or calls engine. |
| 3 | `WallpaperCard.swift` | Clicking a card does nothing. No navigation to DetailView. |
| 4 | `HomeView.swift` | All data is hardcoded placeholders. Never calls API. |
| 5 | `GalleryWindow.swift:103` | Admin tab visible to all users. No role check. |

### Priority 2: Buttons That Do Nothing

| # | File | Issue |
|---|---|---|
| 6 | `HeroSection.swift:49` | "View Wallpaper" button empty action. |
| 7 | `HeroSection.swift:55` | Favorite heart button empty action. |
| 8 | `DetailView.swift:73` | Favorite heart button empty action. |
| 9 | `DetailView.swift:79` | Report flag button empty action. |
| 10 | `ShelfRow.swift:16` | "See All" button empty action. |

### Priority 3: Incomplete Views

| # | File | Issue |
|---|---|---|
| 11 | `ExploreView.swift` | Entire view is placeholder text. No categories, no API calls. |
| 12 | `DetailView.swift:5-6` | Download progress state never updates. |

### Priority 4: Menu Bar & Misc

| # | File | Issue |
|---|---|---|
| 13 | `App.swift:88` | "Next Wallpaper" menu item empty. Not wired to playlist. |
| 14 | `UpdateManager.swift:14` | checkForUpdates() is empty. Sparkle not integrated. |

---

## Task 1: Shared App State

**Problem:** Views can't talk to WallpaperEngine or APIClient because there's no shared state.

**Files:**
- Create: `app/Sources/ScreenSpace/AppState.swift`

- [ ] **Step 1: Create observable app state**

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

    var currentUser: UserResponse?
    var isLoggedIn: Bool { currentUser != nil }
    var isAdmin: Bool { currentUser?.role == "admin" }
    var currentWallpaperURL: URL?

    init() {
        self.configManager = .shared
        self.playlistManager = .shared
        self.engine = WallpaperEngine(configManager: configManager)
        self.api = APIClient()
        self.lockScreen = LockScreenManager()
    }

    func setWallpaper(url: URL) {
        engine.setWallpaperOnAllDisplays(url: url)
        currentWallpaperURL = url
        try? configManager.update { $0.lastPlayedURL = url.absoluteString }
    }

    func setWallpaper(url: URL, forDisplay displayID: String) {
        engine.setWallpaper(url: url, forDisplay: displayID)
        currentWallpaperURL = url
        try? configManager.update { $0.lastPlayedURL = url.absoluteString }
    }

    func login(email: String, password: String) async throws {
        let response = try await api.login(email: email, password: password)
        currentUser = try await api.me()
    }

    func register(email: String, password: String) async throws {
        let response = try await api.register(email: email, password: password)
        currentUser = try await api.me()
    }

    func logout() {
        api.logout()
        currentUser = nil
    }

    func restoreSession() async {
        guard KeychainHelper.loadToken() != nil else { return }
        currentUser = try? await api.me()
    }
}
```

- [ ] **Step 2: Inject into App.swift**

In AppDelegate, create `AppState` and inject into SwiftUI via `.environment()` on the hosting view. Call `engine.start()` on launch. Call `restoreSession()` to check for existing login.

- [ ] **Step 3: Commit**

---

## Task 2: Wire "Set as Wallpaper" (Issues #1, #2)

**Files:**
- Modify: `app/Sources/ScreenSpace/UI/Views/LibraryView.swift`
- Modify: `app/Sources/ScreenSpace/UI/Views/DetailView.swift`

- [ ] **Step 1: LibraryView - wire to engine**

Add `@Environment(AppState.self) var appState` to LibraryView. In `setWallpaper(url:)`, call `appState.setWallpaper(url: url)`.

- [ ] **Step 2: DetailView - wire download + set**

Add `@Environment(AppState.self) var appState`. In `setAsWallpaper()`:
- If wallpaper has a local cache hit (via CacheManager), set directly
- Otherwise download via pre-signed URL, show progress, cache, then set
- Update `isDownloading` and `downloadProgress` state during download
- Wire favorite button to `appState.api.toggleFavorite(id:)`
- Wire report button to `appState.api.reportWallpaper(id:reason:)`

- [ ] **Step 3: Commit**

---

## Task 3: Wire Navigation - Card to Detail (Issue #3)

**Files:**
- Modify: `app/Sources/ScreenSpace/UI/Components/WallpaperCard.swift`
- Modify: `app/Sources/ScreenSpace/UI/Components/ShelfRow.swift`
- Modify: `app/Sources/ScreenSpace/UI/Views/HomeView.swift`

- [ ] **Step 1: Add onTap callback to WallpaperCard**

Add `var onTap: (() -> Void)?` to WallpaperCard. Wrap the card body in a `Button` with `.plain` style that calls `onTap`.

- [ ] **Step 2: ShelfRow passes through the callback**

Add `var onSelectWallpaper: ((WallpaperCardData) -> Void)?` to ShelfRow. Pass to each WallpaperCard's `onTap`.

- [ ] **Step 3: HomeView shows DetailView in sheet**

Add `@State var selectedWallpaper: WallpaperResponse?` and show a `.sheet` with DetailView when set. Convert `WallpaperCardData` tap into an API call to get full `WallpaperResponse`, then show detail.

- [ ] **Step 4: Commit**

---

## Task 4: Wire HomeView to API (Issue #4)

**Files:**
- Modify: `app/Sources/ScreenSpace/UI/Views/HomeView.swift`

- [ ] **Step 1: Replace placeholder data with API calls**

Add `@Environment(AppState.self) var appState`. Add state:
```swift
@State private var popular: [WallpaperCardData] = []
@State private var recent: [WallpaperCardData] = []
@State private var featured: WallpaperCardData?
@State private var isLoading = true
```

In `.task`, call `appState.api.popularWallpapers()` and `appState.api.recentWallpapers()`. Map `WallpaperResponse` to `WallpaperCardData`. Set `featured` to first popular wallpaper.

- [ ] **Step 2: Show loading state**

While loading, show a `ProgressView`. On error (no server), show a message like "Connect to a server in Settings to browse community wallpapers" instead of crashing.

- [ ] **Step 3: Keep placeholder for offline/no-server**

If API calls fail, fall back to showing the local library instead of empty state.

- [ ] **Step 4: Commit**

---

## Task 5: Wire ExploreView (Issue #11)

**Files:**
- Modify: `app/Sources/ScreenSpace/UI/Views/ExploreView.swift`

- [ ] **Step 1: Build category browser**

Add categories: "Nature", "Abstract", "Urban", "Cinematic", "Space", "Underwater". Display as a grid of cards. On tap, fetch wallpapers filtered by category via `appState.api.listWallpapers(category:)`. Show results in a grid below.

- [ ] **Step 2: Add search**

Add a search field at the top. On submit, call `appState.api.listWallpapers(query:)`.

- [ ] **Step 3: Commit**

---

## Task 6: Hide Admin for Non-Admins (Issue #5)

**Files:**
- Modify: `app/Sources/ScreenSpace/UI/GalleryWindow.swift`

- [ ] **Step 1: Conditionally show admin sidebar item**

Add `@Environment(AppState.self) var appState` to GalleryContentView. Only show the "Manage" section in the sidebar when `appState.isAdmin` is true.

```swift
if appState.isAdmin {
    Section("Manage") {
        Label("Admin", systemImage: "shield")
            .tag(GallerySection.admin)
    }
}
```

- [ ] **Step 2: Commit**

---

## Task 7: Wire Hero + Shelf Buttons (Issues #6, #7, #10)

**Files:**
- Modify: `app/Sources/ScreenSpace/UI/Components/HeroSection.swift`
- Modify: `app/Sources/ScreenSpace/UI/Components/ShelfRow.swift`

- [ ] **Step 1: HeroSection callbacks**

Add `onViewWallpaper: (() -> Void)?` and `onFavorite: (() -> Void)?` callbacks. Wire to the buttons.

- [ ] **Step 2: ShelfRow "See All" callback**

Add `onSeeAll: (() -> Void)?` callback. Wire to the button. Parent view navigates to a filtered list.

- [ ] **Step 3: Commit**

---

## Task 8: Wire Menu Bar "Next Wallpaper" (Issue #13)

**Files:**
- Modify: `app/Sources/ScreenSpace/App.swift`

- [ ] **Step 1: Implement skipToNext**

Get the current playlist from PlaylistManager. Advance to the next item. Call `appState.setWallpaper(url:)` with the next video URL.

- [ ] **Step 2: Commit**

---

## Task 9: Add Login UI

**Problem:** Users can't log in. There's a logout button in settings but no login flow.

**Files:**
- Create: `app/Sources/ScreenSpace/UI/Views/LoginView.swift`
- Modify: `app/Sources/ScreenSpace/UI/Views/SettingsView.swift`

- [ ] **Step 1: Create LoginView**

Simple form: email, password, login button, register button. Calls `appState.login()` or `appState.register()`. Shows error on failure. Dismisses on success.

- [ ] **Step 2: Wire into SettingsView**

Replace the "Not logged in" text with a button that opens LoginView as a sheet. Show current user email and role when logged in.

- [ ] **Step 3: Commit**

---

## Not Fixing Now (Accepted Placeholders)

| # | Issue | Why |
|---|---|---|
| 14 | Sparkle UpdateManager empty | Needs the Sparkle package dependency added. Do when cutting first release. |
