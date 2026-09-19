package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/Aduneer/FlyTrap/internal/models"
)

type IssueStore interface {
	CreateIssue(context.Context, models.CreateIssueRequest) (models.Issue, error)
	ListIssues(context.Context) ([]models.Issue, error)
}

type IssueHandler struct {
	store IssueStore
}

func NewIssueHandler(store IssueStore) *IssueHandler {
	return &IssueHandler{store: store}
}

func (h *IssueHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

func (h *IssueHandler) create(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var input models.CreateIssueRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "request body must be valid JSON")
		return
	}

	if strings.TrimSpace(input.Message) == "" {
		writeError(w, http.StatusBadRequest, "message is required")
		return
	}

	issue, err := h.store.CreateIssue(r.Context(), input)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not save issue")
		return
	}

	writeJSON(w, http.StatusCreated, issue)
}

func (h *IssueHandler) list(w http.ResponseWriter, r *http.Request) {
	issues, err := h.store.ListIssues(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load issues")
		return
	}

	writeJSON(w, http.StatusOK, issues)
}
