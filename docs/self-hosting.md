# Self-Hosting ScreenSpace

## Quick Start (Docker)

1. Clone the repo:
   ```bash
   git clone https://github.com/0x63616c/screenspace.git
   cd screenspace
   ```

2. Copy env file and edit:
   ```bash
   cp .env.example .env
   # Edit .env with your values (especially passwords and JWT secret)
   # Generate a JWT secret: openssl rand -hex 32
   ```

3. Start:
   ```bash
   docker compose up -d
   ```

4. Server is running at `http://localhost:8080`

5. Register with the email in ADMIN_EMAIL to get admin access.

## Configure the macOS App

1. Open ScreenSpace > Settings > General
2. Change Server URL to your server address (e.g. `https://wallpaper.yourdomain.com`)
3. Register an account

## Storage Providers

The server uses S3-compatible storage. Swap MinIO for any provider by changing the env vars:

### Hetzner Object Storage
```
S3_ENDPOINT=https://fsn1.your-objectstorage.com
S3_ACCESS_KEY=your-access-key
S3_SECRET_KEY=your-secret-key
S3_BUCKET=screenspace
```

### Cloudflare R2
```
S3_ENDPOINT=https://<account-id>.r2.cloudflarestorage.com
S3_ACCESS_KEY=your-access-key
S3_SECRET_KEY=your-secret-key
S3_BUCKET=screenspace
```

### AWS S3
```
S3_ENDPOINT=https://s3.amazonaws.com
S3_ACCESS_KEY=your-access-key
S3_SECRET_KEY=your-secret-key
S3_BUCKET=screenspace
```

## Reverse Proxy (Caddy)

```
screenspace.yourdomain.com {
    reverse_proxy localhost:8080
}
```

## Bare Metal (no Docker)

1. Install Postgres 16 and create a database
2. Install ffmpeg
3. Download the server binary from GitHub Releases
4. Set environment variables (see .env.example)
5. Run: `./server`
