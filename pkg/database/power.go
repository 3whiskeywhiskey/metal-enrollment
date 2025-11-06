package database

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/3whiskeywhiskey/metal-enrollment/pkg/models"
	"github.com/google/uuid"
)

// CreatePowerOperation creates a new power operation record
func (db *DB) CreatePowerOperation(op *models.PowerOperation) error {
	op.ID = uuid.New().String()
	op.CreatedAt = time.Now()

	query := `
		INSERT INTO power_operations (
			id, machine_id, operation, status, result, error, initiated_by, created_at, completed_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	if db.driver == "postgres" {
		query = `
			INSERT INTO power_operations (
				id, machine_id, operation, status, result, error, initiated_by, created_at, completed_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		`
	}

	_, err := db.Exec(query,
		op.ID,
		op.MachineID,
		op.Operation,
		op.Status,
		op.Result,
		op.Error,
		op.InitiatedBy,
		op.CreatedAt,
		op.CompletedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create power operation: %w", err)
	}

	return nil
}

// UpdatePowerOperation updates a power operation record
func (db *DB) UpdatePowerOperation(op *models.PowerOperation) error {
	query := `
		UPDATE power_operations SET
			status = ?, result = ?, error = ?, completed_at = ?
		WHERE id = ?
	`

	if db.driver == "postgres" {
		query = `
			UPDATE power_operations SET
				status = $1, result = $2, error = $3, completed_at = $4
			WHERE id = $5
		`
	}

	_, err := db.Exec(query,
		op.Status,
		op.Result,
		op.Error,
		op.CompletedAt,
		op.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update power operation: %w", err)
	}

	return nil
}

// GetPowerOperation retrieves a power operation by ID
func (db *DB) GetPowerOperation(id string) (*models.PowerOperation, error) {
	op := &models.PowerOperation{}
	var result, errorMsg sql.NullString
	var completedAt sql.NullTime

	query := `
		SELECT id, machine_id, operation, status, result, error, initiated_by, created_at, completed_at
		FROM power_operations WHERE id = ?
	`

	if db.driver == "postgres" {
		query = `
			SELECT id, machine_id, operation, status, result, error, initiated_by, created_at, completed_at
			FROM power_operations WHERE id = $1
		`
	}

	err := db.QueryRow(query, id).Scan(
		&op.ID,
		&op.MachineID,
		&op.Operation,
		&op.Status,
		&result,
		&errorMsg,
		&op.InitiatedBy,
		&op.CreatedAt,
		&completedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get power operation: %w", err)
	}

	if result.Valid {
		op.Result = result.String
	}
	if errorMsg.Valid {
		op.Error = errorMsg.String
	}
	if completedAt.Valid {
		op.CompletedAt = &completedAt.Time
	}

	return op, nil
}

// ListPowerOperations retrieves power operations for a machine
func (db *DB) ListPowerOperations(machineID string, limit int) ([]*models.PowerOperation, error) {
	query := `
		SELECT id, machine_id, operation, status, result, error, initiated_by, created_at, completed_at
		FROM power_operations
		WHERE machine_id = ?
		ORDER BY created_at DESC
		LIMIT ?
	`

	if db.driver == "postgres" {
		query = `
			SELECT id, machine_id, operation, status, result, error, initiated_by, created_at, completed_at
			FROM power_operations
			WHERE machine_id = $1
			ORDER BY created_at DESC
			LIMIT $2
		`
	}

	rows, err := db.Query(query, machineID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list power operations: %w", err)
	}
	defer rows.Close()

	var operations []*models.PowerOperation
	for rows.Next() {
		op := &models.PowerOperation{}
		var result, errorMsg sql.NullString
		var completedAt sql.NullTime

		err := rows.Scan(
			&op.ID,
			&op.MachineID,
			&op.Operation,
			&op.Status,
			&result,
			&errorMsg,
			&op.InitiatedBy,
			&op.CreatedAt,
			&completedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan power operation: %w", err)
		}

		if result.Valid {
			op.Result = result.String
		}
		if errorMsg.Valid {
			op.Error = errorMsg.String
		}
		if completedAt.Valid {
			op.CompletedAt = &completedAt.Time
		}

		operations = append(operations, op)
	}

	return operations, nil
}
