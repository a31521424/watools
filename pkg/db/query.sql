-- name: CreateCommand :one
INSERT INTO commands (name, description, category, path, icon_path)
VALUES (?, ?, ?, ?, ?)
RETURNING *;

-- name: GetCommands :many
SELECT *
FROM commands;

-- name: GetCommand :one
SELECT *
FROM commands
WHERE id = ?;

-- name: FindCommandByPath :one
SELECT *
FROM commands
WHERE path = ?;

-- name: FindExpiredCommands :many
SELECT *
FROM commands
WHERE updated_at < ?
  AND is_deleted = 0;

-- name: DeleteCommand :exec
DELETE
FROM commands
WHERE id = ?;

-- name: UpdateCommandPartial :exec
Update commands
SET name        = COALESCE(sqlc.narg(name), name),
    description = COALESCE(sqlc.narg(description), description),
    category    = COALESCE(sqlc.narg(category), category),
    path        = COALESCE(sqlc.narg(path), path),
    icon_path   = COALESCE(sqlc.narg(icon_path), icon_path),
    is_deleted  = COALESCE(sqlc.narg(is_deleted), is_deleted),
    updated_at  = datetime('now')
WHERE id = @id;