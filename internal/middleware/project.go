package middleware

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/Aduneer/Faultwing/internal/models"
)

type ProjectOwnerStore interface {
	GetProjectForOwner(context.Context, int64, int64) (models.Project, error)
}

func RequireOwnedProject(store ProjectOwnerStore, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := UserFromContext(r.Context())
		if !ok {
			writeAuthError(w, http.StatusInternalServerError, "authenticated user is missing")
			return
		}

		projectID, err := strconv.ParseInt(r.PathValue("projectID"), 10, 64)
		if err != nil || projectID < 1 {
			writeAuthError(w, http.StatusNotFound, models.ErrProjectNotFound.Error())
			return
		}
		project, err := store.GetProjectForOwner(r.Context(), user.ID, projectID)
		if err != nil {
			if errors.Is(err, models.ErrProjectNotFound) {
				writeAuthError(w, http.StatusNotFound, err.Error())
				return
			}
			writeAuthError(w, http.StatusInternalServerError, "could not authorize project")
			return
		}

		ctx := context.WithValue(r.Context(), projectContextKey{}, project)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
