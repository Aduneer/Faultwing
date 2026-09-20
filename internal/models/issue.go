package models

import (
	"errors"
	"time"
)

type IssueStatus string

const (
	IssueStatusOpen     IssueStatus = "open"
	IssueStatusResolved IssueStatus = "resolved"
	IssueStatusIgnored  IssueStatus = "ignored"
)

var ErrIssueNotFound = errors.New("issue not found")

func (s IssueStatus) Valid() bool {
	return s == IssueStatusOpen || s == IssueStatusResolved || s == IssueStatusIgnored
}

type Issue struct {
	ID            int64       `json:"id"`
	ProjectID     int64       `json:"project_id"`
	Fingerprint   string      `json:"fingerprint"`
	ExceptionType string      `json:"exception_type"`
	Message       string      `json:"message"`
	Stacktrace    string      `json:"stacktrace,omitempty"`
	Status        IssueStatus `json:"status"`
	EventCount    int64       `json:"event_count"`
	FirstSeen     time.Time   `json:"first_seen"`
	LastSeen      time.Time   `json:"last_seen"`
	ResolvedAt    *time.Time  `json:"resolved_at,omitempty"`
	CreatedAt     time.Time   `json:"created_at"`
}

type IssueDetail struct {
	Issue
	Environments []string `json:"environments"`
	Releases     []string `json:"releases"`
}
