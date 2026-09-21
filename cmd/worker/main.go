package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/Aduneer/FlyTrap/internal/config"
	"github.com/Aduneer/FlyTrap/internal/database"
	"github.com/Aduneer/FlyTrap/internal/worker"
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

	processor := worker.NewProcessor(database.NewStore(pool), log.Default())
	log.Println("FlyTrap worker started")
	processor.Run(ctx)
	log.Println("FlyTrap worker stopped")

	return nil
}
