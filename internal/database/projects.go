package database

import (
	"context"
	"errors"
	"fmt"

	"github.com/Aduneer/Faultwing/internal/apikey"
	"github.com/Aduneer/Faultwing/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func (s *Store) CreateProject(ctx context.Context, ownerID int64, name string) (models.Project, string, error) {
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
		INSERT INTO projects (owner_id, name)
		VALUES ($1, $2)
		RETURNING id, owner_id, name, created_at
	`, ownerID, name).Scan(&project.ID, &project.OwnerID, &project.Name, &project.CreatedAt)
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

func (s *Store) ListProjects(ctx context.Context, ownerID int64) ([]models.Project, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, owner_id, name, created_at
		FROM projects
		WHERE owner_id = $1
		ORDER BY created_at DESC, id DESC
	`, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	projects := make([]models.Project, 0)
	for rows.Next() {
		var project models.Project
		if err := rows.Scan(&project.ID, &project.OwnerID, &project.Name, &project.CreatedAt); err != nil {
			return nil, err
		}
		projects = append(projects, project)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return projects, nil
}

func (s *Store) GetProjectForOwner(ctx context.Context, ownerID, projectID int64) (models.Project, error) {
	var project models.Project
	err := s.pool.QueryRow(ctx, `
		SELECT id, owner_id, name, created_at
		FROM projects
		WHERE owner_id = $1 AND id = $2
	`, ownerID, projectID).Scan(
		&project.ID,
		&project.OwnerID,
		&project.Name,
		&project.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.Project{}, models.ErrProjectNotFound
	}
	if err != nil {
		return models.Project{}, err
	}

	return project, nil
}
