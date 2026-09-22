package database

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Aduneer/Faultwing/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

const issueUpdatesChannel = "faultwing_issue_updates"

type IssueUpdateListener struct {
	connection *pgxpool.Conn
}

func (s *Store) OpenIssueUpdateListener(ctx context.Context) (*IssueUpdateListener, error) {
	connection, err := s.pool.Acquire(ctx)
	if err != nil {
		return nil, err
	}

	if _, err := connection.Exec(ctx, "LISTEN "+issueUpdatesChannel); err != nil {
		connection.Release()
		return nil, err
	}

	return &IssueUpdateListener{connection: connection}, nil
}

func (l *IssueUpdateListener) Wait(ctx context.Context) (models.IssueUpdate, error) {
	notification, err := l.connection.Conn().WaitForNotification(ctx)
	if err != nil {
		return models.IssueUpdate{}, err
	}

	var update models.IssueUpdate
	if err := json.Unmarshal([]byte(notification.Payload), &update); err != nil {
		return models.IssueUpdate{}, fmt.Errorf("decode issue update: %w", err)
	}
	if notification.Channel != issueUpdatesChannel || update.Type != models.IssueUpdateType || update.ProjectID < 1 || update.IssueID < 1 {
		return models.IssueUpdate{}, fmt.Errorf("invalid issue update payload")
	}

	return update, nil
}

func (l *IssueUpdateListener) Close() error {
	if l == nil || l.connection == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err := l.connection.Exec(ctx, "UNLISTEN "+issueUpdatesChannel)
	l.connection.Release()
	l.connection = nil
	return err
}
