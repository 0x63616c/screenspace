-- name: CreateWallpaper :one
INSERT INTO wallpapers (title, uploader_id, storage_key, resolution, width, height, duration, file_size, format, thumbnail_key, preview_key)
VALUES ($1, $2, $3, '', 0, 0, 0, 0, '', '', '')
RETURNING id, title, uploader_id, status, COALESCE(category, '') AS category, tags,
          resolution, width, height, duration, file_size, format, download_count,
          storage_key, thumbnail_key, preview_key, rejection_reason, created_at, updated_at;

-- name: GetWallpaperByID :one
SELECT id, title, uploader_id, status, COALESCE(category, '') AS category, tags,
       resolution, width, height, duration, file_size, format, download_count,
       storage_key, thumbnail_key, preview_key, rejection_reason, created_at, updated_at
FROM wallpapers
WHERE id = $1;

-- name: ListWallpapersRecent :many
SELECT id, title, uploader_id, status, COALESCE(category, '') AS category, tags,
       resolution, width, height, duration, file_size, format, download_count,
       storage_key, thumbnail_key, preview_key, rejection_reason, created_at, updated_at
FROM wallpapers
WHERE status = sqlc.arg('status')
  AND (sqlc.narg('category')::text IS NULL OR category ILIKE sqlc.narg('category'))
  AND (sqlc.narg('query')::text IS NULL OR title ILIKE sqlc.narg('query'))
ORDER BY created_at DESC
LIMIT sqlc.arg('lim') OFFSET sqlc.arg('off');

-- name: ListWallpapersPopular :many
SELECT id, title, uploader_id, status, COALESCE(category, '') AS category, tags,
       resolution, width, height, duration, file_size, format, download_count,
       storage_key, thumbnail_key, preview_key, rejection_reason, created_at, updated_at
FROM wallpapers
WHERE status = sqlc.arg('status')
  AND (sqlc.narg('category')::text IS NULL OR category ILIKE sqlc.narg('category'))
  AND (sqlc.narg('query')::text IS NULL OR title ILIKE sqlc.narg('query'))
ORDER BY download_count DESC, created_at DESC
LIMIT sqlc.arg('lim') OFFSET sqlc.arg('off');

-- name: CountWallpapers :one
SELECT COUNT(*)
FROM wallpapers
WHERE status = sqlc.arg('status')
  AND (sqlc.narg('category')::text IS NULL OR category ILIKE sqlc.narg('category'))
  AND (sqlc.narg('query')::text IS NULL OR title ILIKE sqlc.narg('query'));

-- name: UpdateWallpaperStatus :exec
UPDATE wallpapers
SET status = $1, updated_at = now()
WHERE id = $2;

-- name: UpdateWallpaperStatusWithReason :exec
UPDATE wallpapers
SET status = $1, rejection_reason = $2, updated_at = now()
WHERE id = $3;

-- name: UpdateWallpaperMetadata :exec
UPDATE wallpapers
SET title = $1, category = $2, tags = $3, updated_at = now()
WHERE id = $4;

-- name: UpdateWallpaperAfterFinalize :one
UPDATE wallpapers
SET width = $1,
    height = $2,
    duration = $3,
    file_size = $4,
    format = $5,
    resolution = $6,
    thumbnail_key = $7,
    preview_key = $8,
    status = $9,
    updated_at = now()
WHERE id = $10
RETURNING id, title, uploader_id, status, COALESCE(category, '') AS category, tags,
          resolution, width, height, duration, file_size, format, download_count,
          storage_key, thumbnail_key, preview_key, rejection_reason, created_at, updated_at;

-- name: IncrementDownloadCount :exec
UPDATE wallpapers
SET download_count = download_count + 1
WHERE id = $1;

-- name: DeleteWallpaper :exec
DELETE FROM wallpapers WHERE id = $1;
