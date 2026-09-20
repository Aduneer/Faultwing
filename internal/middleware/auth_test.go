package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Aduneer/FlyTrap/internal/apikey"
	"github.com/Aduneer/FlyTrap/internal/models"
)

type fakeAuthenticator struct {
	project models.Project
	err     error
	key     string
}

func (a *fakeAuthenticator) AuthenticateProject(_ context.Context, key string) (models.Project, error) {
	a.key = key
	return a.project, a.err
}

func TestRequireAPIKeyAddsProjectToContext(t *testing.T) {
	authenticator := &fakeAuthenticator{
		project: models.Project{ID: 69, Name: "My App"},
	}
	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		project, ok := ProjectFromContext(r.Context())
		if !ok {
			t.Fatal("expected project in request context")
		}
		if project.ID != 69 {
			t.Fatalf("expected project 69, got %d", project.ID)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	handler := RequireAPIKey(authenticator, next)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/events", nil)
	request.Header.Set("Authorization", "Bearer fly_test-key")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, response.Code)
	}
	if !nextCalled {
		t.Fatal("expected next handler to be called")
	}
	if authenticator.key != "fly_test-key" {
		t.Fatalf("unexpected key passed to authenticator: %q", authenticator.key)
	}
}

func TestRequireAPIKeyRejectsMissingKey(t *testing.T) {
	authenticator := &fakeAuthenticator{}
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("next handler should not be called")
	})
	handler := RequireAPIKey(authenticator, next)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/events", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, response.Code)
	}
	if response.Header().Get("WWW-Authenticate") != "Bearer" {
		t.Fatal("expected a Bearer authentication challenge")
	}
}

func TestRequireAPIKeyRejectsInvalidKey(t *testing.T) {
	authenticator := &fakeAuthenticator{err: apikey.ErrInvalid}
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("next handler should not be called")
	})
	handler := RequireAPIKey(authenticator, next)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/events", nil)
	request.Header.Set("Authorization", "Bearer fly_invalid")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, response.Code)
	}
}

func TestRequireAPIKeyHandlesAuthenticatorFailure(t *testing.T) {
	authenticator := &fakeAuthenticator{err: errors.New("database unavailable")}
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("next handler should not be called")
	})
	handler := RequireAPIKey(authenticator, next)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/events", nil)
	request.Header.Set("Authorization", "Bearer fly_test-key")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, response.Code)
	}
}
