package config

import "os"

type Config struct {
	HTTPAddr    string
	DatabaseURL string
}

func Load() Config {
	addr := os.Getenv("HTTP_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://flytrap:flytrap@localhost:5432/flytrap?sslmode=disable"
	}

	return Config{
		HTTPAddr:    addr,
		DatabaseURL: databaseURL,
	}
}
