package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/Aduneer/FlyTrap/internal/models"
)

const maxProjectRequestSize = 1 << 20

type ProjectStore interface {
	CreateProject(context.Context, string) (models.Project, string, error)
}

type ProjectHandler struct {
	store ProjectStore
}

type createProjectRequest struct {
	Name string `json:"name"`
}

type createProjectResponse struct {
	Project models.Project `json:"project"`
	APIKey  string         `json:"api_key"`
}

func NewProjectHandler(store ProjectStore) *ProjectHandler {
	return &ProjectHandler{store: store}
}

func (h *ProjectHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	h.create(w, r)
}

func (h *ProjectHandler) create(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxProjectRequestSize)
	defer r.Body.Close()

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	var input createProjectRequest
	if err := decoder.Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "request body must be valid JSON")
		return
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		writeError(w, http.StatusBadRequest, "request body must contain one JSON object")
		return
	}

	name := strings.TrimSpace(input.Name)
	if name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if utf8.RuneCountInString(name) > 100 {
		writeError(w, http.StatusBadRequest, "name must be 100 characters or fewer")
		return
	}

	project, key, err := h.store.CreateProject(r.Context(), name)
	if err != nil {
		if errors.Is(err, models.ErrProjectNameTaken) {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "could not create project")
		return
	}

	writeJSON(w, http.StatusCreated, createProjectResponse{
		Project: project,
		APIKey:  key,
	})
}
