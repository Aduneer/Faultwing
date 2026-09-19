package main

import (
	"context"
	"log"
	"net/http"

	"github.com/Aduneer/FlyTrap/internal"
	"github.com/Aduneer/FlyTrap/internal/config"
	"github.com/Aduneer/FlyTrap/internal/database"
)

func main() {
	cfg := config.Load()

	pool, err := database.Open(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	store := database.NewStore(pool)
	server := &http.Server{
		Addr:    cfg.HTTPAddr,
		Handler: internal.NewRouter(store),
	}

	log.Printf("FlyTrap API listening on %s", cfg.HTTPAddr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
