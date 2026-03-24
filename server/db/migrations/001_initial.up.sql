CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT 'user',
    banned BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE wallpapers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    title TEXT NOT NULL,
    uploader_id UUID NOT NULL REFERENCES users(id),
    status TEXT NOT NULL DEFAULT 'pending',
    category TEXT,
    tags TEXT[] DEFAULT '{}',
    resolution TEXT NOT NULL,
    width INT NOT NULL,
    height INT NOT NULL,
    duration FLOAT NOT NULL,
    file_size BIGINT NOT NULL,
    format TEXT NOT NULL,
    download_count BIGINT NOT NULL DEFAULT 0,
    storage_key TEXT NOT NULL,
    thumbnail_key TEXT NOT NULL,
    preview_key TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE favorites (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    wallpaper_id UUID NOT NULL REFERENCES wallpapers(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, wallpaper_id)
);

CREATE TABLE reports (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    wallpaper_id UUID NOT NULL REFERENCES wallpapers(id) ON DELETE CASCADE,
    reporter_id UUID NOT NULL REFERENCES users(id),
    reason TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_wallpapers_status ON wallpapers(status);
CREATE INDEX idx_wallpapers_category ON wallpapers(category);
CREATE INDEX idx_wallpapers_download_count ON wallpapers(download_count DESC);
CREATE INDEX idx_wallpapers_created_at ON wallpapers(created_at DESC);
CREATE INDEX idx_reports_status ON reports(status);
