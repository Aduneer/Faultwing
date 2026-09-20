package database

import (
	"context"

	"github.com/Aduneer/FlyTrap/internal/fingerprint"
	"github.com/Aduneer/FlyTrap/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) CreateEvent(ctx context.Context, projectID int64, input models.CreateEventRequest) (models.Event, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return models.Event{}, err
	}
	defer tx.Rollback(ctx)

	issueFingerprint := fingerprint.Event(input.ExceptionType, input.Stacktrace)
	var issueID int64
	err = tx.QueryRow(ctx, `
		INSERT INTO issues (project_id, fingerprint, exception_type, message, stacktrace, event_count)
		VALUES ($1, $2, $3, $4, $5, 1)
		ON CONFLICT (project_id, fingerprint) DO UPDATE
		SET event_count = issues.event_count + 1,
			last_seen = NOW(),
			exception_type = EXCLUDED.exception_type,
			message = EXCLUDED.message,
			stacktrace = EXCLUDED.stacktrace
		RETURNING id
	`, projectID, issueFingerprint, input.ExceptionType, input.Message, input.Stacktrace).Scan(&issueID)
	if err != nil {
		return models.Event{}, err
	}

	var event models.Event

	err = tx.QueryRow(ctx, `
		INSERT INTO events (issue_id, exception_type, message, stacktrace, environment, release)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, issue_id, exception_type, message, stacktrace, environment, release, created_at
	`, issueID, input.ExceptionType, input.Message, input.Stacktrace, input.Environment, input.Release).Scan(
		&event.ID,
		&event.IssueID,
		&event.ExceptionType,
		&event.Message,
		&event.Stacktrace,
		&event.Environment,
		&event.Release,
		&event.CreatedAt,
	)
	if err != nil {
		return models.Event{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return models.Event{}, err
	}

	return event, nil
}
