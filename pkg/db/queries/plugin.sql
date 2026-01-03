-- name: GetPlugins :many
SELECT *
FROM plugin_state;

-- name: InsertPlugin :exec
INSERT INTO plugin_state (package_id, enabled, storage)
VALUES (?, ?, ?);

-- name: DeletePlugin :exec
DELETE FROM plugin_state
WHERE package_id = ?;

-- name: UpdatePluginUsage :exec
UPDATE plugin_state
SET last_used_at = ?, used_count = ?
WHERE package_id = ?;

-- name: UpdatePluginEnabled :exec
UPDATE plugin_state
SET enabled = ?
WHERE package_id = ?;

-- name: UpdatePluginStorage :exec
UPDATE plugin_state
SET storage = ?
WHERE package_id = ?;