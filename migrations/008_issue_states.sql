ALTER TABLE issues
    ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'open',
    ADD COLUMN IF NOT EXISTS resolved_at TIMESTAMPTZ,
    ADD CONSTRAINT issues_status_check
        CHECK (status IN ('open', 'resolved', 'ignored'));

CREATE INDEX IF NOT EXISTS issues_project_status_idx
    ON issues(project_id, status, last_seen DESC);
