-- name: CreatePlugin :one
INSERT INTO plugins (id, package_id, name, version, description)
VALUES (@id, @package_id, @name, @version, @description)
RETURNING *;

-- name: GetPlugins :many
SELECT *
FROM plugins;

-- name: GetPlugin :one
SELECT *
FROM plugins
WHERE id = @id;

-- name: DeletePlugin :exec
DELETE
FROM plugins
WHERE id = @id;