# ScreenSpace Full Audit

Complete inventory of every issue found across the app and server. Organized by priority, grouped by theme. Each issue has a unique ID for tracking.

**Source:** Code review + visual inspection of running app (2026-03-23)
**Scope:** All Swift files in `app/Sources/ScreenSpace/`, all Go files in `server/`, and visual UI review

---

## Section 1: Disconnected UI (from original wiring plan)

Already documented in `2026-03-23-wiring-fixes.md`. Validated, all 14 claims confirmed accurate.

| ID | File | Issue | Plan Task |
|----|------|-------|-----------|
| W1 | LibraryView.swift:93 | "Set as Wallpaper" empty, never calls engine | Task 2 |
| W2 | DetailView.swift:102 | "Set as Wallpaper" empty, never downloads or calls engine | Task 2 |
| W3 | WallpaperCard.swift | Card click does nothing, no navigation to DetailView | Task 3 |
| W4 | HomeView.swift:20-29 | All data is hardcoded placeholders, never calls API | Task 4 |
| W5 | GalleryWindow.swift:103 | Admin tab visible to all users, no role check | Task 6 |
| W6 | HeroSection.swift:49 | "View Wallpaper" button empty action | Task 7 |
| W7 | HeroSection.swift:55 | Favorite heart button empty action | Task 7 |
| W8 | DetailView.swift:73 | Favorite heart button empty action | Task 2 |
| W9 | DetailView.swift:79 | Report flag button empty action | Task 2 |
| W10 | ShelfRow.swift:16 | "See All" button empty action | Task 7 |
| W11 | ExploreView.swift | Entire view is placeholder text | Task 5 |
| W12 | DetailView.swift:5-6 | Download progress state never updates | Task 2 |
| W13 | App.swift:87-89 | "Next Wallpaper" menu item empty | Task 8 |
| W14 | UpdateManager.swift:14 | checkForUpdates() empty, Sparkle not integrated | Deferred |

---

## Section 2: Missing Spec Features (not in wiring plan)

Backend services exist but have zero connection to the UI or engine lifecycle.

### **Engine/Lifecycle**

| ID | Issue | Severity |
|----|-------|----------|
| M1 | **PauseController never connected to WallpaperEngine.** PauseController detects battery/sleep/lock but nothing subscribes to `shouldPause` and calls `engine.pauseAll()`/`resumeAll()`. Auto-pause is completely dead. | Critical |
| M2 | **Fullscreen occlusion pause missing.** Config has `pauseOnFullscreen` toggle, PauseController never checks `NSWindow.occlusionState`. Spec explicitly describes this. | Critical |
| M3 | **No wallpaper restored on launch.** `engine.start()` creates windows but plays nothing. Config stores `lastPlayedURL` but nobody reads it at startup. App launches with blank desktops. | Critical |
| M4 | **"Now playing" menu item never updates.** Hardcoded to "No wallpaper active". Nothing writes current wallpaper info to the status menu. | High |

### **Missing UI for Existing Backends**

| ID | Issue | Severity |
|----|-------|----------|
| M5 | **No playlist UI anywhere.** PlaylistManager has full CRUD but zero UI. No way to create, view, edit, or assign playlists. Spec describes interval rotation, shuffle, per-display playlists. | High |
| M6 | **Displays tab is read-only.** Shows display names but can't assign wallpapers or playlists. Says "Assign from Library tab" but Library doesn't support this either. | High |
| M7 | **No favorites view.** API has `listFavorites()`, heart buttons exist (empty). No UI to see your favorited wallpapers. | High |
| M8 | **No lock screen button anywhere.** LockScreenManager is fully implemented but no UI trigger exists. | Medium |
| M9 | **Screensaver never installed.** ScreenSpaceSaver.swift exists but nothing copies the `.saver` bundle to `~/Library/Screen Savers/`. No install button. | Medium |

### **Missing Detail/Polish Features from Spec**

| ID | Issue | Severity |
|----|-------|----------|
| M10 | **No hover video preview on cards.** Spec says "10s low-res preview video autoplays via AVPlayer" on hover. Card only does scale effect. | Low |
| M11 | **No video playback in DetailView.** Just a static play icon. Spec says "full video playback preview." | Medium |
| M12 | **No "Your Downloads" shelf in HomeView.** Spec mentions this as a content shelf. | Low |
| M13 | **No cache eviction.** Config has `cacheSizeLimitMB` and `clearCache()` works, but nothing auto-evicts when limit is hit. | Medium |
| M14 | **No thumbnails for local videos in Library.** ThumbnailGenerator exists but library cards show static gray rectangles. | Medium |

---

## Section 3: Visual/UX Issues (from screenshots)

### **Settings Panel**

