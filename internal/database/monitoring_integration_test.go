package database

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/Aduneer/FlyTrap/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestMonitoringFlow(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	store := newIntegrationStore(t, ctx, databaseURL)
	owner, err := store.CreateUser(ctx, "monitoring@example.com", "test-password-hash")
	if err != nil {
		t.Fatalf("create project owner: %v", err)
	}

	projectA, keyA, err := store.CreateProject(ctx, owner.ID, "Project A")
	if err != nil {
		t.Fatalf("create project A: %v", err)
	}
	projectB, keyB, err := store.CreateProject(ctx, owner.ID, "Project B")
	if err != nil {
		t.Fatalf("create project B: %v", err)
	}

	authenticatedA, err := store.AuthenticateProject(ctx, keyA)
	if err != nil {
		t.Fatalf("authenticate project A: %v", err)
	}
	authenticatedB, err := store.AuthenticateProject(ctx, keyB)
	if err != nil {
		t.Fatalf("authenticate project B: %v", err)
	}
	if authenticatedA.ID != projectA.ID || authenticatedB.ID != projectB.ID {
		t.Fatal("API keys resolved to the wrong projects")
	}

	event := models.CreateEventRequest{
		ExceptionType: "DatabaseTimeoutError",
		Message:       "database connection timed out",
		Stacktrace:    "db/client.go:42",
		Environment:   "production",
		Release:       "1.3.2",
	}

	first, err := store.CreateEvent(ctx, authenticatedA.ID, event)
	if err != nil {
		t.Fatalf("create first project A event: %v", err)
	}
	second, err := store.CreateEvent(ctx, authenticatedA.ID, event)
	if err != nil {
		t.Fatalf("create second project A event: %v", err)
	}
	otherProject, err := store.CreateEvent(ctx, authenticatedB.ID, event)
	if err != nil {
		t.Fatalf("create project B event: %v", err)
	}

	if first.IssueID != second.IssueID {
		t.Fatal("matching events in one project should share an issue")
	}
	if first.IssueID == otherProject.IssueID {
		t.Fatal("events from different projects should not share an issue")
	}

	issuesA, err := store.ListIssues(ctx, projectA.ID, models.IssueListQuery{Limit: 10})
	if err != nil {
		t.Fatalf("list project A issues: %v", err)
	}
	issuesB, err := store.ListIssues(ctx, projectB.ID, models.IssueListQuery{Limit: 10})
	if err != nil {
		t.Fatalf("list project B issues: %v", err)
	}
	if len(issuesA.Issues) != 1 || issuesA.Issues[0].EventCount != 2 {
		t.Fatalf("expected project A count 2, got %#v", issuesA)
	}
	if len(issuesB.Issues) != 1 || issuesB.Issues[0].EventCount != 1 {
		t.Fatalf("expected project B count 1, got %#v", issuesB)
	}

	resolved, err := store.UpdateIssueStatus(ctx, projectA.ID, first.IssueID, models.IssueStatusResolved)
	if err != nil {
		t.Fatalf("resolve project A issue: %v", err)
	}
	if resolved.Status != models.IssueStatusResolved || resolved.ResolvedAt == nil {
		t.Fatalf("expected resolved issue with timestamp, got %#v", resolved)
	}

	if _, err := store.CreateEvent(ctx, projectA.ID, event); err != nil {
		t.Fatalf("create regression event: %v", err)
	}
	detail, err := store.GetIssue(ctx, projectA.ID, first.IssueID)
	if err != nil {
		t.Fatalf("load reopened issue: %v", err)
	}
	if detail.Status != models.IssueStatusOpen || detail.ResolvedAt != nil || detail.EventCount != 3 {
		t.Fatalf("expected reopened issue with count 3, got %#v", detail.Issue)
	}
	if len(detail.Environments) != 1 || detail.Environments[0] != "production" ||
		len(detail.Releases) != 1 || detail.Releases[0] != "1.3.2" {
		t.Fatalf("unexpected monitoring context: %#v", detail)
	}

	anotherEvent := event
	anotherEvent.ExceptionType = "CacheMissError"
	anotherEvent.Stacktrace = "cache/client.go:17"
	if _, err := store.CreateEvent(ctx, projectA.ID, anotherEvent); err != nil {
		t.Fatalf("create second issue: %v", err)
	}

	firstPage, err := store.ListIssues(ctx, projectA.ID, models.IssueListQuery{Limit: 1})
	if err != nil {
		t.Fatalf("load first issue page: %v", err)
	}
	if len(firstPage.Issues) != 1 || !firstPage.HasMore {
		t.Fatalf("expected a full first page, got %#v", firstPage)
	}

	lastIssue := firstPage.Issues[0]
	secondPage, err := store.ListIssues(ctx, projectA.ID, models.IssueListQuery{
		Limit: 1,
		Cursor: &models.IssueCursor{
			LastSeen: lastIssue.LastSeen,
			ID:       lastIssue.ID,
		},
	})
	if err != nil {
		t.Fatalf("load second issue page: %v", err)
	}
	if len(secondPage.Issues) != 1 || secondPage.HasMore || secondPage.Issues[0].ID == lastIssue.ID {
		t.Fatalf("unexpected second page: %#v", secondPage)
	}

	job, err := store.EnqueueEvent(ctx, projectA.ID, event)
	if err != nil {
		t.Fatalf("enqueue event: %v", err)
	}
	if job.ID == 0 || job.ProjectID != projectA.ID || job.Payload.Message != event.Message || job.CreatedAt.IsZero() {
		t.Fatalf("unexpected queued event: %#v", job)
	}
	listener, err := store.OpenIssueUpdateListener(ctx)
	if err != nil {
		t.Fatalf("listen for issue updates: %v", err)
	}
	defer listener.Close()

	processed, err := store.ProcessNextEventJob(ctx)
	if err != nil {
		t.Fatalf("process queued event: %v", err)
	}
	if !processed {
		t.Fatal("expected a queued event to be processed")
	}
	processedIssue, err := store.GetIssue(ctx, projectA.ID, first.IssueID)
	if err != nil {
		t.Fatalf("load issue after queued event: %v", err)
	}
	if processedIssue.EventCount != 4 {
		t.Fatalf("expected queued event to increment issue count, got %d", processedIssue.EventCount)
	}
	notificationCtx, cancelNotification := context.WithTimeout(ctx, 2*time.Second)
	defer cancelNotification()
	update, err := listener.Wait(notificationCtx)
	if err != nil {
		t.Fatalf("wait for issue update: %v", err)
	}
	if update.Type != models.IssueUpdateType || update.ProjectID != projectA.ID || update.IssueID != first.IssueID {
		t.Fatalf("unexpected issue update: %#v", update)
	}

	processed, err = store.ProcessNextEventJob(ctx)
	if err != nil {
		t.Fatalf("check empty event queue: %v", err)
	}
	if processed {
		t.Fatal("expected event queue to be empty")
	}
}

func newIntegrationStore(t *testing.T, ctx context.Context, databaseURL string) *Store {
	t.Helper()

	adminPool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("open integration database: %v", err)
	}
	t.Cleanup(adminPool.Close)

	schema := fmt.Sprintf("flytrap_test_%d", time.Now().UnixNano())
	identifier := pgx.Identifier{schema}.Sanitize()
	if _, err := adminPool.Exec(ctx, "CREATE SCHEMA "+identifier); err != nil {
		t.Fatalf("create integration schema: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, err := adminPool.Exec(cleanupCtx, "DROP SCHEMA "+identifier+" CASCADE"); err != nil {
			t.Errorf("drop integration schema: %v", err)
		}
	})

	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		t.Fatalf("parse integration database URL: %v", err)
	}
	config.ConnConfig.RuntimeParams["search_path"] = schema

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatalf("open isolated integration schema: %v", err)
	}
	t.Cleanup(pool.Close)

	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("ping integration database: %v", err)
	}
	if err := migrate(ctx, pool); err != nil {
		t.Fatalf("migrate integration schema: %v", err)
	}

	return NewStore(pool)
}
