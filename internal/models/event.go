package models

import "time"

type Event struct {
	ID            int64     `json:"id"`
	IssueID       int64     `json:"issue_id"`
	ExceptionType string    `json:"exception_type"`
	Message       string    `json:"message"`
	Stacktrace    string    `json:"stacktrace,omitempty"`
	Environment   string    `json:"environment"`
	Release       string    `json:"release,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

type CreateEventRequest struct {
	ExceptionType string `json:"exception_type"`
	Message       string `json:"message"`
	Stacktrace    string `json:"stacktrace"`
	Environment   string `json:"environment"`
	Release       string `json:"release"`
}
