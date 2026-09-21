package database

import (
	"context"
	"fmt"
	"time"

	"github.com/Aduneer/FlyTrap/internal/models"
	"github.com/jackc/pgx/v5"
)

const maxEventJobRetryDelay = 5 * time.Minute

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

func (s *Store) ProcessNextEventJob(ctx context.Context) (bool, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)

	var job models.EventJob
	err = tx.QueryRow(ctx, `
		SELECT id, project_id, exception_type, message, stacktrace, environment,
			release, attempts, available_at, created_at
		FROM event_jobs
		WHERE available_at <= NOW()
		ORDER BY available_at, id
		FOR UPDATE SKIP LOCKED
		LIMIT 1
	`).Scan(
		&job.ID,
		&job.ProjectID,
		&job.Payload.ExceptionType,
		&job.Payload.Message,
		&job.Payload.Stacktrace,
		&job.Payload.Environment,
		&job.Payload.Release,
		&job.Attempts,
		&job.AvailableAt,
		&job.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return false, nil
		}
		return false, err
	}

	processingTx, err := tx.Begin(ctx)
	if err != nil {
		return true, err
	}

	event, processingErr := createEvent(ctx, processingTx, job.ProjectID, job.Payload)
	if processingErr == nil {
		_, processingErr = processingTx.Exec(ctx, `
			SELECT pg_notify(
				'flytrap_issue_updates',
				json_build_object(
					'type', 'issue.updated',
					'project_id', $1::bigint,
					'issue_id', $2::bigint
				)::text
			)
		`, job.ProjectID, event.IssueID)
	}
	if processingErr == nil {
		_, processingErr = processingTx.Exec(ctx, `DELETE FROM event_jobs WHERE id = $1`, job.ID)
	}
	if processingErr == nil {
		if err := processingTx.Commit(ctx); err != nil {
			return true, err
		}
		if err := tx.Commit(ctx); err != nil {
			return true, err
		}
		return true, nil
	}

	if err := processingTx.Rollback(ctx); err != nil {
		return true, fmt.Errorf("roll back event job %d: %w", job.ID, err)
	}
	if ctx.Err() != nil {
		return true, ctx.Err()
	}

	attempts := job.Attempts + 1
	retryAt := time.Now().Add(eventJobRetryDelay(attempts))
	_, err = tx.Exec(ctx, `
		UPDATE event_jobs
		SET attempts = $2, available_at = $3, last_error = $4
		WHERE id = $1
	`, job.ID, attempts, retryAt, processingErr.Error())
	if err != nil {
		return true, fmt.Errorf("reschedule event job %d: %w", job.ID, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return true, fmt.Errorf("commit event job %d retry: %w", job.ID, err)
	}

	return true, fmt.Errorf("process event job %d: %w", job.ID, processingErr)
}

func eventJobRetryDelay(attempt int) time.Duration {
	delay := time.Second * time.Duration(1<<min(attempt-1, 9))
	return min(delay, maxEventJobRetryDelay)
}
