package handlers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/Aduneer/FlyTrap/internal/middleware"
	"github.com/Aduneer/FlyTrap/internal/models"
)

const (
	defaultIssuePageSize = 50
	maxIssuePageSize     = 100
)

type IssueStore interface {
	ListIssues(context.Context, int64, models.IssueListQuery) (models.IssuePage, error)
	GetIssue(context.Context, int64, int64) (models.IssueDetail, error)
	UpdateIssueStatus(context.Context, int64, int64, models.IssueStatus) (models.Issue, error)
}

type IssueHandler struct {
	store IssueStore
}

type updateIssueRequest struct {
	Status models.IssueStatus `json:"status"`
}

type issueListResponse struct {
	Issues     []models.Issue `json:"issues"`
	NextCursor string         `json:"next_cursor,omitempty"`
}

func NewIssueHandler(store IssueStore) *IssueHandler {
	return &IssueHandler{store: store}
}

func (h *IssueHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	project, ok := middleware.ProjectFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusInternalServerError, "authenticated project is missing")
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/v1/issues")
	path = strings.Trim(path, "/")
	if path == "" {
		if r.Method != http.MethodGet {
			methodNotAllowed(w, "GET")
			return
		}
		h.list(w, r, project.ID)
		return
	}

	issueID, err := strconv.ParseInt(path, 10, 64)
	if err != nil || issueID < 1 {
		writeError(w, http.StatusNotFound, "issue not found")
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.get(w, r, project.ID, issueID)
	case http.MethodPatch:
		h.updateStatus(w, r, project.ID, issueID)
	default:
		methodNotAllowed(w, "GET, PATCH")
	}
}

func (h *IssueHandler) list(w http.ResponseWriter, r *http.Request, projectID int64) {
	status := models.IssueStatus(r.URL.Query().Get("status"))
	if status != "" && !status.Valid() {
		writeError(w, http.StatusBadRequest, "status must be open, resolved, or ignored")
		return
	}

	limit := defaultIssuePageSize
	if value := r.URL.Query().Get("limit"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 || parsed > maxIssuePageSize {
			writeError(w, http.StatusBadRequest, "limit must be between 1 and 100")
			return
		}
		limit = parsed
	}

	var cursor *models.IssueCursor
	if value := r.URL.Query().Get("cursor"); value != "" {
		parsed, err := decodeIssueCursor(value)
		if err != nil {
			writeError(w, http.StatusBadRequest, "cursor is invalid")
			return
		}
		cursor = &parsed
	}

	page, err := h.store.ListIssues(r.Context(), projectID, models.IssueListQuery{
		Status: status,
		Limit:  limit,
		Cursor: cursor,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load issues")
		return
	}

	response := issueListResponse{Issues: page.Issues}
	if page.HasMore && len(page.Issues) > 0 {
		last := page.Issues[len(page.Issues)-1]
		response.NextCursor = encodeIssueCursor(models.IssueCursor{
			LastSeen: last.LastSeen,
			ID:       last.ID,
		})
	}

	writeJSON(w, http.StatusOK, response)
}

func encodeIssueCursor(cursor models.IssueCursor) string {
	data, _ := json.Marshal(cursor)
	return base64.RawURLEncoding.EncodeToString(data)
}

func decodeIssueCursor(value string) (models.IssueCursor, error) {
	data, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return models.IssueCursor{}, err
	}

	var cursor models.IssueCursor
	if err := json.Unmarshal(data, &cursor); err != nil {
		return models.IssueCursor{}, err
	}
	if cursor.ID < 1 || cursor.LastSeen.IsZero() {
		return models.IssueCursor{}, errors.New("invalid cursor values")
	}
	return cursor, nil
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

func (h *IssueHandler) updateStatus(w http.ResponseWriter, r *http.Request, projectID, issueID int64) {
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	defer r.Body.Close()

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	var input updateIssueRequest
	if err := decoder.Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "request body must be valid JSON")
		return
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		writeError(w, http.StatusBadRequest, "request body must contain one JSON object")
		return
	}
	if !input.Status.Valid() {
		writeError(w, http.StatusBadRequest, "status must be open, resolved, or ignored")
		return
	}

	issue, err := h.store.UpdateIssueStatus(r.Context(), projectID, issueID, input.Status)
	if err != nil {
		if errors.Is(err, models.ErrIssueNotFound) {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "could not update issue")
		return
	}

	writeJSON(w, http.StatusOK, issue)
}
