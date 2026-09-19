package models

import (
	"errors"
	"time"
)

var ErrProjectNameTaken = errors.New("project name is already in use")

type Project struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}
