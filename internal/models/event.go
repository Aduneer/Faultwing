package models

import "time"

type Event struct {
	ID         int64     `json:"id"`
	IssueID    int64     `json:"issue_id"`
	Message    string    `json:"message"`
	Stacktrace string    `json:"stacktrace,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

type CreateEventRequest struct {
	Message    string `json:"message"`
	Stacktrace string `json:"stacktrace"`
}
