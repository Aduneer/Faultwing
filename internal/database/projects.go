package database

import (
	"context"
	"errors"
	"fmt"

	"github.com/Aduneer/FlyTrap/internal/apikey"
	"github.com/Aduneer/FlyTrap/internal/models"
	"github.com/jackc/pgx/v5/pgconn"
)

func (s *Store) CreateProject(ctx context.Context, name string) (models.Project, string, error) {
	key, err := apikey.Generate()
	if err != nil {
		return models.Project{}, "", fmt.Errorf("generate API key: %w", err)
	}
	keyHash := apikey.Hash(key)

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return models.Project{}, "", err
	}
	defer tx.Rollback(ctx)

	var project models.Project
	err = tx.QueryRow(ctx, `
		INSERT INTO projects (name)
		VALUES ($1)
		RETURNING id, name, created_at
	`, name).Scan(&project.ID, &project.Name, &project.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return models.Project{}, "", models.ErrProjectNameTaken
		}
		return models.Project{}, "", err
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO api_keys (project_id, key_prefix, key_hash)
		VALUES ($1, $2, $3)
	`, project.ID, apikey.DisplayPrefix(key), keyHash[:]); err != nil {
		return models.Project{}, "", err
	}

	if err := tx.Commit(ctx); err != nil {
		return models.Project{}, "", err
	}

	return project, key, nil
}
