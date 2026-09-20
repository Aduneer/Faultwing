package handlers

import (
	"context"
	"net/http"
	"time"
)

type HealthHandler struct{}

type HealthStore interface {
	Ping(context.Context) error
}

type ReadinessHandler struct {
	store HealthStore
}

func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

func NewReadinessHandler(store HealthStore) *ReadinessHandler {
	return &ReadinessHandler{store: store}
}

func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, "GET")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *ReadinessHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, "GET")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if err := h.store.Ping(ctx); err != nil {
		writeError(w, http.StatusServiceUnavailable, "database is unavailable")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}
