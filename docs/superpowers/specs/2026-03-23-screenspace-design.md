# ScreenSpace Design Spec

Open-source macOS live wallpaper app. Free alternative to Wallspace with no paywalls, community gallery, and native feel.

## Overview

A macOS menu bar app with a full gallery window for browsing, managing, and applying live video wallpapers. Ships with zero bundled content. Users bring their own MP4/MOV files or browse a community gallery. All features are free.

**Target audience:** Mac power users who want live wallpapers without paywalls, and developers who want to self-host or contribute.

**Distribution:** GitHub Releases + Homebrew cask. No App Store (avoids sandbox restrictions needed for desktop-level window and lock screen access).

**Minimum macOS:** 15.0 (Sequoia).

## Architecture

Monorepo with two independent binaries:

```
screenspace/
  app/              # Xcode project, Swift macOS app + .saver bundle
  server/           # Go API server
  docs/
```

### Tech Stack

| Layer | Tech |
|---|---|
| App UI | SwiftUI (gallery, settings) + AppKit (menu bar, wallpaper windows) |
| Menu bar | AppKit `NSStatusItem` |
| Wallpaper engine | AppKit `NSWindow` + AVKit `AVPlayer` + `AVPlayerLayer` |
| GPU rendering | Metal (via AVFoundation hardware decode on Apple Silicon) |
| Multi-monitor | `NSScreen` enumeration, `CGDirectDisplayID` keying |
| Lock screen (macOS 15) | `FileManager` + frame extraction via `AVAssetImageGenerator` |
| Lock screen (macOS 26) | Reverse-engineered `WallpaperAerialsExtension` XPC protocol |
| Screensaver | `ScreenSaver` framework, `.saver` bundle with embedded `AVPlayer` |
| Playlists/config | JSON files in `~/Library/Application Support/ScreenSpace/` |
| Keychain | JWT token storage for community gallery auth |
| Build system | Xcode + Swift Package Manager |
| API server | Go + `net/http` |
| Storage | S3-compatible (abstracted interface, swappable between S3/Hetzner Object Storage/R2) |
| Database | Postgres |
| Server deploy | Docker on Hetzner (or anywhere) |

---

## 1. Wallpaper Engine

Renders video behind desktop icons on each display.

### Components

**WallpaperWindow** - Subclass of `NSWindow`
- Borderless: `NSWindow.StyleMask.borderless`
- Non-activating, non-interactive (ignores all events)
- Window level: `NSWindow.Level(rawValue: Int(CGWindowLevelForKey(.desktopWindow)) + 1)`
- Spans the full frame of its assigned `NSScreen`
- Content view is layer-backed (`wantsLayer = true`) with `AVPlayerLayer` as the backing layer
- Listens for `NSApplication.didChangeScreenParametersNotification` to handle display changes

**WallpaperEngine** - Manages one `WallpaperWindow` per connected display
- Maintains a map of `CGDirectDisplayID -> WallpaperWindow`
- Display ID extracted via `screen.deviceDescription[NSDeviceDescriptionKey("NSScreenNumber")]` cast to `CGDirectDisplayID`
- Handles play/pause/swap operations
- Auto-pause triggers:
  - Battery mode: `IOPSCopyPowerSourcesInfo()` + `IOPSGetProvidingPowerSourceType()`, compare against `kIOPSBatteryPowerValue`
  - Low power mode: `ProcessInfo.processInfo.isLowPowerModeEnabled` + `NSProcessInfoPowerStateDidChange` notification
  - Screen locked: `DistributedNotificationCenter` observing `"com.apple.screenIsLocked"` / `"com.apple.screenIsUnlocked"`
  - Sleep/wake: `NSWorkspace.willSleepNotification` / `NSWorkspace.didWakeNotification`
  - Fullscreen app active: `NSWorkspace.activeSpaceDidChangeNotification`
  - Screensaver running: detect and defer to screensaver
- Auto-resumes when conditions clear. Resumes from current position, not restart.

**Video Playback**
- `AVQueuePlayer` with `AVPlayerLooper` for seamless looping
- Hardware-accelerated decoding is automatic via AVFoundation/VideoToolbox on Apple Silicon
- Accepted formats: MP4 (H.264, H.265), MOV

**Performance budget:** Under 2% CPU, under 100MB RAM per display with a single 4K wallpaper.

---

## 2. Playlists & Configuration

All state stored locally as JSON.

### File Layout

```
~/Library/Application Support/ScreenSpace/
  config.json
  playlists/
    default.json
    <user-created>.json
  cache/
    thumbnails/
    community/          # Downloaded community wallpapers
```

### Playlist Model

```json
{
  "id": "uuid",
  "name": "My Playlist",
  "items": [
    { "path": "/Users/.../wallpaper.mp4", "source": "local" },
    { "id": "community-uuid", "source": "community" }
  ],
  "interval": 1800,
  "shuffle": false
}
```

