package database

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

func Open(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, err
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}

	if err := migrate(ctx, pool); err != nil {
		pool.Close()
		return nil, err
	}

	return pool, nil
}

func migrate(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS issues (
			id BIGSERIAL PRIMARY KEY,
			fingerprint TEXT NOT NULL UNIQUE,
			message TEXT NOT NULL,
			stacktrace TEXT NOT NULL DEFAULT '',
			event_count BIGINT NOT NULL DEFAULT 0,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);

		CREATE TABLE IF NOT EXISTS events (
			id BIGSERIAL PRIMARY KEY,
			issue_id BIGINT REFERENCES issues(id),
			message TEXT NOT NULL,
			stacktrace TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);

		ALTER TABLE events ADD COLUMN IF NOT EXISTS issue_id BIGINT REFERENCES issues(id);
	`)
	return err
}
