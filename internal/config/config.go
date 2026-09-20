package config

import (
	"fmt"
	"os"
	"strconv"
)

const (
	defaultEventRateLimit = 60
	defaultEventRateBurst = 10
)

type Config struct {
	HTTPAddr                string
	DatabaseURL             string
	EventRateLimitPerMinute int
	EventRateLimitBurst     int
}

func Load() (Config, error) {
	addr := os.Getenv("HTTP_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://flytrap:flytrap@localhost:5432/flytrap?sslmode=disable"
	}

	eventsPerMinute, err := positiveInt("EVENT_RATE_LIMIT_PER_MINUTE", defaultEventRateLimit)
	if err != nil {
		return Config{}, err
	}
	burst, err := positiveInt("EVENT_RATE_LIMIT_BURST", defaultEventRateBurst)
	if err != nil {
		return Config{}, err
	}

	return Config{
		HTTPAddr:                addr,
		DatabaseURL:             databaseURL,
		EventRateLimitPerMinute: eventsPerMinute,
		EventRateLimitBurst:     burst,
	}, nil
}

func positiveInt(name string, fallback int) (int, error) {
	value := os.Getenv(name)
	if value == "" {
		return fallback, nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 1 {
		return 0, fmt.Errorf("%s must be a positive integer", name)
	}
	return parsed, nil
}
