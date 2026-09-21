package internal

import (
	"net/http"

	"github.com/Aduneer/FlyTrap/internal/api/handlers"
	"github.com/Aduneer/FlyTrap/internal/middleware"
)

type Store interface {
	handlers.AuthStore
	handlers.EventStore
	handlers.HealthStore
	handlers.IssueStore
	handlers.ProjectStore
	middleware.APIKeyAuthenticator
	middleware.UserSessionAuthenticator
}

func NewRouter(store Store, eventLimiter *middleware.ProjectRateLimiter) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/health", handlers.NewHealthHandler())
	mux.Handle("/ready", handlers.NewReadinessHandler(store))
	authHandler := handlers.NewAuthHandler(store)
	mux.HandleFunc("/api/v1/auth/register", authHandler.Register)
	mux.HandleFunc("/api/v1/auth/login", authHandler.Login)
	mux.Handle("/api/v1/auth/logout", middleware.RequireUserSession(store, http.HandlerFunc(authHandler.Logout)))
	mux.Handle("/api/v1/projects", handlers.NewProjectHandler(store))
	eventHandler := middleware.RateLimitByProject(eventLimiter, handlers.NewEventHandler(store))
	mux.Handle("/api/v1/events", middleware.RequireAPIKey(store, eventHandler))
	issueHandler := middleware.RequireAPIKey(store, handlers.NewIssueHandler(store))
	mux.Handle("/api/v1/issues", issueHandler)
	mux.Handle("/api/v1/issues/", issueHandler)

	return mux
}
