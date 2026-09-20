package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/Aduneer/FlyTrap/internal/apikey"
	"github.com/Aduneer/FlyTrap/internal/models"
)

type APIKeyAuthenticator interface {
	AuthenticateProject(context.Context, string) (models.Project, error)
}

type projectContextKey struct{}

func RequireAPIKey(authenticator APIKeyAuthenticator, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key, ok := bearerToken(r.Header.Get("Authorization"))
		if !ok {
			writeAuthError(w, http.StatusUnauthorized, "a bearer API key is required")
			return
		}

		project, err := authenticator.AuthenticateProject(r.Context(), key)
		if err != nil {
			if errors.Is(err, apikey.ErrInvalid) {
				writeAuthError(w, http.StatusUnauthorized, "invalid API key")
				return
			}
			writeAuthError(w, http.StatusInternalServerError, "could not authenticate API key")
			return
		}

		ctx := context.WithValue(r.Context(), projectContextKey{}, project)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func ProjectFromContext(ctx context.Context) (models.Project, bool) {
	project, ok := ctx.Value(projectContextKey{}).(models.Project)
	return project, ok
}

func bearerToken(header string) (string, bool) {
	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", false
	}
	return parts[1], true
}

func writeAuthError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	if status == http.StatusUnauthorized {
		w.Header().Set("WWW-Authenticate", "Bearer")
	}
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
