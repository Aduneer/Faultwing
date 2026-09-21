package models

import (
	"errors"
	"time"
)

var (
	ErrEmailTaken     = errors.New("email is already registered")
	ErrUserNotFound   = errors.New("user not found")
	ErrInvalidSession = errors.New("invalid or expired session")
)

type User struct {
	ID           int64     `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

type Session struct {
	ID        int64      `json:"id"`
	UserID    int64      `json:"user_id"`
	ExpiresAt time.Time  `json:"expires_at"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}
