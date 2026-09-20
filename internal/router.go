package internal

import (
	"net/http"

	"github.com/Aduneer/FlyTrap/internal/api/handlers"
	"github.com/Aduneer/FlyTrap/internal/middleware"
)

type Store interface {
	handlers.EventStore
	handlers.ProjectStore
	middleware.APIKeyAuthenticator
}

func NewRouter(store Store) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/api/v1/projects", handlers.NewProjectHandler(store))
	mux.Handle("/api/v1/events", middleware.RequireAPIKey(store, handlers.NewEventHandler(store)))

	return mux
}
