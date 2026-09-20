package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/Aduneer/FlyTrap/internal/middleware"
	"github.com/Aduneer/FlyTrap/internal/models"
)

type EventStore interface {
	EnqueueEvent(context.Context, int64, models.CreateEventRequest) (models.EventJob, error)
}

type EventHandler struct {
	store EventStore
}

type enqueueEventResponse struct {
	JobID    int64     `json:"job_id"`
	QueuedAt time.Time `json:"queued_at"`
}

func NewEventHandler(store EventStore) *EventHandler {
	return &EventHandler{store: store}
}

func (h *EventHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	h.create(w, r)
}

func (h *EventHandler) create(w http.ResponseWriter, r *http.Request) {
	project, ok := middleware.ProjectFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusInternalServerError, "authenticated project is missing")
		return
	}

	defer r.Body.Close()

	var input models.CreateEventRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "request body must be valid JSON")
		return
	}

	input.ExceptionType = strings.TrimSpace(input.ExceptionType)
	input.Message = strings.TrimSpace(input.Message)
	input.Environment = strings.TrimSpace(input.Environment)
	input.Release = strings.TrimSpace(input.Release)

	if input.ExceptionType == "" {
		writeError(w, http.StatusBadRequest, "exception_type is required")
		return
	}
	if input.Message == "" {
		writeError(w, http.StatusBadRequest, "message is required")
		return
	}
	if input.Environment == "" {
		writeError(w, http.StatusBadRequest, "environment is required")
		return
	}

	job, err := h.store.EnqueueEvent(r.Context(), project.ID, input)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not queue event")
		return
	}

	writeJSON(w, http.StatusAccepted, enqueueEventResponse{
		JobID:    job.ID,
		QueuedAt: job.CreatedAt,
	})
}
