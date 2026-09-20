package models

import "time"

type Issue struct {
	ID            int64     `json:"id"`
	ProjectID     int64     `json:"project_id"`
	Fingerprint   string    `json:"fingerprint"`
	ExceptionType string    `json:"exception_type"`
	Message       string    `json:"message"`
	Stacktrace    string    `json:"stacktrace,omitempty"`
	EventCount    int64     `json:"event_count"`
	FirstSeen     time.Time `json:"first_seen"`
	LastSeen      time.Time `json:"last_seen"`
	CreatedAt     time.Time `json:"created_at"`
}

type CreateIssueRequest struct {
	Message    string `json:"message"`
	Stacktrace string `json:"stacktrace"`
}
