package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/Aduneer/FlyTrap/internal/middleware"
	"github.com/Aduneer/FlyTrap/internal/models"
)

type EventStore interface {
	CreateEvent(context.Context, int64, models.CreateEventRequest) (models.Event, error)
}

type EventHandler struct {
	store EventStore
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

	event, err := h.store.CreateEvent(r.Context(), project.ID, input)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not save event")
		return
	}

	writeJSON(w, http.StatusCreated, event)
}
