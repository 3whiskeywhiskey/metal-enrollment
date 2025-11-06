package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/3whiskeywhiskey/metal-enrollment/pkg/models"
	"github.com/google/uuid"
)

// CreateGroup creates a new machine group
func (db *DB) CreateGroup(name, description string, tags []string) (*models.MachineGroup, error) {
	group := &models.MachineGroup{
		ID:          uuid.New().String(),
		Name:        name,
		Description: description,
		Tags:        tags,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	tagsJSON, err := json.Marshal(group.Tags)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal tags: %w", err)
	}

	query := `
		INSERT INTO groups (id, name, description, tags, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	if db.driver == "postgres" {
		query = `
			INSERT INTO groups (id, name, description, tags, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6)
		`
	}

	_, err = db.Exec(query,
		group.ID,
		group.Name,
		group.Description,
		tagsJSON,
		group.CreatedAt,
		group.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create group: %w", err)
	}

	return group, nil
}

// GetGroup retrieves a group by ID
func (db *DB) GetGroup(id string) (*models.MachineGroup, error) {
	group := &models.MachineGroup{}
	var tagsJSON []byte
	var description sql.NullString

	query := `
		SELECT id, name, description, tags, created_at, updated_at
		FROM groups WHERE id = ?
	`

	if db.driver == "postgres" {
		query = `
			SELECT id, name, description, tags, created_at, updated_at
			FROM groups WHERE id = $1
		`
	}

	err := db.QueryRow(query, id).Scan(
		&group.ID,
		&group.Name,
		&description,
		&tagsJSON,
		&group.CreatedAt,
		&group.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get group: %w", err)
	}

	if description.Valid {
		group.Description = description.String
	}

	if tagsJSON != nil {
		if err := json.Unmarshal(tagsJSON, &group.Tags); err != nil {
			return nil, fmt.Errorf("failed to unmarshal tags: %w", err)
		}
	}

	return group, nil
}

// GetGroupByName retrieves a group by name
func (db *DB) GetGroupByName(name string) (*models.MachineGroup, error) {
	group := &models.MachineGroup{}
	var tagsJSON []byte
	var description sql.NullString

	query := `
		SELECT id, name, description, tags, created_at, updated_at
		FROM groups WHERE name = ?
	`

	if db.driver == "postgres" {
		query = `
			SELECT id, name, description, tags, created_at, updated_at
			FROM groups WHERE name = $1
		`
	}

	err := db.QueryRow(query, name).Scan(
		&group.ID,
		&group.Name,
		&description,
		&tagsJSON,
		&group.CreatedAt,
		&group.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get group: %w", err)
	}

	if description.Valid {
		group.Description = description.String
	}

	if tagsJSON != nil {
		if err := json.Unmarshal(tagsJSON, &group.Tags); err != nil {
			return nil, fmt.Errorf("failed to unmarshal tags: %w", err)
		}
	}

	return group, nil
}

// ListGroups retrieves all groups
func (db *DB) ListGroups() ([]*models.MachineGroup, error) {
	query := `
		SELECT id, name, description, tags, created_at, updated_at
		FROM groups
		ORDER BY name ASC
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list groups: %w", err)
	}
	defer rows.Close()

	var groups []*models.MachineGroup
	for rows.Next() {
		group := &models.MachineGroup{}
		var tagsJSON []byte
		var description sql.NullString

		err := rows.Scan(
			&group.ID,
			&group.Name,
			&description,
			&tagsJSON,
			&group.CreatedAt,
			&group.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan group: %w", err)
		}

		if description.Valid {
			group.Description = description.String
		}

		if tagsJSON != nil {
			if err := json.Unmarshal(tagsJSON, &group.Tags); err != nil {
				return nil, fmt.Errorf("failed to unmarshal tags: %w", err)
			}
		}

		groups = append(groups, group)
	}

	return groups, nil
}

// UpdateGroup updates a group record
func (db *DB) UpdateGroup(group *models.MachineGroup) error {
	group.UpdatedAt = time.Now()

	tagsJSON, err := json.Marshal(group.Tags)
	if err != nil {
		return fmt.Errorf("failed to marshal tags: %w", err)
	}

	query := `
		UPDATE groups SET
			name = ?, description = ?, tags = ?, updated_at = ?
		WHERE id = ?
	`

	if db.driver == "postgres" {
		query = `
			UPDATE groups SET
				name = $1, description = $2, tags = $3, updated_at = $4
			WHERE id = $5
		`
	}

	_, err = db.Exec(query,
		group.Name,
		group.Description,
		tagsJSON,
		group.UpdatedAt,
		group.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update group: %w", err)
	}

	return nil
}

// DeleteGroup deletes a group and its memberships
func (db *DB) DeleteGroup(id string) error {
	query := "DELETE FROM groups WHERE id = ?"
	if db.driver == "postgres" {
		query = "DELETE FROM groups WHERE id = $1"
	}

	_, err := db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete group: %w", err)
	}

	return nil
}

