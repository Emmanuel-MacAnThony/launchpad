CREATE TABLE services (
    id               TEXT PRIMARY KEY,
    name             TEXT NOT NULL,
    repo_url         TEXT NOT NULL,
    domain           TEXT NOT NULL UNIQUE,
    health_check_url TEXT NOT NULL,
    webhook_secret   TEXT NOT NULL,
    host             TEXT NOT NULL,
    ssh_user         TEXT NOT NULL,
    ssh_key_path     TEXT NOT NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
