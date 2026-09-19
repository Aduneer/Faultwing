package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/Aduneer/FlyTrap/internal/models"
)

type EventStore interface {
	CreateEvent(context.Context, models.CreateEventRequest) (models.Event, error)
	ListEvents(context.Context) ([]models.Event, error)
}

type EventHandler struct {
	store EventStore
}

func NewEventHandler(store EventStore) *EventHandler {
	return &EventHandler{store: store}
}

func (h *EventHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.create(w, r)
	case http.MethodGet:
		h.list(w, r)
	default:
		w.Header().Set("Allow", "GET, POST")
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *EventHandler) create(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var input models.CreateEventRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "request body must be valid JSON")
		return
	}

	if strings.TrimSpace(input.Message) == "" {
		writeError(w, http.StatusBadRequest, "message is required")
		return
	}

	event, err := h.store.CreateEvent(r.Context(), input)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not save event")
		return
	}

	writeJSON(w, http.StatusCreated, event)
}

func (h *EventHandler) list(w http.ResponseWriter, r *http.Request) {
	events, err := h.store.ListEvents(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load events")
		return
	}

	writeJSON(w, http.StatusOK, events)
}
