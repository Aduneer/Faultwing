package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Aduneer/Faultwing/internal/models"
)

type fakeUserSessionAuthenticator struct {
	called bool
}

func (a *fakeUserSessionAuthenticator) AuthenticateSession(context.Context, [32]byte) (models.User, models.Session, error) {
	a.called = true
	return models.User{}, models.Session{}, models.ErrInvalidSession
}

func TestRequireUserSessionRejectsMalformedToken(t *testing.T) {
	authenticator := &fakeUserSessionAuthenticator{}
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("next handler should not be called")
	})
	handler := RequireUserSession(authenticator, next)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/projects", nil)
	request.Header.Set("Authorization", "Bearer not-a-session-token")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, response.Code)
	}
	if authenticator.called {
		t.Fatal("malformed token should be rejected before the database lookup")
	}
}