- Each display can have its own playlist or share one
- `interval` is seconds between wallpaper changes (0 = no auto-rotate)
- Community-sourced wallpapers store a reference ID, resolved to a cached local file or re-downloaded
- Thumbnails generated on import via `AVAssetImageGenerator.images(for:)` (modern async API)

### Config Model

```json
{
  "launchAtLogin": true,
  "pauseOnBattery": true,
  "pauseOnFullscreen": true,
  "videoQuality": "original",
  "cacheSizeLimitMB": 5120,
  "screenAssignments": {
    "<CGDirectDisplayID>": "<playlist-id>"
  }
}
```

- Launch at login via `SMAppService.mainApp` (macOS 13+, replaces deprecated `SMLoginItemSetEnabled`)
- Serialized with `Codable` / `JSONEncoder` / `JSONDecoder`

---

## 3. UI Design

### App Structure

- **Menu bar icon** (`NSStatusItem`): Primary always-on presence. Quick controls for play/pause, skip, current wallpaper info.
- **Full app window**: The main experience for browsing, exploring, and managing wallpapers. Opens from menu bar or on first launch.
- **No dock icon**: `LSUIElement = YES` in Info.plist (or `NSApp.setActivationPolicy(.accessory)` at runtime)

### Gallery Window (Netflix/Apple TV Style)

**Top navigation bar:**
- ScreenSpace logo
- Home / Explore / Library tabs
- Upload button
- Settings gear

**Hero section:**
- Full-width featured wallpaper with video playing as background
- Glass overlay card showing title, resolution, file size
- "View Wallpaper" and favorite buttons

**Content shelves:**
- Horizontal scrollable rows: "Popular", "Recently Added", "Your Downloads", category rows
- Each wallpaper card shows a thumbnail
- On hover (`.onHover` modifier): card scales up slightly, 10s low-res preview video autoplays via `AVPlayer`

**Detail view:**
- Full video playback preview
- Glass overlay for metadata, download button, "Set as Wallpaper" button

### Glass/Native Feel

**macOS 26+ (Tahoe):**
- `.glassEffect()` modifier for sidebar, cards, overlays
- `GlassEffectContainer` to group glass elements (glass cannot sample other glass)
- Liquid Glass with lensing effect

**macOS 15-25 (fallback):**
- SwiftUI `.ultraThinMaterial` / `.thinMaterial` backgrounds
- Uses `NSVisualEffectView` under the hood
- Still glassy, just not the new Liquid Glass

---

## 4. Community Gallery & Backend

Go API server with S3-compatible storage.

### API Endpoints

```
POST   /api/v1/wallpapers              # Upload (authenticated)
GET    /api/v1/wallpapers              # Browse (public, paginated)
GET    /api/v1/wallpapers/:id          # Metadata + pre-signed download URL
DELETE /api/v1/wallpapers/:id          # Remove (admin/uploader only)
GET    /api/v1/wallpapers/popular      # Sorted by download count
GET    /api/v1/wallpapers/recent       # Sorted by upload date
POST   /api/v1/wallpapers/:id/report   # Flag content

POST   /api/v1/auth/register           # Create account (email + password)
POST   /api/v1/auth/login              # Get JWT token
GET    /api/v1/auth/me                 # Current user info

GET    /api/v1/admin/queue             # Pending uploads (admin only)
POST   /api/v1/admin/queue/:id/approve
POST   /api/v1/admin/queue/:id/reject
```

### Upload Flow

1. User uploads MP4/MOV from the macOS app (or web form later)
2. Server validates: format (H.264/H.265), max 200MB, max 60s duration, min 1080p resolution
3. Server generates thumbnail and 10s low-res preview clip via `ffmpeg`
4. Wallpaper enters `pending` state in the database
5. Admin reviews in moderation queue, approves or rejects
6. On approval, wallpaper becomes visible in browse/search

### Storage Layout

```
bucket/
  wallpapers/<id>/original.mp4
  wallpapers/<id>/thumbnail.jpg
  wallpapers/<id>/preview.mp4
```

### Database Schema (Postgres)

```sql
wallpapers (
  id            UUID PRIMARY KEY,
  title         TEXT NOT NULL,
  uploader_id   UUID REFERENCES users(id),
  status        TEXT NOT NULL DEFAULT 'pending',  -- pending/approved/rejected
  resolution    TEXT NOT NULL,                     -- e.g. "3840x2160"
  duration      FLOAT NOT NULL,                   -- seconds
  file_size     BIGINT NOT NULL,                  -- bytes
  format        TEXT NOT NULL,                     -- h264/h265
  download_count BIGINT DEFAULT 0,
  storage_key   TEXT NOT NULL,
  thumbnail_key TEXT NOT NULL,
  preview_key   TEXT NOT NULL,
  created_at    TIMESTAMPTZ DEFAULT now(),
  updated_at    TIMESTAMPTZ DEFAULT now()
)

users (
  id            UUID PRIMARY KEY,
  email         TEXT UNIQUE NOT NULL,
  password_hash TEXT NOT NULL,
  role          TEXT NOT NULL DEFAULT 'user',      -- user/admin
  created_at    TIMESTAMPTZ DEFAULT now()
)

reports (
  id            UUID PRIMARY KEY,
  wallpaper_id  UUID REFERENCES wallpapers(id),
  reporter_id   UUID REFERENCES users(id),
  reason        TEXT NOT NULL,
  created_at    TIMESTAMPTZ DEFAULT now()
)
```

