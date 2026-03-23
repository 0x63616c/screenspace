# Integration & Deployment Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Wire the macOS app to the Go server, set up deployment pipelines, and create self-hosting documentation.

**Architecture:** macOS app connects to the Go server via REST API. Server deployed via Docker. App distributed via GitHub Releases + Homebrew cask. CI/CD via GitHub Actions.

**Tech Stack:** GitHub Actions, Docker, Homebrew, Sparkle appcast, `gh` CLI, `xcrun notarytool`.

**Spec:** `docs/superpowers/specs/2026-03-23-screenspace-design.md` (Sections 8, 9, 12)

**Depends on:** Go server plan and macOS app plan being complete.

---

## File Structure

```
.github/
  workflows/
    server.yml                    # Build, test, push Docker image
    app.yml                       # Build, sign, notarize, release macOS app
    appcast.yml                   # Generate Sparkle appcast XML on release
docker-compose.yml                # Production-ready compose (root of repo)
docs/
  self-hosting.md                 # Step-by-step self-hosting guide
CONTENT_POLICY.md                 # Community content policy
PRIVACY_POLICY.md                 # Privacy policy
```

---

### Task 1: End-to-End Smoke Test

**Files:**
- Create: `tests/e2e/smoke_test.sh`

- [ ] **Step 1: Write smoke test script**

```bash
#!/bin/bash
# tests/e2e/smoke_test.sh
# Requires: docker compose, curl, jq
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

echo "=== Starting services ==="
cd "$REPO_ROOT/server"
docker compose up -d --build
sleep 5

BASE="http://localhost:8080"

echo "=== Health check ==="
STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$BASE/health")
[ "$STATUS" = "200" ] || { echo "FAIL: health check returned $STATUS"; exit 1; }

echo "=== Register admin ==="
ADMIN=$(curl -s -X POST "$BASE/api/v1/auth/register" \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@example.com","password":"testpassword123"}')
TOKEN=$(echo "$ADMIN" | jq -r '.token')
[ "$TOKEN" != "null" ] || { echo "FAIL: no token"; exit 1; }
ROLE=$(echo "$ADMIN" | jq -r '.role')
[ "$ROLE" = "admin" ] || { echo "FAIL: expected admin role, got $ROLE"; exit 1; }

echo "=== Register user ==="
USER=$(curl -s -X POST "$BASE/api/v1/auth/register" \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"testpassword123"}')
USER_TOKEN=$(echo "$USER" | jq -r '.token')

echo "=== Browse wallpapers (empty) ==="
LIST=$(curl -s "$BASE/api/v1/wallpapers")
COUNT=$(echo "$LIST" | jq -r '.total // 0')
[ "$COUNT" = "0" ] || { echo "FAIL: expected 0 wallpapers"; exit 1; }

echo "=== Login ==="
LOGIN=$(curl -s -X POST "$BASE/api/v1/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@example.com","password":"testpassword123"}')
LOGIN_TOKEN=$(echo "$LOGIN" | jq -r '.token')
[ "$LOGIN_TOKEN" != "null" ] || { echo "FAIL: login failed"; exit 1; }

echo "=== Get me ==="
ME=$(curl -s "$BASE/api/v1/auth/me" -H "Authorization: Bearer $TOKEN")
EMAIL=$(echo "$ME" | jq -r '.email')
[ "$EMAIL" = "admin@example.com" ] || { echo "FAIL: wrong email $EMAIL"; exit 1; }

echo "=== All smoke tests passed ==="
docker compose down
```

- [ ] **Step 2: Run smoke test**

Run: `bash tests/e2e/smoke_test.sh`
Expected: All checks pass.

- [ ] **Step 3: Commit**

```bash
git add tests/
git commit -m "test: add end-to-end smoke test for server API"
git push
```

---

### Task 2: Server CI/CD (GitHub Actions)

**Files:**
- Create: `.github/workflows/server.yml`

- [ ] **Step 1: Create server workflow**

```yaml
# .github/workflows/server.yml
name: Server CI

on:
  push:
    paths: ["server/**"]
  pull_request:
    paths: ["server/**"]

jobs:
  test:
    runs-on: ubuntu-latest
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
          go-version: "1.22"
      - name: Run tests with coverage
        working-directory: server
        env:
          DATABASE_URL: postgres://test:test@localhost:5432/screenspace_test?sslmode=disable
          S3_ENDPOINT: http://localhost:9000
          S3_ACCESS_KEY: minioadmin
          S3_SECRET_KEY: minioadmin
          JWT_SECRET: test-secret
        run: |
          go test -coverprofile=coverage.out ./... -v -race
          COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
          echo "Coverage: $COVERAGE%"
          if (( $(echo "$COVERAGE < 95" | bc -l) )); then
            echo "FAIL: coverage $COVERAGE% is below 95% threshold"
            exit 1
          fi

  docker:
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

- [ ] **Step 2: Push and verify CI runs**

Expected: GitHub Actions runs tests with real Postgres + MinIO. On main, pushes Docker image to GHCR.

- [ ] **Step 3: Commit**

```bash
git add .github/
git commit -m "ci: add server test and Docker image build workflow"
git push
```

---

### Task 3: macOS App CI/CD (GitHub Actions)

**Files:**
- Create: `.github/workflows/app.yml`

- [ ] **Step 1: Create app build workflow**

```yaml
# .github/workflows/app.yml
name: macOS App CI

