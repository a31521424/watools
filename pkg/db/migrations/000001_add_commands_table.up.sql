CREATE TABLE IF NOT EXISTS commands
(
    id          integer      NOT NULL PRIMARY KEY AUTOINCREMENT,
    name        VARCHAR(255) NOT NULL,
    description VARCHAR(255) NOT NULL,
    category    VARCHAR(255) NOT NULL,
    path        VARCHAR(255) NOT NULL UNIQUE,
    icon_path   VARCHAR(255) NOT NULL,
    created_at  TEXT         NOT NULL DEFAULT (datetime('now')),
    updated_at  TEXT         NOT NULL DEFAULT (datetime('now')),
    is_deleted  BOOLEAN      NOT NULL DEFAULT 0
);

CREATE TABLE metadata
(
    key   TEXT PRIMARY KEY,
    value TEXT
);