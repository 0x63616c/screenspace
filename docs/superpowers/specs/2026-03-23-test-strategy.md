# ScreenSpace Test Strategy

How to verify every change autonomously without manual screenshots.

---

## Available Tools

| Tool | Path | What it does |
|------|------|-------------|
| `swift build` | SPM project at `app/` | Compile check. Catches type errors, missing imports, syntax. |
| `swift test` | `app/Tests/ScreenSpaceTests/` | Unit tests. 5 existing test files. |
| `screencapture -x /tmp/ss.png` | macOS built-in | Silent screenshot of entire screen. I can read the image with the Read tool. |
| `screencapture -l<windowID> /tmp/ss.png` | macOS built-in | Capture specific window by CGWindowID. |
| `osascript` | macOS built-in | AppleScript for UI interaction. Click buttons, read element trees, navigate tabs. |
| `System Events` accessibility API | via osascript | Query UI element hierarchy, check if buttons exist, read labels, verify structure. |
| `go test ./...` | `server/` | Go tests. 16 existing test files covering all handlers/repos. |
| `curl` | system | Hit server endpoints directly for API testing. |
| `docker compose` | `server/docker-compose.yml` | Spin up full server stack (Go + Postgres + MinIO) for integration tests. |

---

## Test Layers

### Layer 1: Compile Gate (every change)

```bash
cd app && swift build 2>&1
```

If this fails, nothing else matters. Run after every file edit. Swift 6 strict concurrency means the compiler catches a lot of thread safety issues too.

### Layer 2: Unit Tests (every logic change)

```bash
cd app && swift test 2>&1
```

**Existing tests:**
- `PauseControllerTests` -- battery, lock, config update (6 tests)
- `APIClientTests` -- URL building, JSON decoding (4 tests)
- `ConfigTests` -- config persistence
- `DisplayIdentifierTests` -- stable ID generation
- `VideoImporterTests` -- file validation

**Tests to add for this plan:**

| Test File | What it covers |
|-----------|---------------|
| `AppStateTests.swift` | AppState init, setWallpaper updates currentWallpaperURL, login/logout state changes, isAdmin computed property |
| `PlaylistManagerTests.swift` | Create/delete/update playlists, interval advancement, shuffle logic, edge case: empty playlist |
| `CacheManagerTests.swift` | Cache eviction when over limit, cachedURL returns nil for missing files, cacheFile copies correctly |
| `PauseControllerTests.swift` (expand) | Add fullscreen occlusion test, multiple pause reasons clearing independently |

**How I run them:**
```bash
cd /Users/calum/code/github.com/0x63616c/screenspace/app && swift test --filter AppStateTests
```

### Layer 3: UI Structure Verification (accessibility tree)

This is the key insight. I don't need screenshots to verify UI structure. I can query the accessibility tree of the running app via `osascript`:

**Launch the app:**
```bash
cd /Users/calum/code/github.com/0x63616c/screenspace/app && swift build && .build/arm64-apple-macosx/debug/ScreenSpace &
sleep 2
```

**Query window exists:**
```applescript
tell application "System Events"
    tell process "ScreenSpace"
        get every window
        get name of every window
    end tell
end tell
```

**Verify sidebar items:**
```applescript
tell application "System Events"
    tell process "ScreenSpace"
        tell window 1
            -- Check sidebar has expected items
            get entire contents of group 1
        end tell
    end tell
end tell
```

**Verify a specific button exists:**
```applescript
tell application "System Events"
    tell process "ScreenSpace"
        tell window 1
            get every button whose name contains "Set as Wallpaper"
        end tell
    end tell
end tell
```

**Verify admin tab is hidden for non-admin:**
```applescript
tell application "System Events"
    tell process "ScreenSpace"
        tell window 1
            -- Should NOT find "Admin" in sidebar when not logged in as admin
            get static texts whose value is "Admin"
        end tell
    end tell
end tell
```

**Click through tabs and verify content:**
```applescript
tell application "System Events"
    tell process "ScreenSpace"
        tell window 1
            -- Click Explore in sidebar
            click static text "Explore" of group 1
            delay 0.5
            -- Verify Explore view loaded (should have search field after fix)
            get every text field
        end tell
    end tell
end tell
```

