package database

import (
	"context"
	"errors"

	"github.com/Aduneer/FlyTrap/internal/models"
	"github.com/jackc/pgx/v5"
)

func (s *Store) ListIssues(ctx context.Context, projectID int64, query models.IssueListQuery) (models.IssuePage, error) {
	var cursorTime any
	var cursorID int64
	if query.Cursor != nil {
		cursorTime = query.Cursor.LastSeen
		cursorID = query.Cursor.ID
	}

	rows, err := s.pool.Query(ctx, `
		SELECT id, project_id, fingerprint, exception_type, message, stacktrace,
			status, event_count, first_seen, last_seen, resolved_at, created_at
		FROM issues
		WHERE project_id = $1
		  AND ($2 = '' OR status = $2)
		  AND ($3::timestamptz IS NULL OR (last_seen, id) < ($3, $4))
		ORDER BY last_seen DESC, id DESC
		LIMIT $5
	`, projectID, string(query.Status), cursorTime, cursorID, query.Limit+1)
	if err != nil {
		return models.IssuePage{}, err
	}
	defer rows.Close()

	issues := make([]models.Issue, 0, query.Limit+1)
	for rows.Next() {
		var issue models.Issue
		if err := scanIssue(rows, &issue); err != nil {
			return models.IssuePage{}, err
		}
		issues = append(issues, issue)
	}

	if err := rows.Err(); err != nil {
		return models.IssuePage{}, err
	}

	page := models.IssuePage{Issues: issues}
	if len(page.Issues) > query.Limit {
		page.Issues = page.Issues[:query.Limit]
		page.HasMore = true
	}

	return page, nil
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

func (s *Store) UpdateIssueStatus(
	ctx context.Context,
	projectID, issueID int64,
	status models.IssueStatus,
) (models.Issue, error) {
	var issue models.Issue
	err := scanIssue(s.pool.QueryRow(ctx, `
		UPDATE issues
		SET status = $3,
			resolved_at = CASE
				WHEN $3 = 'resolved' THEN COALESCE(resolved_at, NOW())
				ELSE NULL
			END
		WHERE project_id = $1 AND id = $2
		RETURNING id, project_id, fingerprint, exception_type, message, stacktrace,
			status, event_count, first_seen, last_seen, resolved_at, created_at
	`, projectID, issueID, string(status)), &issue)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.Issue{}, models.ErrIssueNotFound
	}
	if err != nil {
		return models.Issue{}, err
	}

	return issue, nil
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
