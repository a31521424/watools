-- name: CreateApplication :one
INSERT INTO applications (id, name, description, category, path, icon_path, dir_updated_at)
VALUES (@id, @name, @description, @category, @path, @icon_path, @dir_updated_at)
RETURNING *;

-- name: GetApplications :many
SELECT *
FROM applications;

-- name: GetExpiredApplications :many
SELECT *
FROM applications
WHERE updated_at < @updated_at;

-- name: DeleteApplication :exec
DELETE
FROM applications
WHERE id = @id;

-- name: UpdateApplicationPartial :exec
UPDATE applications
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
FROM applications
WHERE dir_updated_at != @dir_updated_at
  AND path = @path
LIMIT 1;