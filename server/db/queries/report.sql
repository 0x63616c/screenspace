-- name: CreateReport :one
INSERT INTO reports (wallpaper_id, reporter_id, reason)
VALUES ($1, $2, $3)
RETURNING id, wallpaper_id, reporter_id, reason, status, created_at;

-- name: ListPendingReports :many
SELECT id, wallpaper_id, reporter_id, reason, status, created_at
FROM reports
WHERE status = 'pending'
ORDER BY created_at DESC
LIMIT sqlc.arg('lim') OFFSET sqlc.arg('off');

-- name: CountPendingReports :one
SELECT COUNT(*) FROM reports WHERE status = 'pending';

-- name: DismissReport :exec
UPDATE reports SET status = 'dismissed' WHERE id = $1;