on:
  push:
    paths: ["app/**"]
    tags: ["v*"]
  pull_request:
    paths: ["app/**"]

jobs:
  build:
    runs-on: macos-15
    steps:
      - uses: actions/checkout@v4
      - name: Build
        working-directory: app
        run: |
          xcodebuild -project ScreenSpace.xcodeproj \
            -scheme ScreenSpace \
            -configuration Release \
            -derivedDataPath build \
            CODE_SIGN_IDENTITY="-" \
            build
      - name: Run tests
        working-directory: app
        run: |
          xcodebuild -project ScreenSpace.xcodeproj \
            -scheme ScreenSpace \
            -configuration Debug \
            -derivedDataPath build \
            test

  release:
    needs: build
    if: startsWith(github.ref, 'refs/tags/v')
    runs-on: macos-15
    steps:
      - uses: actions/checkout@v4
      - name: Build for release
        working-directory: app
        run: |
          xcodebuild -project ScreenSpace.xcodeproj \
            -scheme ScreenSpace \
            -configuration Release \
            -derivedDataPath build \
            -archivePath build/ScreenSpace.xcarchive \
            archive
      - name: Export archive
        working-directory: app
        run: |
          xcodebuild -exportArchive \
            -archivePath build/ScreenSpace.xcarchive \
            -exportPath build/export \
            -exportOptionsPlist ExportOptions.plist
      - name: Create DMG
        run: |
          hdiutil create -volname "ScreenSpace" \
            -srcfolder "app/build/export/ScreenSpace.app" \
            -ov -format UDZO \
            "ScreenSpace.dmg"
      - name: Notarize
        env:
          APPLE_ID: ${{ secrets.APPLE_ID }}
          APPLE_PASSWORD: ${{ secrets.APPLE_APP_PASSWORD }}
          TEAM_ID: ${{ secrets.APPLE_TEAM_ID }}
        run: |
          xcrun notarytool submit ScreenSpace.dmg \
            --apple-id "$APPLE_ID" \
            --password "$APPLE_PASSWORD" \
            --team-id "$TEAM_ID" \
            --wait
          xcrun stapler staple ScreenSpace.dmg
      - name: Upload release asset
        uses: softprops/action-gh-release@v1
        with:
          files: ScreenSpace.dmg
```

- [ ] **Step 2: Commit**

```bash
git add .github/
git commit -m "ci: add macOS app build, test, and release workflow"
git push
```

---

### Task 4: Sparkle Appcast Generation

**Files:**
- Create: `.github/workflows/appcast.yml`

- [ ] **Step 1: Create appcast workflow**

```yaml
name: Sparkle Appcast
on:
  release:
    types: [published]
jobs:
  appcast:
    runs-on: macos-15
    steps:
      - uses: actions/checkout@v4
      - name: Download release DMG
        run: gh release download ${{ github.event.release.tag_name }} -p "ScreenSpace.dmg"
        env:
          GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
      - name: Generate appcast
        env:
          SPARKLE_KEY: ${{ secrets.SPARKLE_EDDSA_KEY }}
        run: |
          brew install sparkle
          echo "$SPARKLE_KEY" > sparkle_key
          generate_appcast --ed-key-file sparkle_key --download-url-prefix "https://github.com/0x63616c/screenspace/releases/download/${{ github.event.release.tag_name }}/" .
          rm sparkle_key
      - name: Upload appcast
        run: gh release upload ${{ github.event.release.tag_name }} appcast.xml --clobber
        env:
          GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

- [ ] **Step 2: Commit**

```bash
git add .github/
git commit -m "ci: add Sparkle appcast generation on release"
git push
```

---

### Task 5: Production Docker Compose & Self-Hosting Docs

**Files:**
- Create: `docker-compose.yml` (repo root)
- Create: `docs/self-hosting.md`

- [ ] **Step 1: Create production docker-compose.yml**

