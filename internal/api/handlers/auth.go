package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"github.com/Aduneer/FlyTrap/internal/middleware"
	"github.com/Aduneer/FlyTrap/internal/models"
	"github.com/Aduneer/FlyTrap/internal/userauth"
)

const (
	maxAuthRequestSize = 1 << 20
	minPasswordLength  = 8
	maxPasswordLength  = 72
)

type AuthStore interface {
	CreateUser(context.Context, string, string) (models.User, error)
	FindUserByEmail(context.Context, string) (models.User, error)
	CreateSession(context.Context, int64, [32]byte, time.Time) (models.Session, error)
	RevokeSession(context.Context, [32]byte) error
}

type AuthHandler struct {
	store AuthStore
}

type credentialsRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginResponse struct {
	User      models.User `json:"user"`
	Token     string      `json:"token"`
	ExpiresAt time.Time   `json:"expires_at"`
}

func NewAuthHandler(store AuthStore) *AuthHandler {
	return &AuthHandler{store: store}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}

	input, ok := decodeCredentials(w, r)
	if !ok {
		return
	}

	passwordHash, err := userauth.HashPassword(input.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not secure password")
		return
	}
	user, err := h.store.CreateUser(r.Context(), input.Email, passwordHash)
	if err != nil {
		if errors.Is(err, models.ErrEmailTaken) {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "could not create user")
		return
	}

	writeJSON(w, http.StatusCreated, user)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}

	input, ok := decodeCredentials(w, r)
	if !ok {
		return
	}

	user, err := h.store.FindUserByEmail(r.Context(), input.Email)
	if err != nil {
		if errors.Is(err, models.ErrUserNotFound) {
			writeError(w, http.StatusUnauthorized, "invalid email or password")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not log in")
		return
	}
	if !userauth.PasswordMatches(user.PasswordHash, input.Password) {
		writeError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	token, tokenHash, err := userauth.GenerateSessionToken()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create session")
		return
	}
	expiresAt := time.Now().Add(userauth.SessionDuration)
	session, err := h.store.CreateSession(r.Context(), user.ID, tokenHash, expiresAt)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create session")
		return
	}

	writeJSON(w, http.StatusOK, loginResponse{
		User:      user,
		Token:     token,
		ExpiresAt: session.ExpiresAt,
	})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}

	tokenHash, ok := middleware.SessionHashFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusInternalServerError, "authenticated session is missing")
		return
	}
	if err := h.store.RevokeSession(r.Context(), tokenHash); err != nil {
		writeError(w, http.StatusInternalServerError, "could not log out")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func decodeCredentials(w http.ResponseWriter, r *http.Request) (credentialsRequest, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, maxAuthRequestSize)
	defer r.Body.Close()

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	var input credentialsRequest
	if err := decoder.Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "request body must be valid JSON")
		return credentialsRequest{}, false
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		writeError(w, http.StatusBadRequest, "request body must contain one JSON object")
		return credentialsRequest{}, false
	}

	input.Email = strings.ToLower(strings.TrimSpace(input.Email))
	address, err := mail.ParseAddress(input.Email)
	if err != nil || address.Address != input.Email || len(input.Email) > 254 {
		writeError(w, http.StatusBadRequest, "email must be valid")
		return credentialsRequest{}, false
	}
	passwordLength := len([]byte(input.Password))
	if passwordLength < minPasswordLength || passwordLength > maxPasswordLength {
		writeError(w, http.StatusBadRequest, "password must be between 8 and 72 bytes")
		return credentialsRequest{}, false
	}

	return input, true
}
