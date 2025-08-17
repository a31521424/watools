CREATE TABLE IF NOT EXISTS commands
(
    id             VARCHAR(36)  NOT NULL PRIMARY KEY,
    name           VARCHAR(255) NOT NULL,
    description    VARCHAR(255) NOT NULL,
    category       VARCHAR(255) NOT NULL,
    path           TEXT         NOT NULL UNIQUE,
    icon_path      TEXT         NOT NULL,
    updated_at     TEXT         NOT NULL DEFAULT (datetime('now', 'localtime')),
    dir_updated_at TEXT         NOT NULL
);

CREATE TABLE metadata
(
    key   TEXT PRIMARY KEY,
    value TEXT
);