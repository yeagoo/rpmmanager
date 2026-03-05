-- Products: plugin-style product definitions
CREATE TABLE products (
    id                  INTEGER PRIMARY KEY AUTOINCREMENT,
    name                TEXT NOT NULL UNIQUE,
    display_name        TEXT NOT NULL,
    description         TEXT DEFAULT '',

    -- Upstream source
    source_type         TEXT NOT NULL DEFAULT 'github',
    source_github_owner TEXT DEFAULT '',
    source_github_repo  TEXT DEFAULT '',
    source_url_template TEXT DEFAULT '',

    -- nfpm packaging template (JSON)
    nfpm_config         TEXT NOT NULL DEFAULT '{}',

    -- Target distros and architectures (JSON arrays)
    target_distros      TEXT NOT NULL DEFAULT '[]',
    architectures       TEXT NOT NULL DEFAULT '["x86_64","aarch64"]',

    -- Product line overrides (JSON, NULL = use defaults)
    product_lines       TEXT DEFAULT NULL,

    -- Packaging metadata
    maintainer          TEXT DEFAULT '',
    vendor              TEXT DEFAULT '',
    homepage            TEXT DEFAULT '',
    license             TEXT DEFAULT 'Apache-2.0',

    -- Scripts
    script_postinstall  TEXT DEFAULT '',
    script_preremove    TEXT DEFAULT '',

    -- Systemd service file content
    systemd_service     TEXT DEFAULT '',

    -- Default config file
    default_config      TEXT DEFAULT '',
    default_config_path TEXT DEFAULT '',

    -- Additional files (JSON array)
    extra_files         TEXT DEFAULT '[]',

    -- GPG key association
    gpg_key_id          INTEGER REFERENCES gpg_keys(id) ON DELETE SET NULL,

    -- Base URL override
    base_url            TEXT DEFAULT '',

    -- SM2 support
    sm2_enabled         BOOLEAN DEFAULT FALSE,

    enabled             BOOLEAN DEFAULT TRUE,
    created_at          DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at          DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Builds: build execution history
CREATE TABLE builds (
    id                  INTEGER PRIMARY KEY AUTOINCREMENT,
    product_id          INTEGER NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    version             TEXT NOT NULL,
    status              TEXT NOT NULL DEFAULT 'pending',
    current_stage       TEXT DEFAULT '',
    trigger_type        TEXT NOT NULL DEFAULT 'manual',

    -- Target selection snapshot
    target_distros      TEXT NOT NULL DEFAULT '[]',
    architectures       TEXT NOT NULL DEFAULT '[]',

    -- Results
    rpm_count           INTEGER DEFAULT 0,
    symlink_count       INTEGER DEFAULT 0,
    error_message       TEXT DEFAULT '',
    log_file            TEXT DEFAULT '',

    -- Timing
    started_at          DATETIME,
    finished_at         DATETIME,
    duration_seconds    INTEGER DEFAULT 0,

    created_at          DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- GPG keys
CREATE TABLE gpg_keys (
    id                  INTEGER PRIMARY KEY AUTOINCREMENT,
    name                TEXT NOT NULL,
    fingerprint         TEXT NOT NULL UNIQUE,
    key_id              TEXT NOT NULL,
    uid_name            TEXT DEFAULT '',
    uid_email           TEXT DEFAULT '',
    algorithm           TEXT DEFAULT '',
    key_length          INTEGER DEFAULT 0,
    created_date        DATETIME,
    expires_date        DATETIME,
    has_private         BOOLEAN DEFAULT FALSE,
    public_key_armor    TEXT DEFAULT '',
    is_default          BOOLEAN DEFAULT FALSE,
    created_at          DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at          DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Version monitors
CREATE TABLE monitors (
    id                  INTEGER PRIMARY KEY AUTOINCREMENT,
    product_id          INTEGER NOT NULL UNIQUE REFERENCES products(id) ON DELETE CASCADE,
    enabled             BOOLEAN DEFAULT TRUE,
    check_interval      TEXT DEFAULT '6h',
    auto_build          BOOLEAN DEFAULT FALSE,
    last_checked_at     DATETIME,
    last_known_version  TEXT DEFAULT '',
    last_error          TEXT DEFAULT '',
    created_at          DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at          DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Settings (key-value)
CREATE TABLE settings (
    key                 TEXT PRIMARY KEY,
    value               TEXT NOT NULL,
    updated_at          DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Indexes
CREATE INDEX idx_builds_product_id ON builds(product_id);
CREATE INDEX idx_builds_status ON builds(status);
CREATE INDEX idx_builds_created_at ON builds(created_at DESC);
CREATE INDEX idx_monitors_product_id ON monitors(product_id);
