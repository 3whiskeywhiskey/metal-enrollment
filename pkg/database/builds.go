package database

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/3whiskeywhiskey/metal-enrollment/pkg/models"
	"github.com/google/uuid"
)

// CreateBuild creates a new build request
func (db *DB) CreateBuild(machineID, config string) (*models.BuildRequest, error) {
	build := &models.BuildRequest{
		ID:        uuid.New().String(),
		MachineID: machineID,
		Status:    "pending",
		Config:    config,
		CreatedAt: time.Now(),
	}

	query := `
		INSERT INTO builds (id, machine_id, status, config, created_at)
		VALUES (?, ?, ?, ?, ?)
	`

	if db.driver == "postgres" {
		query = `
			INSERT INTO builds (id, machine_id, status, config, created_at)
			VALUES ($1, $2, $3, $4, $5)
		`
	}

	_, err := db.Exec(query,
		build.ID,
		build.MachineID,
		build.Status,
		build.Config,
		build.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create build: %w", err)
	}

	return build, nil
}

// GetBuild retrieves a build by ID
func (db *DB) GetBuild(id string) (*models.BuildRequest, error) {
	build := &models.BuildRequest{}

	query := `
		SELECT id, machine_id, status, config, log_output, error, artifact_url,
		       created_at, completed_at
		FROM builds WHERE id = ?
	`

	if db.driver == "postgres" {
		query = `
			SELECT id, machine_id, status, config, log_output, error, artifact_url,
			       created_at, completed_at
			FROM builds WHERE id = $1
		`
	}

	err := db.QueryRow(query, id).Scan(
		&build.ID,
		&build.MachineID,
		&build.Status,
		&build.Config,
		&build.LogOutput,
		&build.Error,
		&build.ArtifactURL,
		&build.CreatedAt,
		&build.CompletedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get build: %w", err)
	}

	return build, nil
}

// ListBuildsByMachine retrieves all builds for a machine
func (db *DB) ListBuildsByMachine(machineID string) ([]*models.BuildRequest, error) {
	query := `
		SELECT id, machine_id, status, config, log_output, error, artifact_url,
		       created_at, completed_at
		FROM builds
		WHERE machine_id = ?
		ORDER BY created_at DESC
	`

	if db.driver == "postgres" {
		query = `
			SELECT id, machine_id, status, config, log_output, error, artifact_url,
			       created_at, completed_at
			FROM builds
			WHERE machine_id = $1
			ORDER BY created_at DESC
		`
	}

	rows, err := db.Query(query, machineID)
	if err != nil {
		return nil, fmt.Errorf("failed to list builds: %w", err)
	}
	defer rows.Close()

	var builds []*models.BuildRequest
	for rows.Next() {
		build := &models.BuildRequest{}
		err := rows.Scan(
			&build.ID,
			&build.MachineID,
			&build.Status,
			&build.Config,
			&build.LogOutput,
			&build.Error,
			&build.ArtifactURL,
			&build.CreatedAt,
			&build.CompletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan build: %w", err)
		}
		builds = append(builds, build)
	}

	return builds, nil
}

// UpdateBuild updates a build record
func (db *DB) UpdateBuild(build *models.BuildRequest) error {
	query := `
		UPDATE builds SET
			status = ?, log_output = ?, error = ?, artifact_url = ?, completed_at = ?
		WHERE id = ?
	`

	if db.driver == "postgres" {
		query = `
			UPDATE builds SET
				status = $1, log_output = $2, error = $3, artifact_url = $4, completed_at = $5
			WHERE id = $6
		`
	}

	_, err := db.Exec(query,
		build.Status,
		build.LogOutput,
		build.Error,
		build.ArtifactURL,
		build.CompletedAt,
		build.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update build: %w", err)
	}

	return nil
}