### Storage Abstraction

Go interface for swappable storage backends:

```go
type Store interface {
    Put(ctx context.Context, key string, reader io.Reader) error
    Get(ctx context.Context, key string) (io.ReadCloser, error)
    List(ctx context.Context, prefix string) ([]string, error)
    Delete(ctx context.Context, key string) error
    PreSignedURL(ctx context.Context, key string, expiry time.Duration) (string, error)
}
```

Implemented for S3-compatible storage. Swapping providers is a config change.

### Auth

- Simple JWT (no OAuth for v1)
- macOS app stores token in Keychain
- Rate limiting on uploads: 5/day per user

### Downloads

- App downloads directly from storage via pre-signed S3 URLs, not through the API server
- Reduces server load and leverages CDN/storage provider bandwidth

---

## 5. Lock Screen

### macOS 15 (Sequoia) - Static Frame

No public API for lock screen wallpapers. Workaround:

1. Extract a high-quality still frame using `AVAssetImageGenerator.image(at:)` at a user-selectable timestamp (default 2s)
2. Write image to `/Library/Caches/Desktop Pictures/<UUID>/lockscreen.png`
3. UUID discovered by scanning that directory for the current user's folder
4. Requires elevated permissions, prompt user once with explanation

**Caveats surfaced to user:**
- Lock screen is a static frame, not video (macOS limitation)
- May reset after macOS updates
- Requires write permission to `/Library/Caches`
- Optional feature, app works fully without it

### macOS 26 (Tahoe) - Live Video

macOS 26 introduced animated lock screen wallpapers via `WallpaperAerialsExtension`, which communicates with System Settings via an undocumented XPC protocol.

- No public API exists
- Wallspace and Backdrop both reverse-engineered this for their lock screen features
- Uses `.MOV` files at 4K HDR
- We will reverse-engineer the XPC protocol to inject custom videos
- Fragile, may break on macOS updates, requires maintenance

**v1 plan:** Ship static lock screen on macOS 15. Live lock screen on macOS 26 as a stretch goal once we reverse-engineer the protocol.

---

## 6. Screensaver

Proper Apple-supported API via the `ScreenSaver` framework.

### Implementation

- `ScreenSpaceSaver.saver` bundle, built as a separate Xcode target
- Subclasses `ScreenSaverView`
- Embeds `AVPlayerLayer` in a layer-backed view for video playback
- Reads playlist/config from `~/Library/Application Support/ScreenSpace/config.json`
- Installed to `~/Library/Screen Savers/` by the main app
- User selects "ScreenSpace" in System Settings > Screen Saver

### Features

- Plays the current wallpaper or cycles through the active playlist
- Respects shuffle/order settings from the main app
- Falls back to a subtle dark gradient if no wallpaper is configured
- Shares the video cache with the main app (no duplicate downloads)

### Communication

- No IPC needed
- Screensaver reads config JSON at launch, main app writes it
- Video files live in the shared cache directory

---

## 7. Multi-Monitor

- Enumerate `NSScreen.screens` on launch to discover all connected displays
- Create one `WallpaperWindow` per display, keyed by `CGDirectDisplayID`
- Each display can have an independent wallpaper or playlist
- Listen for `NSApplication.didChangeScreenParametersNotification` to detect displays added/removed/rearranged
- Display added: create new window, apply default or last-used wallpaper
- Display removed: tear down window, release `AVPlayer`
- Settings UI shows a visual display arrangement (like System Settings > Displays) for per-screen wallpaper assignment
- If display is mirrored or lid-closed, don't play (avoid wasting resources)

---

## 8. Build & Distribution

### macOS App

- Xcode project with two targets: main app + `.saver` bundle
- Code signed with Developer ID certificate
- Notarized via Apple's notary service (required for non-App Store distribution since macOS 10.15)
- Distributed as `.dmg` via GitHub Releases
- Homebrew cask for `brew install --cask screenspace`

### Go Server

- Single binary, compiled with `go build`
- Docker image for deployment
- Environment-based configuration (storage credentials, database URL, JWT secret)
- Deploy to Hetzner VPS (or anywhere Docker runs)

### Non-Sandboxed

The app is non-sandboxed because:
- Desktop-level `NSWindow` requires placing windows at `kCGDesktopWindowLevel`
- Lock screen modification requires filesystem access to `/Library/Caches`
- Screensaver installation requires writing to `~/Library/Screen Savers/`

This means we cannot distribute via the Mac App Store, hence GitHub + Homebrew.

Notarization is still required and achievable for non-sandboxed apps.
