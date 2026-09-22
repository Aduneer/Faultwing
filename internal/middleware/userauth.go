package middleware

import (
	"context"
	"errors"
	"net/http"

	"github.com/Aduneer/Faultwing/internal/models"
	"github.com/Aduneer/Faultwing/internal/userauth"
)

type UserSessionAuthenticator interface {
	AuthenticateSession(context.Context, [32]byte) (models.User, models.Session, error)
}

type authenticatedUser struct {
	user      models.User
	session   models.Session
	tokenHash [32]byte
}

type userContextKey struct{}

func RequireUserSession(authenticator UserSessionAuthenticator, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, ok := bearerToken(r.Header.Get("Authorization"))
		if !ok {
			writeAuthError(w, http.StatusUnauthorized, "a bearer session token is required")
			return
		}

		tokenHash, ok := userauth.ParseSessionToken(token)
		if !ok {
			writeAuthError(w, http.StatusUnauthorized, "invalid or expired session")
			return
		}
		user, session, err := authenticator.AuthenticateSession(r.Context(), tokenHash)
		if err != nil {
			if errors.Is(err, models.ErrInvalidSession) {
				writeAuthError(w, http.StatusUnauthorized, err.Error())
				return
			}
			writeAuthError(w, http.StatusInternalServerError, "could not authenticate session")
			return
		}

		identity := authenticatedUser{user: user, session: session, tokenHash: tokenHash}
		ctx := context.WithValue(r.Context(), userContextKey{}, identity)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func UserFromContext(ctx context.Context) (models.User, bool) {
	identity, ok := ctx.Value(userContextKey{}).(authenticatedUser)
	return identity.user, ok
}

func SessionHashFromContext(ctx context.Context) ([32]byte, bool) {
	identity, ok := ctx.Value(userContextKey{}).(authenticatedUser)
	return identity.tokenHash, ok
}
