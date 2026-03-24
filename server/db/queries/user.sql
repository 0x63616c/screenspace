-- name: CreateUser :one
INSERT INTO users (email, password_hash, role)
VALUES ($1, $2, $3)
RETURNING id, email, password_hash, role, banned, created_at;

-- name: GetUserByID :one
SELECT id, email, password_hash, role, banned, created_at
FROM users
WHERE id = $1;

-- name: GetUserByEmail :one
SELECT id, email, password_hash, role, banned, created_at
FROM users
WHERE email = $1;

-- name: ListUsers :many
SELECT id, email, password_hash, role, banned, created_at
FROM users
ORDER BY created_at DESC
LIMIT sqlc.arg('lim') OFFSET sqlc.arg('off');

-- name: CountUsers :one
SELECT COUNT(*) FROM users;

-- name: ListUsersWithSearch :many
SELECT id, email, password_hash, role, banned, created_at
FROM users
WHERE email ILIKE sqlc.arg('query')
ORDER BY created_at DESC
LIMIT sqlc.arg('lim') OFFSET sqlc.arg('off');

-- name: CountUsersWithSearch :one
SELECT COUNT(*) FROM users
WHERE email ILIKE sqlc.arg('query');

-- name: SetBanned :exec
UPDATE users SET banned = $1 WHERE id = $2;

-- name: SetRole :exec
UPDATE users SET role = $1 WHERE id = $2;
