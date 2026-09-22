package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Aduneer/Faultwing/internal/middleware"
	"github.com/Aduneer/Faultwing/internal/models"
)

type fakeEventStore struct {
	jobs      []models.EventJob
	projectID int64
}

type eventAuthenticator struct {
	project models.Project
}

func (a eventAuthenticator) AuthenticateProject(context.Context, string) (models.Project, error) {
	return a.project, nil
}

func (s *fakeEventStore) EnqueueEvent(_ context.Context, projectID int64, input models.CreateEventRequest) (models.EventJob, error) {
	s.projectID = projectID
	job := models.EventJob{
		ID:        int64(len(s.jobs) + 1),
		ProjectID: projectID,
		Payload:   input,
		CreatedAt: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC),
	}
	s.jobs = append(s.jobs, job)

	return job, nil
}

func TestEventHandlerQueuesEventForAuthenticatedProject(t *testing.T) {
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
	createRequest.Header.Set("Authorization", "Bearer faultwing_test-key")
	createResponse := httptest.NewRecorder()

	handler.ServeHTTP(createResponse, createRequest)

	if createResponse.Code != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d", http.StatusAccepted, createResponse.Code)
	}

	var queued enqueueEventResponse
	if err := json.NewDecoder(createResponse.Body).Decode(&queued); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if queued.JobID != 1 || queued.QueuedAt.IsZero() {
		t.Fatalf("expected a queue receipt, got %#v", queued)
	}
	if len(store.jobs) != 1 || store.jobs[0].Payload.Environment != "production" || store.jobs[0].Payload.Release != "1.3.2" {
		t.Fatalf("expected monitoring context in queued job, got %#v", store.jobs)
	}
	if store.projectID != 42 {
		t.Fatalf("expected project 42, got %d", store.projectID)
	}
}