| ID | Issue | Severity |
|----|-------|----------|
| V1 | **Content vertically centered with massive empty space.** Form controls float in the middle of the panel on every tab. Should be top-aligned. | High |
| V2 | **Double border effect.** Visible inner outline/border inside the sheet, creating a nested rectangle. Styling artifact. | Medium |
| V3 | **Displays tab: raw display ID exposed.** Shows "1552-41041-4251086178" to user. Internal stable ID should be hidden. | Medium |
| V4 | **Storage tab: "5,120 MB" instead of "5 GB".** Should use human-readable formatting. | Low |
| V5 | **Account tab: no login button.** Just passive text "Not logged in". Dead end. (Covered by wiring plan Task 9 but UX is confusing.) | High |
| V6 | **General tab: no server URL validation or test button.** User can type anything with no feedback. | Low |
| V7 | **General tab: version hardcoded.** `Text("Version 0.1.0")` instead of reading from bundle. | Low |
| V8 | **No app branding in titlebar.** Spec says "ScreenSpace logo" in top nav. Title is hidden. | Low |

### **Upload Sheet**

| ID | Issue | Severity |
|----|-------|----------|
| V9 | **"content policy" text is not a link.** Spec says it should be linked. Currently plain text in toggle label. | Medium |
| V10 | **No auth gate on upload.** Sheet opens for anyone. Upload will fail silently at API. Should require login first. | High |
| V11 | **No file size/format validation shown.** Spec says max 200MB, max 60s, min 1080p. No feedback about limits. | Medium |
| V12 | **Category is freetext, not a picker.** Spec defines fixed categories. Dropdown would prevent typos. | Medium |
| V13 | **Same double-border artifact as settings.** | Medium |

### **Home/Gallery**

| ID | Issue | Severity |
|----|-------|----------|
| V14 | **Hero section title clips on narrow windows.** "Sea..." truncated. No responsive text handling. | Low |
| V15 | **No library video deletion.** Can add via drag-and-drop but no way to remove. | Medium |
| V16 | **Upload button in toolbar always visible.** Spec says upload requires auth. Should be hidden or gated. | Medium |

---

## Section 4: Language and Formatting Inconsistencies

### **UI Text**

| ID | Issue | Details |
|----|-------|---------|
| L1 | **Mixed action verbs for same concept.** | "Set as Wallpaper" (LibraryView, DetailView) vs "View Wallpaper" (HeroSection). Both lead to applying a wallpaper. |
| L2 | **"Drop MP4 or MOV files here" vs upload accepts "Video File".** | LibraryView says formats, UploadView says generic "Video File". Should be consistent. |
| L3 | **"Not logged in" is passive, unhelpful.** | Account tab just states fact. Should say "Log in to upload and favorite wallpapers" with a button. |
| L4 | **"Login from the gallery" is wrong.** | Account tab says login from gallery, but there's no login in the gallery either. Misleading. |
| L5 | **Inconsistent section casing.** | Sidebar: "Browse", "Your Stuff", "Manage" (title case). ShelfRow titles: "Popular", "Recently Added" (title case). Consistent, but "See All" button uses title case while other buttons don't. |

### **Typography Inconsistencies**

| ID | Issue | Details |
|----|-------|---------|
| L6 | **Mixed heading levels for equivalent content.** | LibraryView "Your Library" = `.title2`. ShelfRow titles = `.title3`. ExploreView "Explore" = `.title2`. No defined hierarchy. |
| L7 | **Mixed caption levels.** | Some metadata uses `.caption`, others `.caption2`. WallpaperCard duration = `.caption2`, HeroSection duration = `.caption`. |
| L8 | **No consistent spacing scale.** | Spacing values across files: 4, 6, 8, 10, 12, 14, 16, 20, 24, 28. Should follow a defined scale (e.g., 4/8/12/16/24/32). |

### **Button Style Inconsistencies**

| ID | Issue | Details |
|----|-------|---------|
| L9 | **Primary action buttons use two different styles.** | "Set as Wallpaper" in LibraryView = `.bordered`, in DetailView = `.borderedProminent`. Same action, different visual weight. |
| L10 | **"See All" and admin tabs use `.plain` while similar nav actions use `.bordered`.** | Inconsistent interactive affordance. |

---

## Section 5: Code Quality Issues

### **Architecture**

| ID | File | Issue | Severity |
|----|------|-------|----------|
| C1 | UploadView.swift:17 | **Creates its own `APIClient()` instead of shared state.** Works because KeychainHelper is global, but will break when AppState is introduced. | Medium |
| C2 | App.swift:19 | **`WallpaperEngine` created directly in AppDelegate.** Not accessible to SwiftUI views. Root cause of all wiring issues. | High |
| C3 | SettingsView.swift:5 | **Reads config directly from `ConfigManager.shared`.** Other views will need the same. No single source of truth. | Medium |
| C4 | AppConfig.swift | **`videoQuality` field declared but never used anywhere.** | Low |
| C5 | AppConfig.swift | **`screenAssignments` dict declared but never populated or read.** | Low |

