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
	"github.com/Aduneer/FlyTrap/internal/userauth"
)

type fakeProjectStore struct {
	project  models.Project
	key      string
	err      error
	name     string
	ownerID  int64
	projects []models.Project
}

func (s *fakeProjectStore) CreateProject(_ context.Context, ownerID int64, name string) (models.Project, string, error) {
	s.ownerID = ownerID
	s.name = name
	return s.project, s.key, s.err
}

func (s *fakeProjectStore) ListProjects(_ context.Context, ownerID int64) ([]models.Project, error) {
	s.ownerID = ownerID
	return s.projects, s.err
}

type projectUserAuthenticator struct {
	user models.User
}

func (a projectUserAuthenticator) AuthenticateSession(context.Context, [32]byte) (models.User, models.Session, error) {
	return a.user, models.Session{ID: 1, UserID: a.user.ID}, nil
}

func authenticatedProjectHandler(t *testing.T, store *fakeProjectStore) (http.Handler, string) {
	t.Helper()
	token, _, err := userauth.GenerateSessionToken()
	if err != nil {
		t.Fatalf("generate session token: %v", err)
	}
	authenticator := projectUserAuthenticator{user: models.User{ID: 42, Email: "learner@example.com"}}
	return middleware.RequireUserSession(authenticator, NewProjectHandler(store)), token
}

func TestProjectHandlerCreatesAndListsOwnedProjects(t *testing.T) {
	store := &fakeProjectStore{
		project: models.Project{
			ID:        1,
			OwnerID:   42,
			Name:      "My App",
			CreatedAt: time.Date(2026, 9, 19, 12, 0, 0, 0, time.UTC),
		},
		key: "fly_test-key",
	}
	store.projects = []models.Project{store.project}
	handler, token := authenticatedProjectHandler(t, store)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/projects", strings.NewReader(`{"name":"  My App  "}`))
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, response.Code)
	}
	if store.ownerID != 42 || store.name != "My App" {
		t.Fatalf("unexpected project creation: owner=%d name=%q", store.ownerID, store.name)
	}

	var result createProjectResponse
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if result.Project.ID != 1 || result.APIKey != "fly_test-key" {
		t.Fatalf("unexpected response: %#v", result)
	}

	listRequest := httptest.NewRequest(http.MethodGet, "/api/v1/projects", nil)
	listRequest.Header.Set("Authorization", "Bearer "+token)
	listResponse := httptest.NewRecorder()
	handler.ServeHTTP(listResponse, listRequest)
	if listResponse.Code != http.StatusOK {
		t.Fatalf("list: expected status %d, got %d", http.StatusOK, listResponse.Code)
	}
	var projects []models.Project
	if err := json.NewDecoder(listResponse.Body).Decode(&projects); err != nil {
		t.Fatalf("decode project list: %v", err)
	}
	if len(projects) != 1 || projects[0].OwnerID != 42 {
		t.Fatalf("unexpected project list: %#v", projects)
	}
}

func TestProjectHandlerRejectsInvalidName(t *testing.T) {
	store := &fakeProjectStore{}
	handler, token := authenticatedProjectHandler(t, store)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/projects", strings.NewReader(`{"name":"  "}`))
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, response.Code)
	}
}

func TestProjectHandlerRejectsDuplicateName(t *testing.T) {
	store := &fakeProjectStore{err: models.ErrProjectNameTaken}
	handler, token := authenticatedProjectHandler(t, store)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/projects", strings.NewReader(`{"name":"My App"}`))
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusConflict {
		t.Fatalf("expected status %d, got %d", http.StatusConflict, response.Code)
	}
}
