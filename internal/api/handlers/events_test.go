package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Aduneer/FlyTrap/internal/middleware"
	"github.com/Aduneer/FlyTrap/internal/models"
)

type fakeEventStore struct {
	events    []models.Event
	projectID int64
}

type eventAuthenticator struct {
	project models.Project
}

func (a eventAuthenticator) AuthenticateProject(context.Context, string) (models.Project, error) {
	return a.project, nil
}

func (s *fakeEventStore) CreateEvent(_ context.Context, projectID int64, input models.CreateEventRequest) (models.Event, error) {
	s.projectID = projectID
	event := models.Event{
		ID:            int64(len(s.events) + 1),
		ExceptionType: input.ExceptionType,
		Message:       input.Message,
		Stacktrace:    input.Stacktrace,
		Environment:   input.Environment,
		Release:       input.Release,
		CreatedAt:     time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC),
	}
	s.events = append(s.events, event)

	return event, nil
}

func TestEventHandlerCreatesEventForAuthenticatedProject(t *testing.T) {
	store := &fakeEventStore{}
	authenticator := eventAuthenticator{
		project: models.Project{ID: 42, Name: "My App"},
	}
	handler := middleware.RequireAPIKey(authenticator, NewEventHandler(store))

	createRequest := httptest.NewRequest(http.MethodPost, "/api/v1/events", strings.NewReader(`{
		"exception_type": "DatabaseTimeoutError",
		"message": "database connection timed out",
		"stacktrace": "db/client.go:42",
		"environment": "production",
		"release": "1.3.2"
	}`))
	createRequest.Header.Set("Authorization", "Bearer fly_test-key")
	createResponse := httptest.NewRecorder()

	handler.ServeHTTP(createResponse, createRequest)

	if createResponse.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, createResponse.Code)
	}

	var created models.Event
	if err := json.NewDecoder(createResponse.Body).Decode(&created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if created.Message != "database connection timed out" {
		t.Fatalf("expected saved message, got %q", created.Message)
	}
	if created.Environment != "production" || created.Release != "1.3.2" {
		t.Fatalf("expected monitoring context in response, got %#v", created)
	}
	if store.projectID != 42 {
		t.Fatalf("expected project 42, got %d", store.projectID)
	}
}
