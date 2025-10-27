-- name: GetPlugins :many
SELECT *
FROM plugin_state;

-- name: UpdatePluginUsage :exec
UPDATE plugin_state
SET last_used_at = ?, used_count = ?
WHERE package_id = ?;