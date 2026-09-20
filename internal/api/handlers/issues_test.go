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

type fakeIssueStore struct {
	page      models.IssuePage
	detail    models.IssueDetail
	err       error
	projectID int64
	issueID   int64
	query     models.IssueListQuery
	updated   bool
}

func (s *fakeIssueStore) ListIssues(_ context.Context, projectID int64, query models.IssueListQuery) (models.IssuePage, error) {
	s.projectID = projectID
	s.query = query
	return s.page, s.err
}

func (s *fakeIssueStore) GetIssue(_ context.Context, projectID, issueID int64) (models.IssueDetail, error) {
	s.projectID = projectID
	s.issueID = issueID
	return s.detail, s.err
}

func (s *fakeIssueStore) UpdateIssueStatus(
	_ context.Context,
	projectID, issueID int64,
	status models.IssueStatus,
) (models.Issue, error) {
	s.projectID = projectID
	s.issueID = issueID
	s.query.Status = status
	s.updated = true
	return models.Issue{ID: issueID, ProjectID: projectID, Status: status}, s.err
}

func authenticatedIssueHandler(store *fakeIssueStore) http.Handler {
	authenticator := eventAuthenticator{
		project: models.Project{ID: 42, Name: "My App"},
	}
	return middleware.RequireAPIKey(authenticator, NewIssueHandler(store))
}

func TestIssueHandlerListsAuthenticatedProjectIssues(t *testing.T) {
	lastSeen := time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC)
	store := &fakeIssueStore{page: models.IssuePage{
		Issues:  []models.Issue{{ID: 7, LastSeen: lastSeen}},
		HasMore: true,
	}}
	handler := authenticatedIssueHandler(store)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/issues?status=open&limit=1", nil)
	request.Header.Set("Authorization", "Bearer fly_test-key")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}
	if store.projectID != 42 || store.query.Status != models.IssueStatusOpen || store.query.Limit != 1 {
		t.Fatalf("unexpected query scope: project=%d query=%#v", store.projectID, store.query)
	}

	var result issueListResponse
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(result.Issues) != 1 || result.NextCursor == "" {
		t.Fatalf("expected one issue and a next cursor, got %#v", result)
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

func TestIssueHandlerUpdatesStatus(t *testing.T) {
	store := &fakeIssueStore{}
	handler := authenticatedIssueHandler(store)
	request := httptest.NewRequest(http.MethodPatch, "/api/v1/issues/7", strings.NewReader(`{"status":"resolved"}`))
	request.Header.Set("Authorization", "Bearer fly_test-key")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}
	if store.projectID != 42 || store.issueID != 7 || store.query.Status != models.IssueStatusResolved {
		t.Fatalf("unexpected update: project=%d issue=%d status=%q", store.projectID, store.issueID, store.query.Status)
	}
}

func TestIssueHandlerRejectsInvalidStatus(t *testing.T) {
	store := &fakeIssueStore{}
	handler := authenticatedIssueHandler(store)
	request := httptest.NewRequest(http.MethodPatch, "/api/v1/issues/7", strings.NewReader(`{"status":"closed"}`))
	request.Header.Set("Authorization", "Bearer fly_test-key")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, response.Code)
	}
	if store.updated {
		t.Fatal("invalid status should not reach the store")
	}
}