### **Error Handling**

| ID | File | Issue | Severity |
|----|------|-------|----------|
| C6 | SettingsView.swift:28 | `try? SMAppService.mainApp.register()` silently fails. User thinks launch-at-login is on but it isn't. | Medium |
| C7 | SettingsView.swift:85 | `try? CacheManager.shared.clearCache()` sets size to 0 even on failure. | Low |
| C8 | LibraryView.swift:84 | `try? VideoImporter.importVideo()` silently drops import errors from drag-and-drop. | Medium |
| C9 | UploadView.swift:121 | `URL(string: initResponse.uploadURL)!` force unwrap. Crashes on malformed server response. | High |

### **Thread Safety / Sendable**

| ID | File | Issue | Severity |
|----|------|-------|----------|
| C10 | APIClient.swift:3 | `@unchecked Sendable` instead of proper conformance. | Low |
| C11 | ConfigManager.swift:3 | `@unchecked Sendable` instead of proper conformance. | Low |
| C12 | CacheManager.swift:3-4 | `Sendable` class with `@MainActor static let shared`. Conflicting isolation. | Medium |
| C13 | VideoImporter, ThumbnailGenerator | Not marked `Sendable`. Used across concurrency boundaries. | Low |

### **Duplicate Code**

| ID | Issue | Details |
|----|-------|---------|
| C14 | "Set as Wallpaper" button appears in DetailView and LibraryView with different styles. | Should be extracted to shared component. |
| C15 | AVPlayer setup duplicated between WallpaperWindow.swift and ScreenSpaceSaver.swift. | |
| C16 | Admin action handlers (approve/reject/ban/unban) all follow identical error pattern. | Should be extracted to helper. |

### **Missing Standard macOS Patterns**

| ID | Issue | Severity |
|----|-------|----------|
| C17 | **No accessibility labels** on wallpaper cards, resolution badges, metadata items, admin tabs. | High |
| C18 | **No keyboard shortcuts** for upload (Cmd+U), settings (Cmd+,), next wallpaper, favorites. | Medium |
| C19 | **No context menus** on wallpaper cards (download, favorite, report, set as wallpaper). | Medium |
| C20 | **No VoiceOver roles** on interactive elements (cards as buttons, drop zone, flow layout). | High |
| C21 | **`selectedSection` not persisted** between app launches. Always resets to Home. | Low |

---

## Section 6: Server Issues

### **Critical**

| ID | File | Issue |
|----|------|-------|
| S1 | service/auth.go:63-66 | **JWT token claims panic on malformed token.** No type assertion error handling. Crashes server. |

### **High**

| ID | File | Issue |
|----|------|-------|
| S2 | handler/wallpaper.go:355 | **Download count incremented on metadata view, not actual download.** Inflates popularity metrics. |
| S3 | handler/admin.go:226-272 | **Ban/unban/promote don't check if user exists.** Silent failure, no 404. |
| S4 | handler/admin.go:86-107 | **Rejection reason not stored in DB.** Admin provides reason but it's lost. No `rejection_reason` column. |

### **Medium**

| ID | File | Issue |
|----|------|-------|
| S5 | repository/wallpaper.go:128 | **Category filter is case-sensitive.** "Nature" != "nature". Should use ILIKE. |
| S6 | (no validation) | **Categories are freetext.** No enum, no validation. Users can create arbitrary categories via upload. |
| S7 | handler/report.go:62-65 | **No max length on report reason.** Can submit 1MB string. |
| S8 | handler/wallpaper.go:62-65 | **No max length on wallpaper title.** |
| S9 | handler/wallpaper.go:38-42 | **Tags array unbounded.** No limit on count or individual tag length. |
| S10 | middleware/ratelimit.go:27-42 | **Rate limiter map grows indefinitely.** No cleanup of stale entries. Memory leak over time. |
| S11 | handler/wallpaper.go:94 | **Pre-signed upload URL expires after 15 minutes.** May expire mid-upload for large files (200MB). |
| S12 | (all mutation endpoints) | **No audit logging.** Admins can't see who did what. |

---

## Issue Counts

| Section | Critical | High | Medium | Low | Total |
|---------|----------|------|--------|-----|-------|
| 1. Disconnected UI (wiring plan) | - | - | - | - | 14 |
| 2. Missing spec features | 3 | 4 | 4 | 3 | 14 |
| 3. Visual/UX | - | 3 | 7 | 6 | 16 |
| 4. Language/formatting | - | - | 2 | 8 | 10 |
| 5. Code quality | - | 2 | 7 | 12 | 21 |
| 6. Server | 1 | 3 | 8 | 0 | 12 |
| **Total** | **4** | **12** | **28** | **29** | **87** |
