package database

import (
	"database/sql"
	"time"

	"github.com/3whiskeywhiskey/metal-enrollment/pkg/models"
	"github.com/google/uuid"
)

// CreateTemplate creates a new machine template
func (db *DB) CreateTemplate(template *models.MachineTemplate) error {
	template.ID = uuid.New().String()
	template.CreatedAt = time.Now()
	template.UpdatedAt = time.Now()

	query := `
		INSERT INTO machine_templates (id, name, description, nixos_config, bmc_config, tags, variables, created_at, updated_at, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	if db.driver == "sqlite3" {
		query = `
			INSERT INTO machine_templates (id, name, description, nixos_config, bmc_config, tags, variables, created_at, updated_at, created_by)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`
	}

	bmcConfigJSON, err := template.BMCConfig.Value()
	if err != nil {
		return err
	}

	_, err = db.Exec(query,
		template.ID,
		template.Name,
		template.Description,
		template.NixOSConfig,
		bmcConfigJSON,
		template.Tags,
		template.Variables,
		template.CreatedAt,
		template.UpdatedAt,
		template.CreatedBy,
	)

	return err
}

// GetTemplate retrieves a template by ID
func (db *DB) GetTemplate(id string) (*models.MachineTemplate, error) {
	var template models.MachineTemplate

	query := `
		SELECT id, name, description, nixos_config, bmc_config, tags, variables, created_at, updated_at, created_by
		FROM machine_templates
		WHERE id = $1
	`

	if db.driver == "sqlite3" {
		query = `
			SELECT id, name, description, nixos_config, bmc_config, tags, variables, created_at, updated_at, created_by
			FROM machine_templates
			WHERE id = ?
		`
	}

	err := db.QueryRow(query, id).Scan(
		&template.ID,
		&template.Name,
		&template.Description,
		&template.NixOSConfig,
		&template.BMCConfig,
		&template.Tags,
		&template.Variables,
		&template.CreatedAt,
		&template.UpdatedAt,
		&template.CreatedBy,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &template, nil
}

// GetTemplateByName retrieves a template by name
func (db *DB) GetTemplateByName(name string) (*models.MachineTemplate, error) {
	var template models.MachineTemplate

	query := `
		SELECT id, name, description, nixos_config, bmc_config, tags, variables, created_at, updated_at, created_by
		FROM machine_templates
		WHERE name = $1
	`

	if db.driver == "sqlite3" {
		query = `
			SELECT id, name, description, nixos_config, bmc_config, tags, variables, created_at, updated_at, created_by
			FROM machine_templates
			WHERE name = ?
		`
	}

	err := db.QueryRow(query, name).Scan(
		&template.ID,
		&template.Name,
		&template.Description,
		&template.NixOSConfig,
		&template.BMCConfig,
		&template.Tags,
		&template.Variables,
		&template.CreatedAt,
		&template.UpdatedAt,
		&template.CreatedBy,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &template, nil
}

// ListTemplates lists all templates
func (db *DB) ListTemplates() ([]*models.MachineTemplate, error) {
	query := `
		SELECT id, name, description, nixos_config, bmc_config, tags, variables, created_at, updated_at, created_by
		FROM machine_templates
		ORDER BY name ASC
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var templates []*models.MachineTemplate
	for rows.Next() {
		var template models.MachineTemplate
		err := rows.Scan(
			&template.ID,
			&template.Name,
			&template.Description,
			&template.NixOSConfig,
			&template.BMCConfig,
			&template.Tags,
			&template.Variables,
			&template.CreatedAt,
			&template.UpdatedAt,
			&template.CreatedBy,
		)
		if err != nil {
			return nil, err
		}

		templates = append(templates, &template)
	}

	return templates, nil
}

// UpdateTemplate updates a template
func (db *DB) UpdateTemplate(template *models.MachineTemplate) error {
	template.UpdatedAt = time.Now()

	query := `
		UPDATE machine_templates
		SET name = $1, description = $2, nixos_config = $3, bmc_config = $4,
		    tags = $5, variables = $6, updated_at = $7
		WHERE id = $8
	`

	if db.driver == "sqlite3" {
		query = `
			UPDATE machine_templates
			SET name = ?, description = ?, nixos_config = ?, bmc_config = ?,
			    tags = ?, variables = ?, updated_at = ?
			WHERE id = ?
		`
	}

	bmcConfigJSON, err := template.BMCConfig.Value()
	if err != nil {
		return err
	}

	_, err = db.Exec(query,
		template.Name,
		template.Description,
		template.NixOSConfig,
		bmcConfigJSON,
		template.Tags,
		template.Variables,
		template.UpdatedAt,
		template.ID,
	)

	return err
}

// DeleteTemplate deletes a template
func (db *DB) DeleteTemplate(id string) error {
	query := `DELETE FROM machine_templates WHERE id = $1`
	if db.driver == "sqlite3" {
		query = `DELETE FROM machine_templates WHERE id = ?`
	}

	_, err := db.Exec(query, id)
	return err
}
