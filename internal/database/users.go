package database

import (
	"context"
	"errors"
	"time"

	"github.com/Aduneer/FlyTrap/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func (s *Store) CreateUser(ctx context.Context, email, passwordHash string) (models.User, error) {
	var user models.User
	err := s.pool.QueryRow(ctx, `
		INSERT INTO users (email, password_hash)
		VALUES ($1, $2)
		RETURNING id, email, password_hash, created_at
	`, email, passwordHash).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.CreatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return models.User{}, models.ErrEmailTaken
		}
		return models.User{}, err
	}

	return user, nil
}

func (s *Store) FindUserByEmail(ctx context.Context, email string) (models.User, error) {
	var user models.User
	err := s.pool.QueryRow(ctx, `
		SELECT id, email, password_hash, created_at
		FROM users
		WHERE LOWER(email) = LOWER($1)
	`, email).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.User{}, models.ErrUserNotFound
	}
	if err != nil {
		return models.User{}, err
	}

	return user, nil
}

func (s *Store) CreateSession(
	ctx context.Context,
	userID int64,
	tokenHash [32]byte,
	expiresAt time.Time,
) (models.Session, error) {
	var session models.Session
	err := s.pool.QueryRow(ctx, `
		INSERT INTO sessions (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
		RETURNING id, user_id, expires_at, revoked_at, created_at
	`, userID, tokenHash[:], expiresAt).Scan(
		&session.ID,
		&session.UserID,
		&session.ExpiresAt,
		&session.RevokedAt,
		&session.CreatedAt,
	)
	if err != nil {
		return models.Session{}, err
	}

	return session, nil
}

func (s *Store) AuthenticateSession(
	ctx context.Context,
	tokenHash [32]byte,
) (models.User, models.Session, error) {
	var user models.User
	var session models.Session
	err := s.pool.QueryRow(ctx, `
		SELECT users.id, users.email, users.created_at,
			sessions.id, sessions.user_id, sessions.expires_at,
			sessions.revoked_at, sessions.created_at
		FROM sessions
		JOIN users ON users.id = sessions.user_id
		WHERE sessions.token_hash = $1
		  AND sessions.revoked_at IS NULL
		  AND sessions.expires_at > NOW()
	`, tokenHash[:]).Scan(
		&user.ID,
		&user.Email,
		&user.CreatedAt,
		&session.ID,
		&session.UserID,
		&session.ExpiresAt,
		&session.RevokedAt,
		&session.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.User{}, models.Session{}, models.ErrInvalidSession
	}
	if err != nil {
		return models.User{}, models.Session{}, err
	}

	return user, session, nil
}

func (s *Store) RevokeSession(ctx context.Context, tokenHash [32]byte) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE sessions
		SET revoked_at = NOW()
		WHERE token_hash = $1
		  AND revoked_at IS NULL
	`, tokenHash[:])
	return err
}
