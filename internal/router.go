package internal

import (
	"net/http"

	"github.com/Aduneer/FlyTrap/internal/api/handlers"
)

type Store interface {
	handlers.EventStore
	handlers.IssueStore
}

func NewRouter(store Store) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/events", handlers.NewEventHandler(store))
	mux.Handle("/issues", handlers.NewIssueHandler(store))

	return mux
}
