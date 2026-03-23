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
- Window level: `NSWindow.Level(rawValue: Int(CGWindowLevelForKey(.desktopWindow)))` (exact desktop level, needs empirical tuning to confirm it sits above the static wallpaper but below Finder's desktop icon window)
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
  - Fullscreen app covering display: check `NSWindow.occlusionState` on each `WallpaperWindow`. When `.visible` is not present, the window is fully occluded (e.g. by a fullscreen app). This is per-window, so multi-monitor is handled correctly.
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
  "version": 1,
  "launchAtLogin": true,
  "pauseOnBattery": true,
  "pauseOnFullscreen": true,
  "videoQuality": "original",
  "videoGravity": "resizeAspectFill",
  "cacheSizeLimitMB": 5120,
  "serverURL": "https://api.screenspace.app",
  "screenAssignments": {
    "<stable-display-id>": "<playlist-id>"
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
- Each wallpaper card shows: thumbnail, title, resolution badge (corner overlay), duration
- Resolution badge logic: 3840x2160+ = "4K", 2560x1440 = "2K", 1920x1080 = "1080p"
- On hover (`.onHover` modifier): card scales up slightly, 10s low-res preview video autoplays via `AVPlayer`

**Detail view:**
- Full video playback preview
- Glass overlay showing: title, category, tags, resolution (e.g. "3840x2160"), file size (e.g. "85MB"), duration (e.g. "32s"), uploader name, download count
- "Set as Wallpaper" / "Download" / favorite buttons

**Video scaling:**
- `AVPlayerLayer.videoGravity` defaults to `.resizeAspectFill` (fills screen, slight crop if aspect ratio differs)
- User can toggle to `.resizeAspect` in settings (fit to screen, letterboxed) if they prefer no cropping

### Glass/Native Feel

**macOS 26+ (Tahoe):**
- `.glassEffect()` modifier for sidebar, cards, overlays
- `GlassEffectContainer` to group glass elements (glass cannot sample other glass)
- Liquid Glass with lensing effect

**macOS 15 (Sequoia, pre-Tahoe fallback):**
- SwiftUI `.ultraThinMaterial` / `.thinMaterial` backgrounds
- Uses `NSVisualEffectView` under the hood
- Still glassy, just not the new Liquid Glass

---

## 4. Community Gallery & Backend

Go API server with S3-compatible storage.

### API Endpoints

```
POST   /api/v1/wallpapers              # Upload (authenticated)
GET    /api/v1/wallpapers              # Browse (public, paginated, ?q=, ?category=, ?sort=)
GET    /api/v1/wallpapers/:id          # Metadata + pre-signed download URL
DELETE /api/v1/wallpapers/:id          # Remove (admin/uploader only)
GET    /api/v1/wallpapers/popular      # Sorted by download count
GET    /api/v1/wallpapers/recent       # Sorted by upload date
POST   /api/v1/wallpapers/:id/report   # Flag content
POST   /api/v1/wallpapers/:id/favorite # Toggle favorite (authenticated)
GET    /api/v1/me/favorites            # List user's favorites

POST   /api/v1/auth/register           # Create account (email + password)
POST   /api/v1/auth/login              # Get JWT token
GET    /api/v1/auth/me                 # Current user info

GET    /api/v1/admin/queue             # Pending uploads (admin only)
POST   /api/v1/admin/queue/:id/approve
POST   /api/v1/admin/queue/:id/reject  # Includes reason field
GET    /api/v1/admin/wallpapers        # All wallpapers incl. pending/rejected
PATCH  /api/v1/admin/wallpapers/:id    # Edit metadata (title, category, tags)
GET    /api/v1/admin/users             # List/search users
POST   /api/v1/admin/users/:id/ban     # Ban user (can't upload)
POST   /api/v1/admin/users/:id/unban
POST   /api/v1/admin/users/:id/promote # Promote to admin
GET    /api/v1/admin/reports            # List flagged content
POST   /api/v1/admin/reports/:id/dismiss
```

### Upload Flow

Two-phase upload to keep large files (up to 200MB) off the Go server:

1. Client calls `POST /api/v1/wallpapers` with metadata (title, category, tags). Server creates a `pending` record and returns a pre-signed S3 upload URL.
2. Client uploads MP4/MOV directly to S3 via the pre-signed URL.
3. Client calls `POST /api/v1/wallpapers/:id/finalize` to signal upload complete.
4. Server validates the uploaded file: format (H.264/H.265), max 200MB, max 60s duration, min 1080p resolution.
5. Server generates thumbnail and 10s low-res preview clip via `ffmpeg`.
6. Wallpaper enters `pending_review` state in the database.
7. Admin reviews in moderation queue, approves or rejects.
8. On approval, wallpaper becomes visible in browse/search.

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
  category      TEXT,                           -- e.g. "nature", "abstract", "urban"
  tags          TEXT[] DEFAULT '{}',
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

favorites (
  user_id       UUID REFERENCES users(id),
  wallpaper_id  UUID REFERENCES wallpapers(id),
  created_at    TIMESTAMPTZ DEFAULT now(),
  PRIMARY KEY (user_id, wallpaper_id)
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
    PreSignedUploadURL(ctx context.Context, key string, expiry time.Duration) (string, error)
    Stat(ctx context.Context, key string) (ObjectInfo, error)
}
```

Implemented for S3-compatible storage. Swapping providers is a config change.

### Roles & Auth

**Roles:**
- **user** - Browse, download, upload (pending review), favorite, report
- **admin** - All user permissions, plus: approve/reject uploads, remove wallpapers, edit metadata, ban/unban users, promote users to admin

**Role mechanics:**
- Stored in `users.role` column (`user` or `admin`)
- JWT token includes role claim, server middleware checks role on `/admin/*` endpoints
- First account created gets `admin` role (or seed via `ADMIN_EMAIL` env var on first server boot)
- Additional admins promoted by existing admins

**Auth:**
- Simple JWT (no OAuth for v1)
- macOS app stores token in Keychain
- Rate limiting on uploads: 5/day per user
- Banned users receive a 403 on upload attempts

### Admin UI (in macOS App)

When an admin user is logged in, the gallery window shows an additional "Admin" tab with:

**Moderation Queue** - List of pending uploads with video preview, uploader info, metadata. One-click approve/reject. Reject includes a reason field sent to the uploader.

**Content Management** - Search/browse all wallpapers including rejected and pending. Remove any wallpaper. Edit metadata (title, category, tags).

**User Management** - List and search users by email. Ban/unban users. Promote users to admin.

**Reports** - List of flagged wallpapers with report reasons and reporter info. Quick actions: dismiss report, remove wallpaper, ban uploader.

### Downloads

- App downloads directly from storage via pre-signed S3 URLs, not through the API server
- Reduces server load and leverages CDN/storage provider bandwidth

### Error Handling & Offline Behavior

- Local library always works regardless of server connectivity
- Community gallery shows cached content when offline, with a subtle "offline" indicator
- Failed downloads: retry up to 3 times with exponential backoff, then surface error to user
- Interrupted downloads: use `URLSession` background download tasks which support automatic resume
- Expired pre-signed URLs: re-request from API before retrying download

---

## 5. Lock Screen

### macOS 15 (Sequoia) - Static Frame

No public API for lock screen wallpapers. Workaround:

1. Extract a high-quality still frame using `AVAssetImageGenerator.image(at:)` at a user-selectable timestamp (default 2s)
2. Write image to `/Library/Caches/Desktop Pictures/<GeneratedUID>/lockscreen.png`
3. The `GeneratedUID` is the user's directory services UUID, retrieved via `dscl . -read /Users/$USER GeneratedUID`
4. This directory is root-owned. Use `AuthorizationServices` to prompt for admin credentials and run a privileged helper to write the file. Prompt once and explain why.

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
- The screensaver is strictly read-only. It never writes to config or playlist files.
- Screensaver reads config JSON at launch, main app writes it
- If the user changes wallpaper while the screensaver is active, the change takes effect on next screensaver activation
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

---

## 9. Auto-Update

Menu bar apps are easy to forget about. Auto-update is critical, especially since the lock screen XPC integration is fragile and will need patches after macOS updates.

- Use the **Sparkle** framework (standard for non-App Store Mac apps)
- Appcast XML feed hosted alongside GitHub Releases
- Check for updates automatically on launch (configurable interval, default daily)
- User can also manually check via menu bar > "Check for Updates..."
- Updates are downloaded and applied in-background, user prompted to restart
- Delta updates supported by Sparkle to minimize download size
- Sparkle handles code signature verification automatically

---

## 10. Trust & Safety

Required before accepting any user uploads.

### Content Policy

Published as `CONTENT_POLICY.md` in the repo and linked during upload flow in the app.

- No NSFW content
- No copyrighted content without a license (uploaders must confirm they have rights)
- No hateful, violent, or illegal content
- Admins use this policy as the basis for approve/reject decisions

### DMCA Process

- Published contact email for takedown requests
- On valid DMCA notice: remove content within 24 hours, notify uploader
- Counter-notice process for disputed takedowns
- Repeat infringers get banned

### Privacy

- Server stores: email, hashed password, upload history, IP addresses (for rate limiting)
- No tracking, no analytics, no third-party data sharing
- Users can request account deletion (removes all data including uploads)
- Privacy policy published on the project website/repo
- GDPR compliant: EU users can export or delete their data on request

---

## 11. Data Migration & Stability

### Local Config Versioning

- All JSON config files include a `"version"` field (starting at 1)
- On app launch, check version and run upgrade-on-read migrations if needed
- Migrations are sequential (v1 -> v2 -> v3), each a pure function transforming the JSON
- If config is corrupted (invalid JSON), back up the broken file and reset to defaults with a user notification

### Display Identification

`CGDirectDisplayID` can change across reboots. Use a stable composite key instead:

- Primary: `CGDisplayVendorNumber` + `CGDisplayModelNumber` + `CGDisplaySerialNumber`
- Fallback: `CGDirectDisplayID` for displays without serial numbers (e.g. some third-party monitors)
- Store mapping in config as the stable ID string, resolve to current `CGDirectDisplayID` at runtime

### Postgres Migrations

- Use `golang-migrate` for schema versioning
- Migration files stored in `server/migrations/` as sequential numbered SQL files
- Migrations run automatically on server startup
- Never modify an existing migration, always create a new one

---

## 12. Self-Hosting

The community gallery server is designed to be self-hostable.

### App Configuration

- `serverURL` field in `config.json` (default: `https://api.screenspace.app`)
- Configurable in Settings > Gallery > Server URL
- App connects to one server at a time

### Server Setup

Provide a `docker-compose.yml` in the repo:

```yaml
services:
  server:
    image: ghcr.io/0x63616c/screenspace-server:latest
    ports: ["8080:8080"]
    env_file: .env
  postgres:
    image: postgres:16
    volumes: ["pgdata:/var/lib/postgresql/data"]
  minio:
    image: minio/minio
    command: server /data
    volumes: ["s3data:/data"]
volumes:
  pgdata:
  s3data:
```

### Required Environment Variables

```
DATABASE_URL=postgres://user:pass@postgres:5432/screenspace
S3_ENDPOINT=http://minio:9000
S3_BUCKET=screenspace
S3_ACCESS_KEY=...
S3_SECRET_KEY=...
JWT_SECRET=...
ADMIN_EMAIL=admin@example.com
```

### Documentation

- `docs/self-hosting.md` with step-by-step setup guide
- Covers: Docker, bare metal, Hetzner, common S3 providers (MinIO, Hetzner Object Storage, Cloudflare R2, AWS S3)
