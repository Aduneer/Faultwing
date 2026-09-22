package internal

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Aduneer/Faultwing/internal/models"
	"github.com/Aduneer/Faultwing/internal/userauth"
	"github.com/gorilla/websocket"
)

type fakeRealtimeStore struct {
	user      models.User
	tokenHash [32]byte
}

func (s fakeRealtimeStore) AuthenticateSession(_ context.Context, tokenHash [32]byte) (models.User, models.Session, error) {
	if tokenHash != s.tokenHash {
		return models.User{}, models.Session{}, models.ErrInvalidSession
	}
	return s.user, models.Session{ID: 1, UserID: s.user.ID}, nil
}

func (s fakeRealtimeStore) GetProjectForOwner(_ context.Context, ownerID, projectID int64) (models.Project, error) {
	if ownerID != s.user.ID || (projectID != 1 && projectID != 2) {
		return models.Project{}, models.ErrProjectNotFound
	}
	return models.Project{ID: projectID, OwnerID: ownerID}, nil
}

func TestRealtimeWebSocketPublishesOnlyToMatchingProject(t *testing.T) {
	hub := NewRealtimeHub()
	defer hub.Close()
	token, tokenHash, err := userauth.GenerateSessionToken()
	if err != nil {
		t.Fatalf("generate session token: %v", err)
	}
	store := fakeRealtimeStore{user: models.User{ID: 42}, tokenHash: tokenHash}
	mux := http.NewServeMux()
	mux.Handle("/api/v1/projects/{projectID}/realtime", NewRealtimeHandler(store, hub))
	server := httptest.NewServer(mux)
	defer server.Close()

	projectOne := openRealtimeTestConnection(t, server.URL, 1, token)
	defer projectOne.Close()
	projectTwo := openRealtimeTestConnection(t, server.URL, 2, token)
	defer projectTwo.Close()

	hub.Publish(models.IssueUpdate{
		Type:      models.IssueUpdateType,
		ProjectID: 1,
		IssueID:   7,
	})

	var update models.IssueUpdate
	if err := projectOne.ReadJSON(&update); err != nil {
		t.Fatalf("read project update: %v", err)
	}
	if update.ProjectID != 1 || update.IssueID != 7 {
		t.Fatalf("unexpected project update: %#v", update)
	}
	_ = projectTwo.SetReadDeadline(time.Now().Add(50 * time.Millisecond))
	if _, message, err := projectTwo.ReadMessage(); err == nil {
		t.Fatalf("another project received update %s", message)
	}
}

func openRealtimeTestConnection(t *testing.T, serverURL string, projectID int, token string) *websocket.Conn {
	t.Helper()
	url := strings.Replace(serverURL, "http://", "ws://", 1) +
		"/api/v1/projects/" + strconv.Itoa(projectID) + "/realtime"
	connection, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("dial project %d websocket: %v", projectID, err)
	}
	if err := connection.WriteJSON(realtimeAuthMessage{Token: token}); err != nil {
		connection.Close()
		t.Fatalf("authenticate project %d websocket: %v", projectID, err)
	}
	var ready realtimeReadyMessage
	if err := connection.ReadJSON(&ready); err != nil {
		connection.Close()
		t.Fatalf("read project %d ready message: %v", projectID, err)
	}
	if ready.Type != "realtime.ready" || ready.ProjectID != int64(projectID) {
		connection.Close()
		t.Fatalf("unexpected ready message: %#v", ready)
	}
	return connection
}
