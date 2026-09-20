package database

import (
	"context"
	"errors"

	"github.com/Aduneer/FlyTrap/internal/models"
	"github.com/jackc/pgx/v5"
)

func (s *Store) ListIssues(ctx context.Context, projectID int64, status models.IssueStatus) ([]models.Issue, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, project_id, fingerprint, exception_type, message, stacktrace,
			status, event_count, first_seen, last_seen, resolved_at, created_at
		FROM issues
		WHERE project_id = $1
		  AND ($2 = '' OR status = $2)
		ORDER BY last_seen DESC, id DESC
	`, projectID, string(status))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	issues := make([]models.Issue, 0)
	for rows.Next() {
		var issue models.Issue
		if err := scanIssue(rows, &issue); err != nil {
			return nil, err
		}
		issues = append(issues, issue)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return issues, nil
}

func (s *Store) GetIssue(ctx context.Context, projectID, issueID int64) (models.IssueDetail, error) {
	var detail models.IssueDetail
	err := s.pool.QueryRow(ctx, `
		SELECT id, project_id, fingerprint, exception_type, message, stacktrace,
			status, event_count, first_seen, last_seen, resolved_at, created_at,
			ARRAY(
				SELECT DISTINCT environment
				FROM events
				WHERE issue_id = issues.id
				ORDER BY environment
			),
			ARRAY(
				SELECT DISTINCT release
				FROM events
				WHERE issue_id = issues.id AND release <> ''
				ORDER BY release
			)
		FROM issues
		WHERE project_id = $1 AND id = $2
	`, projectID, issueID).Scan(
		&detail.ID,
		&detail.ProjectID,
		&detail.Fingerprint,
		&detail.ExceptionType,
		&detail.Message,
		&detail.Stacktrace,
		&detail.Status,
		&detail.EventCount,
		&detail.FirstSeen,
		&detail.LastSeen,
		&detail.ResolvedAt,
		&detail.CreatedAt,
		&detail.Environments,
		&detail.Releases,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.IssueDetail{}, models.ErrIssueNotFound
	}
	if err != nil {
		return models.IssueDetail{}, err
	}

	return detail, nil
}

type issueScanner interface {
	Scan(...any) error
}

func scanIssue(row issueScanner, issue *models.Issue) error {
	return row.Scan(
		&issue.ID,
		&issue.ProjectID,
		&issue.Fingerprint,
		&issue.ExceptionType,
		&issue.Message,
		&issue.Stacktrace,
		&issue.Status,
		&issue.EventCount,
		&issue.FirstSeen,
		&issue.LastSeen,
		&issue.ResolvedAt,
		&issue.CreatedAt,
	)
}
