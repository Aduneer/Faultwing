CREATE TABLE IF NOT EXISTS events (
    id BIGSERIAL PRIMARY KEY,
    issue_id BIGINT REFERENCES issues(id),
    message TEXT NOT NULL,
    stacktrace TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
