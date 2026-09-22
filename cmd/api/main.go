package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Aduneer/Faultwing/internal"
	"github.com/Aduneer/Faultwing/internal/config"
	"github.com/Aduneer/Faultwing/internal/database"
	"github.com/Aduneer/Faultwing/internal/middleware"
)

const (
	readHeaderTimeout  = 5 * time.Second
	readTimeout        = 15 * time.Second
	writeTimeout       = 30 * time.Second
	idleTimeout        = 60 * time.Second
	shutdownTimeout    = 10 * time.Second
	listenerRetryDelay = time.Second
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}

	pool, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer pool.Close()

	store := database.NewStore(pool)
	realtimeHub := internal.NewRealtimeHub()
	defer realtimeHub.Close()
	go forwardIssueUpdates(ctx, store, realtimeHub)
	eventLimiter := middleware.NewProjectRateLimiter(
		cfg.EventRateLimitPerMinute,
		cfg.EventRateLimitBurst,
	)
	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           internal.NewRouter(store, eventLimiter, realtimeHub),
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}

	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- server.ListenAndServe()
	}()

	log.Printf("Faultwing API listening on %s", cfg.HTTPAddr)
	select {
	case err := <-serverErrors:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve HTTP: %w", err)
		}
		return nil
	case <-ctx.Done():
		stop()
		realtimeHub.Close()
		log.Println("shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		_ = server.Close()
		return fmt.Errorf("shut down HTTP server: %w", err)
	}
	if err := <-serverErrors; err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serve HTTP during shutdown: %w", err)
	}

	log.Println("shutdown complete")
	return nil
}

func forwardIssueUpdates(ctx context.Context, store *database.Store, hub *internal.RealtimeHub) {
	for ctx.Err() == nil {
		listener, err := store.OpenIssueUpdateListener(ctx)
		if err == nil {
			for ctx.Err() == nil {
				update, waitErr := listener.Wait(ctx)
				if waitErr != nil {
					err = waitErr
					break
				}
				hub.Publish(update)
			}
			if closeErr := listener.Close(); err == nil {
				err = closeErr
			}
		}
		if ctx.Err() != nil {
			return
		}
		log.Printf("issue update listener: %v", err)

		timer := time.NewTimer(listenerRetryDelay)
		select {
		case <-timer.C:
		case <-ctx.Done():
			timer.Stop()
			return
		}
	}
}
