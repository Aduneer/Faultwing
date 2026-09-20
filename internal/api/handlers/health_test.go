package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakeHealthStore struct {
	err error
}

func (s fakeHealthStore) Ping(context.Context) error {
	return s.err
}

func TestReadinessHandlerReflectsDatabaseHealth(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		status int
	}{
		{name: "ready", status: http.StatusOK},
		{name: "database unavailable", err: errors.New("connection failed"), status: http.StatusServiceUnavailable},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler := NewReadinessHandler(fakeHealthStore{err: test.err})
			request := httptest.NewRequest(http.MethodGet, "/ready", nil)
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, request)

			if response.Code != test.status {
				t.Fatalf("expected status %d, got %d", test.status, response.Code)
			}
		})
	}
}
