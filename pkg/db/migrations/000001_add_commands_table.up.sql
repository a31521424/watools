CREATE TABLE IF NOT EXISTS application
(
    id             TEXT     NOT NULL PRIMARY KEY,
    name           TEXT     NOT NULL,
    description    TEXT,
    category       TEXT     NOT NULL,
    path           TEXT     NOT NULL UNIQUE,
    icon_path      TEXT,
    updated_at     DATETIME NOT NULL DEFAULT (datetime('now', 'localtime')),
    dir_updated_at DATETIME NOT NULL,
    last_used_at   DATETIME,
    used_count     INT      NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS plugin_state
(
    package_id   TEXT    NOT NULL PRIMARY KEY,
    enabled      BOOLEAN NOT NULL,
    storage      TEXT    NOT NULL,
    last_used_at DATETIME,
    used_count   INT     NOT NULL DEFAULT 0
);

CREATE TABLE metadata
(
    key   TEXT PRIMARY KEY,
    value TEXT
);