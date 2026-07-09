CREATE TABLE deploys (
    id             TEXT        PRIMARY KEY,
    service_id     TEXT        NOT NULL REFERENCES services(id),
    slot           TEXT        DEFAULT NULL
                               CONSTRAINT deploys_slot_check
                               CHECK (slot IN ('blue', 'green')),
    status         TEXT        NOT NULL DEFAULT 'pending'
                               CONSTRAINT deploys_status_check
                               CHECK (status IN ('pending', 'building', 'active', 'failed', 'rolled_back')),
    commit_sha     TEXT        NOT NULL,
    commit_message TEXT        NOT NULL DEFAULT '',
    pushed_at      TIMESTAMPTZ NOT NULL,
    started_at     TIMESTAMPTZ DEFAULT NULL,
    finished_at    TIMESTAMPTZ DEFAULT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX deploys_service_id_idx ON deploys (service_id);
CREATE INDEX deploys_status_idx     ON deploys (service_id, status);

CREATE TABLE deploy_locks (
    deploy_id    TEXT        PRIMARY KEY REFERENCES deploys(id),
    locked_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at   TIMESTAMPTZ NOT NULL,
    released_at  TIMESTAMPTZ DEFAULT NULL
);
