package internal

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/Aduneer/FlyTrap/internal/models"
	"github.com/Aduneer/FlyTrap/internal/userauth"
	"github.com/gorilla/websocket"
)

const (
	realtimeAuthTimeout = 5 * time.Second
	realtimeWriteWait   = 10 * time.Second
	realtimePongWait    = 60 * time.Second
	realtimePingPeriod  = 50 * time.Second
	realtimeSendBuffer  = 16
)

type RealtimeStore interface {
	AuthenticateSession(context.Context, [32]byte) (models.User, models.Session, error)
	GetProjectForOwner(context.Context, int64, int64) (models.Project, error)
}

type realtimeClient struct {
	connection *websocket.Conn
	send       chan []byte
}

type RealtimeHub struct {
	mu      sync.Mutex
	clients map[int64]map[*realtimeClient]struct{}
	closed  bool
}

type realtimeAuthMessage struct {
	Token string `json:"token"`
}

type realtimeReadyMessage struct {
	Type      string `json:"type"`
	ProjectID int64  `json:"project_id"`
}

func NewRealtimeHub() *RealtimeHub {
	return &RealtimeHub{clients: make(map[int64]map[*realtimeClient]struct{})}
}

func (h *RealtimeHub) Publish(update models.IssueUpdate) {
	payload, err := json.Marshal(update)
	if err != nil {
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		return
	}
	for client := range h.clients[update.ProjectID] {
		select {
		case client.send <- payload:
		default:
			h.removeLocked(update.ProjectID, client)
			if client.connection != nil {
				_ = client.connection.Close()
			}
		}
	}
}

func (h *RealtimeHub) Close() {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		return
	}
	h.closed = true
	for projectID, clients := range h.clients {
		for client := range clients {
			h.removeLocked(projectID, client)
		}
	}
}

func (h *RealtimeHub) register(projectID int64, client *realtimeClient) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		return false
	}
	if h.clients[projectID] == nil {
		h.clients[projectID] = make(map[*realtimeClient]struct{})
	}
	h.clients[projectID][client] = struct{}{}
	return true
}

func (h *RealtimeHub) unregister(projectID int64, client *realtimeClient) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.removeLocked(projectID, client)
}

func (h *RealtimeHub) removeLocked(projectID int64, client *realtimeClient) {
	clients := h.clients[projectID]
	if _, ok := clients[client]; !ok {
		return
	}
	delete(clients, client)
	close(client.send)
	if len(clients) == 0 {
		delete(h.clients, projectID)
	}
}

func NewRealtimeHandler(store RealtimeStore, hub *RealtimeHub) http.Handler {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		projectID, err := strconv.ParseInt(r.PathValue("projectID"), 10, 64)
		if err != nil || projectID < 1 {
			http.Error(w, models.ErrProjectNotFound.Error(), http.StatusNotFound)
			return
		}

		connection, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		connection.SetReadLimit(4 << 10)
		_ = connection.SetReadDeadline(time.Now().Add(realtimeAuthTimeout))

		var auth realtimeAuthMessage
		if err := connection.ReadJSON(&auth); err != nil {
			closeRealtimeConnection(connection, "authentication required")
			return
		}
		tokenHash, ok := userauth.ParseSessionToken(auth.Token)
		if !ok {
			closeRealtimeConnection(connection, "invalid session")
			return
		}
		user, _, err := store.AuthenticateSession(r.Context(), tokenHash)
		if err != nil {
			closeRealtimeConnection(connection, "invalid session")
			return
		}
		if _, err := store.GetProjectForOwner(r.Context(), user.ID, projectID); err != nil {
			closeRealtimeConnection(connection, "project not found")
			return
		}

		client := &realtimeClient{
			connection: connection,
			send:       make(chan []byte, realtimeSendBuffer),
		}
		if !hub.register(projectID, client) {
			closeRealtimeConnection(connection, "server shutting down")
			return
		}
		defer hub.unregister(projectID, client)
		ready, _ := json.Marshal(realtimeReadyMessage{
			Type:      "realtime.ready",
			ProjectID: projectID,
		})
		client.send <- ready

		_ = connection.SetReadDeadline(time.Now().Add(realtimePongWait))
		connection.SetPongHandler(func(string) error {
			return connection.SetReadDeadline(time.Now().Add(realtimePongWait))
		})
		go client.writeMessages()
		client.readMessages()
	})
}

func (c *realtimeClient) readMessages() {
	defer c.connection.Close()
	for {
		if _, _, err := c.connection.ReadMessage(); err != nil {
			return
		}
	}
}

func (c *realtimeClient) writeMessages() {
	ticker := time.NewTicker(realtimePingPeriod)
	defer ticker.Stop()
	defer c.connection.Close()

	for {
		select {
		case message, ok := <-c.send:
			_ = c.connection.SetWriteDeadline(time.Now().Add(realtimeWriteWait))
			if !ok {
				_ = c.connection.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.connection.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.connection.SetWriteDeadline(time.Now().Add(realtimeWriteWait))
			if err := c.connection.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func closeRealtimeConnection(connection *websocket.Conn, reason string) {
	deadline := time.Now().Add(realtimeWriteWait)
	_ = connection.WriteControl(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.ClosePolicyViolation, reason),
		deadline,
	)
	_ = connection.Close()
}
