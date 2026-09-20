ALTER TABLE issues
    ADD COLUMN IF NOT EXISTS exception_type TEXT NOT NULL DEFAULT 'UnknownError',
    ADD COLUMN IF NOT EXISTS first_seen TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS last_seen TIMESTAMPTZ;

UPDATE issues
SET first_seen = COALESCE(first_seen, created_at),
    last_seen = COALESCE(last_seen, created_at)
WHERE first_seen IS NULL OR last_seen IS NULL;

ALTER TABLE issues
    ALTER COLUMN first_seen SET DEFAULT NOW(),
    ALTER COLUMN first_seen SET NOT NULL,
    ALTER COLUMN last_seen SET DEFAULT NOW(),
    ALTER COLUMN last_seen SET NOT NULL;

ALTER TABLE events
    ADD COLUMN IF NOT EXISTS exception_type TEXT NOT NULL DEFAULT 'UnknownError',
    ADD COLUMN IF NOT EXISTS environment TEXT NOT NULL DEFAULT 'unknown',
    ADD COLUMN IF NOT EXISTS release TEXT NOT NULL DEFAULT '';
