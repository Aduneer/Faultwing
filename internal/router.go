package internal

import (
	"net/http"

	"github.com/Aduneer/FlyTrap/internal/api/handlers"
)

type Store interface {
	handlers.EventStore
	handlers.IssueStore
	handlers.ProjectStore
}

func NewRouter(store Store) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/events", handlers.NewEventHandler(store))
	mux.Handle("/issues", handlers.NewIssueHandler(store))
	mux.Handle("/api/v1/projects", handlers.NewProjectHandler(store))

	return mux
}
