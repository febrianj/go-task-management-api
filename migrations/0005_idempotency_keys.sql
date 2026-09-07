CREATE TABLE IF NOT EXISTS idempotency_keys (
    id            BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    idem_key      UUID NOT NULL,
    endpoint      TEXT NOT NULL,
    request_hash  TEXT NOT NULL,
    state         TEXT NOT NULL DEFAULT 'in_progress'
                    CHECK (state IN ('in_progress','completed')),
    response_code INT,
    response_body JSONB,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at    TIMESTAMPTZ NOT NULL DEFAULT now() + INTERVAL '24 hours',
    CONSTRAINT idem_unique UNIQUE (user_id, endpoint, idem_key)
);

CREATE INDEX IF NOT EXISTS idem_expires_idx ON idempotency_keys (expires_at);