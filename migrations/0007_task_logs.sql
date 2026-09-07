CREATE TABLE IF NOT EXISTS task_logs (
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    task_id    UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    actor_id   UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    action     TEXT NOT NULL,
    from_value JSONB,
    to_value   JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS task_logs_task_idx ON task_logs (task_id, created_at DESC);