package models

import "time"

type Issue struct {
	ID          int64     `json:"id"`
	Fingerprint string    `json:"fingerprint"`
	Message     string    `json:"message"`
	Stacktrace  string    `json:"stacktrace,omitempty"`
	EventCount  int64     `json:"event_count"`
	CreatedAt   time.Time `json:"created_at"`
}

type CreateIssueRequest struct {
	Message    string `json:"message"`
	Stacktrace string `json:"stacktrace"`
}
