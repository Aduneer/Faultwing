CREATE TABLE IF NOT EXISTS event_jobs (
    id BIGSERIAL PRIMARY KEY,
    project_id BIGINT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    exception_type TEXT NOT NULL,
    message TEXT NOT NULL,
    stacktrace TEXT NOT NULL DEFAULT '',
    environment TEXT NOT NULL,
    release TEXT NOT NULL DEFAULT '',
    attempts INTEGER NOT NULL DEFAULT 0 CHECK (attempts >= 0),
    available_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS event_jobs_available_idx
    ON event_jobs(available_at, id);
