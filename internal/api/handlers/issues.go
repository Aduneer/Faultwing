package handlers

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/Aduneer/FlyTrap/internal/middleware"
	"github.com/Aduneer/FlyTrap/internal/models"
)

type IssueStore interface {
	ListIssues(context.Context, int64, models.IssueStatus) ([]models.Issue, error)
	GetIssue(context.Context, int64, int64) (models.IssueDetail, error)
}

type IssueHandler struct {
	store IssueStore
}

func NewIssueHandler(store IssueStore) *IssueHandler {
	return &IssueHandler{store: store}
}

func (h *IssueHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	project, ok := middleware.ProjectFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusInternalServerError, "authenticated project is missing")
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/v1/issues")
	path = strings.Trim(path, "/")
	if path == "" {
		h.list(w, r, project.ID)
		return
	}

	issueID, err := strconv.ParseInt(path, 10, 64)
	if err != nil || issueID < 1 {
		writeError(w, http.StatusNotFound, "issue not found")
		return
	}

	h.get(w, r, project.ID, issueID)
}

func (h *IssueHandler) list(w http.ResponseWriter, r *http.Request, projectID int64) {
	status := models.IssueStatus(r.URL.Query().Get("status"))
	if status != "" && !status.Valid() {
		writeError(w, http.StatusBadRequest, "status must be open, resolved, or ignored")
		return
	}

	issues, err := h.store.ListIssues(r.Context(), projectID, status)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load issues")
		return
	}

	writeJSON(w, http.StatusOK, issues)
}

func (h *IssueHandler) get(w http.ResponseWriter, r *http.Request, projectID, issueID int64) {
	detail, err := h.store.GetIssue(r.Context(), projectID, issueID)
	if err != nil {
		if errors.Is(err, models.ErrIssueNotFound) {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "could not load issue")
		return
	}

	writeJSON(w, http.StatusOK, detail)
}
