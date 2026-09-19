package database

import (
	"context"

	"github.com/Aduneer/FlyTrap/internal/fingerprint"
	"github.com/Aduneer/FlyTrap/internal/models"
)

func (s *Store) CreateIssue(ctx context.Context, input models.CreateIssueRequest) (models.Issue, error) {
	var issue models.Issue
	err := s.pool.QueryRow(ctx, `
		INSERT INTO issues (fingerprint, message, stacktrace)
		VALUES ($1, $2, $3)
		RETURNING id, fingerprint, message, stacktrace, event_count, created_at
	`, fingerprint.Event(input.Message, input.Stacktrace), input.Message, input.Stacktrace).Scan(
		&issue.ID,
		&issue.Fingerprint,
		&issue.Message,
		&issue.Stacktrace,
		&issue.EventCount,
		&issue.CreatedAt,
	)
	return issue, err
}

func (s *Store) ListIssues(ctx context.Context) ([]models.Issue, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, fingerprint, message, stacktrace, event_count, created_at
		FROM issues
		ORDER BY event_count DESC, created_at DESC, id DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	issues := make([]models.Issue, 0)
	for rows.Next() {
		var issue models.Issue
		if err := rows.Scan(
			&issue.ID,
			&issue.Fingerprint,
			&issue.Message,
			&issue.Stacktrace,
			&issue.EventCount,
			&issue.CreatedAt,
		); err != nil {
			return nil, err
		}
		issues = append(issues, issue)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return issues, nil
}
