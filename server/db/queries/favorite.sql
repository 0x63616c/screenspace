-- name: CheckFavorite :one
SELECT EXISTS(
    SELECT 1 FROM favorites
    WHERE user_id = $1 AND wallpaper_id = $2
) AS exists;

-- name: InsertFavorite :exec
INSERT INTO favorites (user_id, wallpaper_id)
VALUES ($1, $2);

-- name: DeleteFavorite :exec
DELETE FROM favorites
WHERE user_id = $1 AND wallpaper_id = $2;

-- name: ListFavoritesByUser :many
SELECT w.id, w.title, w.uploader_id, w.status, COALESCE(w.category, '') AS category,
       w.tags, w.resolution, w.width, w.height, w.duration, w.file_size, w.format,
       w.download_count, w.storage_key, w.thumbnail_key, w.preview_key,
       w.rejection_reason, w.created_at, w.updated_at
FROM favorites f
JOIN wallpapers w ON w.id = f.wallpaper_id
WHERE f.user_id = $1
ORDER BY f.created_at DESC
LIMIT sqlc.arg('lim') OFFSET sqlc.arg('off');

-- name: CountFavoritesByUser :one
SELECT COUNT(*) FROM favorites WHERE user_id = $1;
