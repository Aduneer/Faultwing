package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Aduneer/FlyTrap/internal/middleware"
	"github.com/Aduneer/FlyTrap/internal/models"
	"github.com/Aduneer/FlyTrap/internal/userauth"
)

type fakeAuthStore struct {
	user      models.User
	session   models.Session
	tokenHash [32]byte
	revoked   bool
}

func (s *fakeAuthStore) CreateUser(_ context.Context, email, passwordHash string) (models.User, error) {
	s.user = models.User{
		ID:           1,
		Email:        email,
		PasswordHash: passwordHash,
		CreatedAt:    time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC),
	}
	return s.user, nil
}

func (s *fakeAuthStore) FindUserByEmail(context.Context, string) (models.User, error) {
	return s.user, nil
}

func (s *fakeAuthStore) CreateSession(_ context.Context, userID int64, tokenHash [32]byte, expiresAt time.Time) (models.Session, error) {
	s.tokenHash = tokenHash
	s.session = models.Session{ID: 1, UserID: userID, ExpiresAt: expiresAt}
	return s.session, nil
}

func (s *fakeAuthStore) AuthenticateSession(_ context.Context, tokenHash [32]byte) (models.User, models.Session, error) {
	if tokenHash != s.tokenHash || s.revoked {
		return models.User{}, models.Session{}, models.ErrInvalidSession
	}
	return s.user, s.session, nil
}

func (s *fakeAuthStore) RevokeSession(_ context.Context, tokenHash [32]byte) error {
	if tokenHash == s.tokenHash {
		s.revoked = true
	}
	return nil
}

func TestAuthHandlerRegisterLoginAndLogout(t *testing.T) {
	store := &fakeAuthStore{}
	handler := NewAuthHandler(store)
	credentials := `{"email":" Learner@Example.com ","password":"correct horse battery staple"}`

	registerRequest := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(credentials))
	registerResponse := httptest.NewRecorder()
	handler.Register(registerResponse, registerRequest)
	if registerResponse.Code != http.StatusCreated {
		t.Fatalf("register: expected status %d, got %d", http.StatusCreated, registerResponse.Code)
	}
	if store.user.Email != "learner@example.com" || !userauth.PasswordMatches(store.user.PasswordHash, "correct horse battery staple") {
		t.Fatalf("registration stored unexpected credentials: %#v", store.user)
	}

	loginRequest := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(credentials))
	loginResponseRecorder := httptest.NewRecorder()
	handler.Login(loginResponseRecorder, loginRequest)
	if loginResponseRecorder.Code != http.StatusOK {
		t.Fatalf("login: expected status %d, got %d", http.StatusOK, loginResponseRecorder.Code)
	}
	var login loginResponse
	if err := json.NewDecoder(loginResponseRecorder.Body).Decode(&login); err != nil {
		t.Fatalf("decode login response: %v", err)
	}
	parsedHash, ok := userauth.ParseSessionToken(login.Token)
	if !ok || parsedHash != store.tokenHash || !login.ExpiresAt.After(time.Now()) {
		t.Fatalf("unexpected login session: %#v", login)
	}

	logoutHandler := middleware.RequireUserSession(store, http.HandlerFunc(handler.Logout))
	logoutRequest := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	logoutRequest.Header.Set("Authorization", "Bearer "+login.Token)
	logoutResponse := httptest.NewRecorder()
	logoutHandler.ServeHTTP(logoutResponse, logoutRequest)
	if logoutResponse.Code != http.StatusNoContent || !store.revoked {
		t.Fatalf("logout: expected revoked session and status %d, got status=%d revoked=%t", http.StatusNoContent, logoutResponse.Code, store.revoked)
	}
}
