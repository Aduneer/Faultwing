package database

import (
	"context"
	"errors"

	"github.com/Aduneer/Faultwing/internal/apikey"
	"github.com/Aduneer/Faultwing/internal/models"
	"github.com/jackc/pgx/v5"
)

func (s *Store) AuthenticateProject(ctx context.Context, key string) (models.Project, error) {
	keyHash := apikey.Hash(key)

	var project models.Project
	err := s.pool.QueryRow(ctx, `
		SELECT projects.id, projects.owner_id, projects.name, projects.created_at
		FROM api_keys
		JOIN projects ON projects.id = api_keys.project_id
		WHERE api_keys.key_hash = $1
		  AND api_keys.revoked_at IS NULL
	`, keyHash[:]).Scan(&project.ID, &project.OwnerID, &project.Name, &project.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.Project{}, apikey.ErrInvalid
	}
	if err != nil {
		return models.Project{}, err
	}

	return project, nil
}
