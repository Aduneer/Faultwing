package database

import (
	"context"
	"crypto/sha256"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/Aduneer/FlyTrap/internal/models"
)

func TestUserSessionFlow(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	store := newIntegrationStore(t, ctx, databaseURL)
	user, err := store.CreateUser(ctx, "learner@example.com", "test-password-hash")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	loaded, err := store.FindUserByEmail(ctx, "LEARNER@example.com")
	if err != nil {
		t.Fatalf("find user: %v", err)
	}
	if loaded.ID != user.ID || loaded.PasswordHash != "test-password-hash" {
		t.Fatalf("unexpected loaded user: %#v", loaded)
	}
	if _, err := store.CreateUser(ctx, "LEARNER@example.com", "another-hash"); !errors.Is(err, models.ErrEmailTaken) {
		t.Fatalf("expected duplicate email error, got %v", err)
	}

	tokenHash := sha256.Sum256([]byte("test-session-token"))
	session, err := store.CreateSession(ctx, user.ID, tokenHash, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	authenticated, authenticatedSession, err := store.AuthenticateSession(ctx, tokenHash)
	if err != nil {
		t.Fatalf("authenticate session: %v", err)
	}
	if authenticated.ID != user.ID || authenticatedSession.ID != session.ID {
		t.Fatalf("session resolved incorrectly: user=%#v session=%#v", authenticated, authenticatedSession)
	}

	if err := store.RevokeSession(ctx, tokenHash); err != nil {
		t.Fatalf("revoke session: %v", err)
	}
	if _, _, err := store.AuthenticateSession(ctx, tokenHash); !errors.Is(err, models.ErrInvalidSession) {
		t.Fatalf("expected revoked session to be rejected, got %v", err)
	}

	project, _, err := store.CreateProject(ctx, user.ID, "My App")
	if err != nil {
		t.Fatalf("create owned project: %v", err)
	}
	otherUser, err := store.CreateUser(ctx, "other@example.com", "test-password-hash")
	if err != nil {
		t.Fatalf("create other user: %v", err)
	}
	otherProject, _, err := store.CreateProject(ctx, otherUser.ID, "my app")
	if err != nil {
		t.Fatalf("reuse project name for another owner: %v", err)
	}
	projects, err := store.ListProjects(ctx, user.ID)
	if err != nil {
		t.Fatalf("list owned projects: %v", err)
	}
	if len(projects) != 1 || projects[0].ID != project.ID || projects[0].OwnerID != user.ID || otherProject.OwnerID != otherUser.ID {
		t.Fatalf("projects were not scoped to their owners: projects=%#v other=%#v", projects, otherProject)
	}
}
