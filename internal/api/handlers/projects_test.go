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

type fakeProjectStore struct {
	project models.Project
	key     string
	err     error
	name    string
}

func (s *fakeProjectStore) CreateProject(_ context.Context, name string) (models.Project, string, error) {
	s.name = name
	return s.project, s.key, s.err
}

func TestProjectHandlerCreate(t *testing.T) {
	store := &fakeProjectStore{
		project: models.Project{
			ID:        1,
			Name:      "My App",
			CreatedAt: time.Date(2026, 9, 19, 12, 0, 0, 0, time.UTC),
		},
		key: "fly_test-key",
	}
	handler := NewProjectHandler(store)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/projects", strings.NewReader(`{"name":"  My App  "}`))
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, response.Code)
	}
	if store.name != "My App" {
		t.Fatalf("expected trimmed name, got %q", store.name)
	}

	var result createProjectResponse
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if result.Project.ID != 1 || result.APIKey != "fly_test-key" {
		t.Fatalf("unexpected response: %#v", result)
	}
}

func TestProjectHandlerRejectsInvalidName(t *testing.T) {
	store := &fakeProjectStore{}
	handler := NewProjectHandler(store)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/projects", strings.NewReader(`{"name":"  "}`))
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, response.Code)
	}
}

func TestProjectHandlerRejectsDuplicateName(t *testing.T) {
	store := &fakeProjectStore{err: models.ErrProjectNameTaken}
	handler := NewProjectHandler(store)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/projects", strings.NewReader(`{"name":"My App"}`))
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusConflict {
		t.Fatalf("expected status %d, got %d", http.StatusConflict, response.Code)
	}
}
