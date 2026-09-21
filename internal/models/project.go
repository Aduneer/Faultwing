package models

import (
	"errors"
	"time"
)

var (
	ErrProjectNameTaken = errors.New("project name is already in use")
	ErrProjectNotFound  = errors.New("project not found")
)

type Project struct {
	ID        int64     `json:"id"`
	OwnerID   int64     `json:"owner_id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}