// AddMachineToGroup adds a machine to a group
func (db *DB) AddMachineToGroup(groupID, machineID string) error {
	query := `
		INSERT INTO group_memberships (group_id, machine_id, added_at)
		VALUES (?, ?, ?)
		ON CONFLICT DO NOTHING
	`

	if db.driver == "postgres" {
		query = `
			INSERT INTO group_memberships (group_id, machine_id, added_at)
			VALUES ($1, $2, $3)
			ON CONFLICT DO NOTHING
		`
	}

	_, err := db.Exec(query, groupID, machineID, time.Now())
	if err != nil {
		return fmt.Errorf("failed to add machine to group: %w", err)
	}

	return nil
}

// RemoveMachineFromGroup removes a machine from a group
func (db *DB) RemoveMachineFromGroup(groupID, machineID string) error {
	query := "DELETE FROM group_memberships WHERE group_id = ? AND machine_id = ?"
	if db.driver == "postgres" {
		query = "DELETE FROM group_memberships WHERE group_id = $1 AND machine_id = $2"
	}

	_, err := db.Exec(query, groupID, machineID)
	if err != nil {
		return fmt.Errorf("failed to remove machine from group: %w", err)
	}

	return nil
}

// GetGroupMachines retrieves all machines in a group
func (db *DB) GetGroupMachines(groupID string) ([]*models.Machine, error) {
	query := `
		SELECT m.id, m.service_tag, m.mac_address, m.status, m.hostname, m.description,
		       m.hardware, m.nixos_config, m.last_build_id, m.last_build_time,
		       m.enrolled_at, m.updated_at, m.last_seen_at
		FROM machines m
		INNER JOIN group_memberships gm ON m.id = gm.machine_id
		WHERE gm.group_id = ?
		ORDER BY m.hostname ASC
	`

	if db.driver == "postgres" {
		query = `
			SELECT m.id, m.service_tag, m.mac_address, m.status, m.hostname, m.description,
			       m.hardware, m.nixos_config, m.last_build_id, m.last_build_time,
			       m.enrolled_at, m.updated_at, m.last_seen_at
			FROM machines m
			INNER JOIN group_memberships gm ON m.id = gm.machine_id
			WHERE gm.group_id = $1
			ORDER BY m.hostname ASC
		`
	}

	rows, err := db.Query(query, groupID)
	if err != nil {
		return nil, fmt.Errorf("failed to get group machines: %w", err)
	}
	defer rows.Close()

	var machines []*models.Machine
	for rows.Next() {
		machine := &models.Machine{}
		var hardwareJSON []byte
		var hostname, description, nixosConfig sql.NullString
		var lastBuildID sql.NullString
		var lastBuildTime, lastSeenAt sql.NullTime

		err := rows.Scan(
			&machine.ID,
			&machine.ServiceTag,
			&machine.MACAddress,
			&machine.Status,
			&hostname,
			&description,
			&hardwareJSON,
			&nixosConfig,
			&lastBuildID,
			&lastBuildTime,
			&machine.EnrolledAt,
			&machine.UpdatedAt,
			&lastSeenAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan machine: %w", err)
		}

		// Convert nullable fields
		if hostname.Valid {
			machine.Hostname = hostname.String
		}
		if description.Valid {
			machine.Description = description.String
		}
		if nixosConfig.Valid {
			machine.NixOSConfig = nixosConfig.String
		}
		if lastBuildID.Valid {
			id := lastBuildID.String
			machine.LastBuildID = &id
		}
		if lastBuildTime.Valid {
			machine.LastBuildTime = &lastBuildTime.Time
		}
		if lastSeenAt.Valid {
			machine.LastSeenAt = &lastSeenAt.Time
		}

		if err := json.Unmarshal(hardwareJSON, &machine.Hardware); err != nil {
			return nil, fmt.Errorf("failed to unmarshal hardware: %w", err)
		}

		machines = append(machines, machine)
	}

	return machines, nil
}

// GetMachineGroups retrieves all groups a machine belongs to
func (db *DB) GetMachineGroups(machineID string) ([]*models.MachineGroup, error) {
	query := `
		SELECT g.id, g.name, g.description, g.tags, g.created_at, g.updated_at
		FROM groups g
		INNER JOIN group_memberships gm ON g.id = gm.group_id
		WHERE gm.machine_id = ?
		ORDER BY g.name ASC
	`

	if db.driver == "postgres" {
		query = `
			SELECT g.id, g.name, g.description, g.tags, g.created_at, g.updated_at
			FROM groups g
			INNER JOIN group_memberships gm ON g.id = gm.group_id
			WHERE gm.machine_id = $1
			ORDER BY g.name ASC
		`
	}

	rows, err := db.Query(query, machineID)
	if err != nil {
		return nil, fmt.Errorf("failed to get machine groups: %w", err)
	}
	defer rows.Close()

	var groups []*models.MachineGroup
	for rows.Next() {
		group := &models.MachineGroup{}
		var tagsJSON []byte
		var description sql.NullString

		err := rows.Scan(
			&group.ID,
			&group.Name,
			&description,
			&tagsJSON,
			&group.CreatedAt,
			&group.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan group: %w", err)
		}

		if description.Valid {
			group.Description = description.String
		}

		if tagsJSON != nil {
			if err := json.Unmarshal(tagsJSON, &group.Tags); err != nil {
				return nil, fmt.Errorf("failed to unmarshal tags: %w", err)
			}
		}

		groups = append(groups, group)
	}

	return groups, nil
}
