ALTER TABLE projects
    ADD COLUMN owner_id BIGINT REFERENCES users(id);

DO $$
DECLARE
    legacy_user_id BIGINT;
BEGIN
    IF EXISTS (SELECT 1 FROM projects WHERE owner_id IS NULL) THEN
        INSERT INTO users (email, password_hash)
        VALUES ('legacy-projects@faultwing.invalid', '!')
        RETURNING id INTO legacy_user_id;

        UPDATE projects
        SET owner_id = legacy_user_id
        WHERE owner_id IS NULL;
    END IF;
END $$;

ALTER TABLE projects
    ALTER COLUMN owner_id SET NOT NULL,
    DROP CONSTRAINT IF EXISTS projects_name_key;

CREATE UNIQUE INDEX projects_owner_name_lower_idx
    ON projects(owner_id, LOWER(name));
