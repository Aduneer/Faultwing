package database

import (
	"context"

	"github.com/Aduneer/FlyTrap/internal/models"
)

func (s *Store) EnqueueEvent(
	ctx context.Context,
	projectID int64,
	input models.CreateEventRequest,
) (models.EventJob, error) {
	job := models.EventJob{
		ProjectID: projectID,
		Payload:   input,
	}

	err := s.pool.QueryRow(ctx, `
		INSERT INTO event_jobs (
			project_id, exception_type, message, stacktrace, environment, release
		)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, attempts, available_at, created_at
	`,
		projectID,
		input.ExceptionType,
		input.Message,
		input.Stacktrace,
		input.Environment,
		input.Release,
	).Scan(
		&job.ID,
		&job.Attempts,
		&job.AvailableAt,
		&job.CreatedAt,
	)
	if err != nil {
		return models.EventJob{}, err
	}

	return job, nil
}
