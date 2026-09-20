package internal

import (
	"net/http"

	"github.com/Aduneer/FlyTrap/internal/api/handlers"
	"github.com/Aduneer/FlyTrap/internal/middleware"
)

type Store interface {
	handlers.EventStore
	handlers.HealthStore
	handlers.IssueStore
	handlers.ProjectStore
	middleware.APIKeyAuthenticator
}

func NewRouter(store Store) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/health", handlers.NewHealthHandler())
	mux.Handle("/ready", handlers.NewReadinessHandler(store))
	mux.Handle("/api/v1/projects", handlers.NewProjectHandler(store))
	mux.Handle("/api/v1/events", middleware.RequireAPIKey(store, handlers.NewEventHandler(store)))
	issueHandler := middleware.RequireAPIKey(store, handlers.NewIssueHandler(store))
	mux.Handle("/api/v1/issues", issueHandler)
	mux.Handle("/api/v1/issues/", issueHandler)

	return mux
}
