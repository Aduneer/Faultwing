package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Aduneer/FlyTrap/internal/models"
)

type fakeEventStore struct {
	events []models.Event
}

func (s *fakeEventStore) CreateEvent(_ context.Context, input models.CreateEventRequest) (models.Event, error) {
	event := models.Event{
		ID:         int64(len(s.events) + 1),
		Message:    input.Message,
		Stacktrace: input.Stacktrace,
		CreatedAt:  time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC),
	}
	s.events = append(s.events, event)

	return event, nil
}

func (s *fakeEventStore) ListEvents(context.Context) ([]models.Event, error) {
	return s.events, nil
}

func TestEventHandlerCreateAndList(t *testing.T) {
	store := &fakeEventStore{}
	handler := NewEventHandler(store)

	createRequest := httptest.NewRequest(http.MethodPost, "/events", strings.NewReader(`{
		"message": "database connection timed out",
		"stacktrace": "db/client.go:42"
	}`))
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

	listRequest := httptest.NewRequest(http.MethodGet, "/events", nil)
	listResponse := httptest.NewRecorder()

	handler.ServeHTTP(listResponse, listRequest)

	if listResponse.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, listResponse.Code)
	}

	var events []models.Event
	if err := json.NewDecoder(listResponse.Body).Decode(&events); err != nil {
		t.Fatalf("decode list response: %v", err)
	}
	if len(events) != 1 || events[0].ID != created.ID {
		t.Fatalf("expected one returned event, got %#v", events)
	}
}
