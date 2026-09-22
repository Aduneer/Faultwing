package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Aduneer/Faultwing/internal/models"
)

func TestRateLimitByProjectRejectsRequestsBeyondBurst(t *testing.T) {
	authenticator := &fakeAuthenticator{project: models.Project{ID: 42}}
	limiter := NewProjectRateLimiter(1, 2)
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})
	handler := RequireAPIKey(authenticator, RateLimitByProject(limiter, next))

	for requestNumber := 1; requestNumber <= 3; requestNumber++ {
		request := httptest.NewRequest(http.MethodPost, "/api/v1/events", nil)
		request.Header.Set("Authorization", "Bearer faultwing_test-key")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)

		if requestNumber <= 2 && response.Code != http.StatusCreated {
			t.Fatalf("request %d: expected status %d, got %d", requestNumber, http.StatusCreated, response.Code)
		}
		if requestNumber == 3 {
			if response.Code != http.StatusTooManyRequests {
				t.Fatalf("expected status %d, got %d", http.StatusTooManyRequests, response.Code)
			}
			if response.Header().Get("Retry-After") == "" {
				t.Fatal("expected Retry-After header")
			}
		}
	}

	otherProject := &fakeAuthenticator{project: models.Project{ID: 99}}
	otherHandler := RequireAPIKey(otherProject, RateLimitByProject(limiter, next))
	request := httptest.NewRequest(http.MethodPost, "/api/v1/events", nil)
	request.Header.Set("Authorization", "Bearer faultwing_other-key")
	response := httptest.NewRecorder()
	otherHandler.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("expected another project to have its own allowance, got %d", response.Code)
	}
}
