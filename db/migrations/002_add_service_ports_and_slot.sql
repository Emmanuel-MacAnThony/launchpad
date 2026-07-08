ALTER TABLE services
    ADD COLUMN blue_port      INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN green_port     INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN container_port INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN active_slot    TEXT    DEFAULT NULL
        CONSTRAINT services_active_slot_check CHECK (active_slot IN ('blue', 'green'));