### Layer 4: Visual Screenshot Verification (layout changes)

For layout/visual issues (V1-V16), I need to actually see the pixels.

**Full app screenshot:**
```bash
screencapture -x -o /tmp/screenspace-home.png
```

**Specific window screenshot:**
```bash
# Get window ID first
osascript -e 'tell application "System Events" to get id of window 1 of process "ScreenSpace"'
# Capture just that window
screencapture -x -o -l<windowID> /tmp/screenspace-window.png
```

Then read the screenshot:
```
Read tool: /tmp/screenspace-window.png
```

I can see the image and verify:
- Settings content is top-aligned (V1)
- No double border (V2)
- Display IDs are hidden (V3)
- Upload sheet has link text (V9)
- Hero section doesn't clip (V14)

**Workflow for visual verification:**
1. Build and launch app
2. Use osascript to navigate to the view I'm checking
3. Capture screenshot
4. Read screenshot with Read tool
5. Verify visually
6. Kill app process

### Layer 5: Server Tests (every backend change)

```bash
cd /Users/calum/code/github.com/0x63616c/screenspace/server && go test ./... 2>&1
```

**Existing coverage:** Auth, wallpaper CRUD, admin actions, favorites, reports, rate limiting, migrations, storage, config.

**Tests to add:**

| Test File | What it covers |
|-----------|---------------|
| `service/auth_test.go` (expand) | Malformed JWT doesn't panic (S1) |
| `handler/wallpaper_test.go` (expand) | Title max length, tags count limit (S8, S9) |
| `handler/report_test.go` (expand) | Reason max length (S7) |
| `handler/admin_test.go` (expand) | Ban/promote non-existent user returns 404 (S3), rejection reason stored (S4) |
| `repository/wallpaper_test.go` (expand) | Case-insensitive category filter (S5) |
| `middleware/ratelimit_test.go` (expand) | Stale entry cleanup (S10) |

**Integration test with curl:**
```bash
# Start server
cd /Users/calum/code/github.com/0x63616c/screenspace/server
docker compose up -d
sleep 3

# Register
curl -s -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"test@test.com","password":"test1234"}'

# Verify category validation
curl -s -X POST http://localhost:8080/api/v1/wallpapers \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"title":"Test","category":"INVALID_CATEGORY","tags":[]}'
# Should return 400

docker compose down
```

### Layer 6: End-to-End Smoke Test

After all changes are complete, full smoke test:

1. `docker compose up -d` (start server)
2. `swift build && open .build/.../ScreenSpace` (launch app)
3. osascript: verify Home loads (accessibility tree has "Popular", "Recently Added")
4. osascript: click "Explore" -- verify categories grid exists
5. osascript: click "Library" -- verify drop zone exists
6. osascript: open Settings sheet, verify all tabs render
7. osascript: verify "Admin" not in sidebar (not logged in)
8. screencapture: take final screenshot, verify layout
9. Kill app, tear down docker

---

## Test Strategy Per Issue Category

| Category | Primary Test | Fallback |
|----------|-------------|----------|
| Disconnected UI (W1-W14) | Unit test: AppState method called. Accessibility: button exists and is wired. | Screenshot |
| Missing features (M1-M14) | Unit test: logic works. Accessibility: UI element exists. | Screenshot |
| Visual/UX (V1-V16) | Screenshot + Read tool verification | osascript element check |
| Language/formatting (L1-L10) | osascript: read text values of UI elements | Screenshot |
| Code quality (C1-C21) | Compile gate + unit tests | N/A (code-level) |
| Server (S1-S12) | `go test` + curl integration tests | N/A |

---

## Key Principles

1. **Build before anything.** `swift build` / `go build` after every change.
2. **Unit test all non-UI logic.** AppState, PauseController wiring, playlist advancement, cache eviction.
3. **Accessibility tree for UI structure.** Verify buttons exist, labels are correct, elements are present/absent.
4. **Screenshots for visual verification.** Layout, spacing, borders, clipping.
5. **Go tests for server.** Every validation fix gets a test.
6. **Don't test what the compiler tests.** Swift 6 strict concurrency catches threading issues at compile time.
