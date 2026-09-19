package internal

import (
	"net/http"

	"github.com/Aduneer/FlyTrap/internal/api/handlers"
)

func NewRouter(store handlers.EventStore) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/events", handlers.NewEventHandler(store))

	return mux
}
