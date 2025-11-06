package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/3whiskeywhiskey/metal-enrollment/pkg/models"
	"github.com/google/uuid"
)

// CreateMachine creates a new machine record
func (db *DB) CreateMachine(req models.EnrollmentRequest) (*models.Machine, error) {
	machine := &models.Machine{
		ID:          uuid.New().String(),
		ServiceTag:  req.ServiceTag,
		MACAddress:  req.MACAddress,
		Status:      models.StatusEnrolled,
		Hardware:    req.Hardware,
		EnrolledAt:  time.Now(),
		UpdatedAt:   time.Now(),
	}

	hardwareJSON, err := json.Marshal(machine.Hardware)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal hardware: %w", err)
	}

	query := `
		INSERT INTO machines (
			id, service_tag, mac_address, status, hardware, enrolled_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	if db.driver == "postgres" {
		query = `
			INSERT INTO machines (
				id, service_tag, mac_address, status, hardware, enrolled_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7)
		`
	}

	_, err = db.Exec(query,
		machine.ID,
		machine.ServiceTag,
		machine.MACAddress,
		machine.Status,
		hardwareJSON,
		machine.EnrolledAt,
		machine.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create machine: %w", err)
	}

	return machine, nil
}

// GetMachine retrieves a machine by ID
func (db *DB) GetMachine(id string) (*models.Machine, error) {
	machine := &models.Machine{}
	var hardwareJSON, bmcJSON []byte
	var hostname, description, nixosConfig sql.NullString
	var lastBuildID sql.NullString
	var lastBuildTime, lastSeenAt sql.NullTime

	query := `
		SELECT id, service_tag, mac_address, status, hostname, description,
		       hardware, nixos_config, last_build_id, last_build_time,
		       enrolled_at, updated_at, last_seen_at, bmc_info
		FROM machines WHERE id = ?
	`

	if db.driver == "postgres" {
		query = `
			SELECT id, service_tag, mac_address, status, hostname, description,
			       hardware, nixos_config, last_build_id, last_build_time,
			       enrolled_at, updated_at, last_seen_at, bmc_info
			FROM machines WHERE id = $1
		`
	}

	err := db.QueryRow(query, id).Scan(
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
		&bmcJSON,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get machine: %w", err)
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

	// Unmarshal BMC info if present
	if len(bmcJSON) > 0 {
		var bmcInfo models.BMCInfo
		if err := json.Unmarshal(bmcJSON, &bmcInfo); err != nil {
			return nil, fmt.Errorf("failed to unmarshal bmc_info: %w", err)
		}
		machine.BMCInfo = &bmcInfo
	}

	return machine, nil
}

// GetMachineByServiceTag retrieves a machine by service tag
func (db *DB) GetMachineByServiceTag(serviceTag string) (*models.Machine, error) {
	machine := &models.Machine{}
	var hardwareJSON, bmcJSON []byte
	var hostname, description, nixosConfig sql.NullString
	var lastBuildID sql.NullString
	var lastBuildTime, lastSeenAt sql.NullTime

	query := `
		SELECT id, service_tag, mac_address, status, hostname, description,
		       hardware, nixos_config, last_build_id, last_build_time,
		       enrolled_at, updated_at, last_seen_at, bmc_info
		FROM machines WHERE service_tag = ?
	`

	if db.driver == "postgres" {
		query = `
			SELECT id, service_tag, mac_address, status, hostname, description,
			       hardware, nixos_config, last_build_id, last_build_time,
			       enrolled_at, updated_at, last_seen_at, bmc_info
			FROM machines WHERE service_tag = $1
		`
	}

	err := db.QueryRow(query, serviceTag).Scan(
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
		&bmcJSON,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get machine: %w", err)
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

	// Unmarshal BMC info if present
	if len(bmcJSON) > 0 {
		var bmcInfo models.BMCInfo
		if err := json.Unmarshal(bmcJSON, &bmcInfo); err != nil {
			return nil, fmt.Errorf("failed to unmarshal bmc_info: %w", err)
		}
		machine.BMCInfo = &bmcInfo
	}

	return machine, nil
}

// ListMachines retrieves all machines
func (db *DB) ListMachines() ([]*models.Machine, error) {
	query := `
		SELECT id, service_tag, mac_address, status, hostname, description,
		       hardware, nixos_config, last_build_id, last_build_time,
		       enrolled_at, updated_at, last_seen_at, bmc_info
		FROM machines
		ORDER BY enrolled_at DESC
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list machines: %w", err)
	}
	defer rows.Close()

	var machines []*models.Machine
	for rows.Next() {
		machine := &models.Machine{}
		var hardwareJSON, bmcJSON []byte
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
			&bmcJSON,
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

		// Unmarshal BMC info if present
		if len(bmcJSON) > 0 {
			var bmcInfo models.BMCInfo
			if err := json.Unmarshal(bmcJSON, &bmcInfo); err != nil {
				return nil, fmt.Errorf("failed to unmarshal bmc_info: %w", err)
			}
			machine.BMCInfo = &bmcInfo
		}

		machines = append(machines, machine)
	}

	return machines, nil
}

// UpdateMachine updates a machine record
func (db *DB) UpdateMachine(machine *models.Machine) error {
	machine.UpdatedAt = time.Now()

	hardwareJSON, err := json.Marshal(machine.Hardware)
	if err != nil {
		return fmt.Errorf("failed to marshal hardware: %w", err)
	}

	var bmcJSON []byte
	if machine.BMCInfo != nil {
		bmcJSON, err = json.Marshal(machine.BMCInfo)
		if err != nil {
			return fmt.Errorf("failed to marshal bmc_info: %w", err)
		}
	}

	query := `
		UPDATE machines SET
			hostname = ?, description = ?, hardware = ?, nixos_config = ?,
			status = ?, last_build_id = ?, last_build_time = ?, updated_at = ?,
			last_seen_at = ?, bmc_info = ?
		WHERE id = ?
	`

	if db.driver == "postgres" {
		query = `
			UPDATE machines SET
				hostname = $1, description = $2, hardware = $3, nixos_config = $4,
				status = $5, last_build_id = $6, last_build_time = $7, updated_at = $8,
				last_seen_at = $9, bmc_info = $10
			WHERE id = $11
		`
	}

	_, err = db.Exec(query,
		machine.Hostname,
		machine.Description,
		hardwareJSON,
		machine.NixOSConfig,
		machine.Status,
		machine.LastBuildID,
		machine.LastBuildTime,
		machine.UpdatedAt,
		machine.LastSeenAt,
		bmcJSON,
		machine.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update machine: %w", err)
	}

	return nil
}

// DeleteMachine deletes a machine record
func (db *DB) DeleteMachine(id string) error {
	query := "DELETE FROM machines WHERE id = ?"
	if db.driver == "postgres" {
		query = "DELETE FROM machines WHERE id = $1"
	}

	_, err := db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete machine: %w", err)
	}

	return nil
}
