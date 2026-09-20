ALTER TABLE issues
    ADD COLUMN IF NOT EXISTS project_id BIGINT REFERENCES projects(id);

DO $$
DECLARE
    legacy_project_id BIGINT;
BEGIN
    IF EXISTS (SELECT 1 FROM issues WHERE project_id IS NULL) THEN
        INSERT INTO projects (name)
        VALUES ('Legacy')
        ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name
        RETURNING id INTO legacy_project_id;

        UPDATE issues
        SET project_id = legacy_project_id
        WHERE project_id IS NULL;
    END IF;
END $$;

ALTER TABLE issues
    ALTER COLUMN project_id SET NOT NULL;

ALTER TABLE issues
    DROP CONSTRAINT IF EXISTS issues_fingerprint_key;

CREATE UNIQUE INDEX IF NOT EXISTS issues_project_fingerprint_idx
    ON issues(project_id, fingerprint);