```yaml
# docker-compose.yml (repo root - for self-hosters)
services:
  server:
    image: ghcr.io/0x63616c/screenspace-server:latest
    ports: ["8080:8080"]
    environment:
      DATABASE_URL: postgres://screenspace:${POSTGRES_PASSWORD}@postgres:5432/screenspace?sslmode=disable
      S3_ENDPOINT: http://minio:9000
      S3_BUCKET: screenspace
      S3_ACCESS_KEY: ${MINIO_ROOT_USER}
      S3_SECRET_KEY: ${MINIO_ROOT_PASSWORD}
      JWT_SECRET: ${JWT_SECRET}
      ADMIN_EMAIL: ${ADMIN_EMAIL}
    depends_on:
      postgres:
        condition: service_healthy
      minio-init:
        condition: service_completed_successfully
    restart: unless-stopped

  postgres:
    image: postgres:16
    environment:
      POSTGRES_DB: screenspace
      POSTGRES_USER: screenspace
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD}
    volumes: ["pgdata:/var/lib/postgresql/data"]
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U screenspace"]
      interval: 5s
      timeout: 5s
      retries: 5
    restart: unless-stopped

  minio:
    image: minio/minio
    command: server /data --console-address ":9001"
    environment:
      MINIO_ROOT_USER: ${MINIO_ROOT_USER}
      MINIO_ROOT_PASSWORD: ${MINIO_ROOT_PASSWORD}
    ports: ["9001:9001"]
    volumes: ["s3data:/data"]
    restart: unless-stopped

  minio-init:
    image: minio/mc
    depends_on:
      - minio
    entrypoint: >
      /bin/sh -c "
      sleep 3;
      mc alias set local http://minio:9000 $${MINIO_ROOT_USER} $${MINIO_ROOT_PASSWORD};
      mc mb --ignore-existing local/screenspace;
      "
    environment:
      MINIO_ROOT_USER: ${MINIO_ROOT_USER}
      MINIO_ROOT_PASSWORD: ${MINIO_ROOT_PASSWORD}

volumes:
  pgdata:
  s3data:
```

- [ ] **Step 2: Create .env.example at repo root**

```
POSTGRES_PASSWORD=change-me-in-production
MINIO_ROOT_USER=screenspace
MINIO_ROOT_PASSWORD=change-me-in-production
JWT_SECRET=change-me-use-openssl-rand-hex-32
ADMIN_EMAIL=you@example.com
```

- [ ] **Step 3: Write self-hosting docs**

```markdown
# Self-Hosting ScreenSpace

## Quick Start (Docker)

1. Clone the repo: `git clone https://github.com/0x63616c/screenspace.git`
2. Copy env file: `cp .env.example .env`
3. Edit `.env` with your values (especially passwords and JWT secret)
4. Start: `docker compose up -d`
5. Server is running at `http://localhost:8080`
6. Register with the email in ADMIN_EMAIL to get admin access

## Configure the macOS App

1. Open ScreenSpace > Settings > Gallery
2. Change Server URL to your server address (e.g. `https://wallpaper.yourdomain.com`)
3. Register an account

## Storage Providers

The server uses S3-compatible storage. Swap MinIO for any provider:

### Hetzner Object Storage
- S3_ENDPOINT=https://fsn1.your-objectstorage.com
- Create a bucket named "screenspace"

### Cloudflare R2
- S3_ENDPOINT=https://<account-id>.r2.cloudflarestorage.com
- Free egress

### AWS S3
- S3_ENDPOINT=https://s3.amazonaws.com
- Set S3_BUCKET to your bucket name

## Reverse Proxy (Caddy example)

screenspace.yourdomain.com {
    reverse_proxy localhost:8080
}

## Bare Metal (no Docker)

1. Install Postgres 16 and create a database
2. Install ffmpeg
3. Download the server binary from GitHub Releases
4. Set environment variables
5. Run: `./server`
```

- [ ] **Step 4: Commit**

```bash
git add docker-compose.yml .env.example docs/self-hosting.md
git commit -m "docs: add production docker-compose and self-hosting guide"
git push
```

---

### Task 6: Content Policy & Privacy Policy

**Files:**
- Create: `CONTENT_POLICY.md`
- Create: `PRIVACY_POLICY.md`

- [ ] **Step 1: Write content policy**

Covers: no NSFW, no copyrighted content without license, no hateful/violent content, uploader responsibility, DMCA takedown process, repeat infringer policy.

- [ ] **Step 2: Write privacy policy**

Covers: what data is collected (email, hashed password, upload history, IP for rate limiting), no tracking/analytics/third-party sharing, data deletion on request, GDPR compliance.

- [ ] **Step 3: Commit**

```bash
git add CONTENT_POLICY.md PRIVACY_POLICY.md
git commit -m "docs: add content policy and privacy policy"
git push
```

---

### Task 7: Homebrew Cask

**Files:**
- Create: `HomebrewFormula/screenspace.rb` (or submit to homebrew-cask)

- [ ] **Step 1: Create Homebrew cask definition**

```ruby
cask "screenspace" do
  version "0.1.0"
  sha256 :no_check  # Updated on each release

  url "https://github.com/0x63616c/screenspace/releases/download/v#{version}/ScreenSpace.dmg"
  name "ScreenSpace"
  desc "Open-source live wallpaper app for macOS"
  homepage "https://github.com/0x63616c/screenspace"

  app "ScreenSpace.app"

  zap trash: [
    "~/Library/Application Support/ScreenSpace",
    "~/Library/Screen Savers/ScreenSpaceSaver.saver",
  ]
end
```

- [ ] **Step 2: Test locally**

Run: `brew install --cask ./HomebrewFormula/screenspace.rb`

- [ ] **Step 3: Commit**

```bash
git add HomebrewFormula/
git commit -m "feat: add Homebrew cask definition"
git push
```
