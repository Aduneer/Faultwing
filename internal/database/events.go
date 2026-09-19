package database

import (
	"context"

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
	var event models.Event

	err := s.pool.QueryRow(ctx, `
		INSERT INTO events (message, stacktrace)
		VALUES ($1, $2)
		RETURNING id, message, stacktrace, created_at
	`, input.Message, input.Stacktrace).Scan(
		&event.ID,
		&event.Message,
		&event.Stacktrace,
		&event.CreatedAt,
	)
	if err != nil {
		return models.Event{}, err
	}

	return event, nil
}

func (s *Store) ListEvents(ctx context.Context) ([]models.Event, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, message, stacktrace, created_at
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
		if err := rows.Scan(&event.ID, &event.Message, &event.Stacktrace, &event.CreatedAt); err != nil {
			return nil, err
		}
		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return events, nil
}
