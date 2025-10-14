-- name: CreateApplication :one
INSERT INTO application (id, name, description, category, path, icon_path, dir_updated_at)
VALUES (@id, @name, @description, @category, @path, @icon_path, @dir_updated_at)
RETURNING *;

-- name: GetApplications :many
SELECT *
FROM application;

-- name: GetExpiredApplications :many
SELECT *
FROM application
WHERE updated_at < @updated_at;

-- name: DeleteApplication :exec
DELETE
FROM application
WHERE id IN (sqlc.slice('ids'));

-- name: UpdateApplicationPartial :exec
UPDATE application
SET name           = COALESCE(sqlc.narg(name), name),
    description    = COALESCE(sqlc.narg(description), description),
    category       = COALESCE(sqlc.narg(category), category),
    path           = COALESCE(sqlc.narg(path), path),
    icon_path      = COALESCE(sqlc.narg(icon_path), icon_path),
    dir_updated_at = COALESCE(sqlc.narg(dir_updated_at), dir_updated_at),
    updated_at     = datetime('now', 'localtime')
WHERE id = @id;

-- name: GetApplicationIsUpdatedDir :one
SELECT *
FROM application
WHERE dir_updated_at != @dir_updated_at
  AND path = @path
LIMIT 1;