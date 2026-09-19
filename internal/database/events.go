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

func (s *Store) CreateEvent(ctx context.Context, input models.CreateEventRequest) (models.Event, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return models.Event{}, err
	}
	defer tx.Rollback(ctx)

	issueFingerprint := fingerprint.Event(input.Message, input.Stacktrace)
	var issue models.Issue
	err = tx.QueryRow(ctx, `
		INSERT INTO issues (fingerprint, message, stacktrace, event_count)
		VALUES ($1, $2, $3, 1)
		ON CONFLICT (fingerprint) DO UPDATE
		SET event_count = issues.event_count + 1
		RETURNING id, fingerprint, message, stacktrace, event_count, created_at
	`, issueFingerprint, input.Message, input.Stacktrace).Scan(
		&issue.ID,
		&issue.Fingerprint,
		&issue.Message,
		&issue.Stacktrace,
		&issue.EventCount,
		&issue.CreatedAt,
	)
	if err != nil {
		return models.Event{}, err
	}

	var event models.Event

	err = tx.QueryRow(ctx, `
		INSERT INTO events (issue_id, message, stacktrace)
		VALUES ($1, $2, $3)
		RETURNING id, issue_id, message, stacktrace, created_at
	`, issue.ID, input.Message, input.Stacktrace).Scan(
		&event.ID,
		&event.IssueID,
		&event.Message,
		&event.Stacktrace,
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

func (s *Store) ListEvents(ctx context.Context) ([]models.Event, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, COALESCE(issue_id, 0), message, stacktrace, created_at
		FROM events
		ORDER BY created_at DESC, id DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := make([]models.Event, 0)
	for rows.Next() {
		var event models.Event
		if err := rows.Scan(&event.ID, &event.IssueID, &event.Message, &event.Stacktrace, &event.CreatedAt); err != nil {
			return nil, err
		}
		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return events, nil
}
