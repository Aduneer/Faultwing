package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Aduneer/FlyTrap/internal/middleware"
	"github.com/Aduneer/FlyTrap/internal/models"
)

type fakeIssueStore struct {
	issues    []models.Issue
	detail    models.IssueDetail
	err       error
	projectID int64
	issueID   int64
	status    models.IssueStatus
}

func (s *fakeIssueStore) ListIssues(_ context.Context, projectID int64, status models.IssueStatus) ([]models.Issue, error) {
	s.projectID = projectID
	s.status = status
	return s.issues, s.err
}

func (s *fakeIssueStore) GetIssue(_ context.Context, projectID, issueID int64) (models.IssueDetail, error) {
	s.projectID = projectID
	s.issueID = issueID
	return s.detail, s.err
}

func authenticatedIssueHandler(store *fakeIssueStore) http.Handler {
	authenticator := eventAuthenticator{
		project: models.Project{ID: 42, Name: "My App"},
	}
	return middleware.RequireAPIKey(authenticator, NewIssueHandler(store))
}

func TestIssueHandlerListsAuthenticatedProjectIssues(t *testing.T) {
	store := &fakeIssueStore{issues: []models.Issue{{ID: 7}}}
	handler := authenticatedIssueHandler(store)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/issues?status=open", nil)
	request.Header.Set("Authorization", "Bearer fly_test-key")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}
	if store.projectID != 42 || store.status != models.IssueStatusOpen {
		t.Fatalf("unexpected query scope: project=%d status=%q", store.projectID, store.status)
	}
}

func TestIssueHandlerGetsIssueForAuthenticatedProject(t *testing.T) {
	store := &fakeIssueStore{detail: models.IssueDetail{
		Issue:        models.Issue{ID: 7},
		Environments: []string{"production"},
	}}
	handler := authenticatedIssueHandler(store)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/issues/7", nil)
	request.Header.Set("Authorization", "Bearer fly_test-key")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}
	if store.projectID != 42 || store.issueID != 7 {
		t.Fatalf("unexpected issue scope: project=%d issue=%d", store.projectID, store.issueID)
	}
}

func TestIssueHandlerDoesNotExposeAnotherProjectsIssue(t *testing.T) {
	store := &fakeIssueStore{err: models.ErrIssueNotFound}
	handler := authenticatedIssueHandler(store)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/issues/99", nil)
	request.Header.Set("Authorization", "Bearer fly_test-key")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, response.Code)
	}
}
