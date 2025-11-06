package database

import (
	"encoding/json"
	"time"

	"github.com/3whiskeywhiskey/metal-enrollment/pkg/models"
	"github.com/google/uuid"
)

// CreateMachineEvent creates a new machine event
func (db *DB) CreateMachineEvent(event *models.MachineEvent) error {
	event.ID = uuid.New().String()
	event.CreatedAt = time.Now()

	query := `
		INSERT INTO machine_events (id, machine_id, event, data, created_at, created_by)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	if db.driver == "sqlite3" {
		query = `
			INSERT INTO machine_events (id, machine_id, event, data, created_at, created_by)
			VALUES (?, ?, ?, ?, ?, ?)
		`
	}

	_, err := db.Exec(query,
		event.ID,
		event.MachineID,
		event.Event,
		event.Data,
		event.CreatedAt,
		event.CreatedBy,
	)

	return err
}

// ListMachineEvents lists events for a machine
func (db *DB) ListMachineEvents(machineID string, limit int) ([]*models.MachineEvent, error) {
	query := `
		SELECT id, machine_id, event, data, created_at, created_by
		FROM machine_events
		WHERE machine_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	if db.driver == "sqlite3" {
		query = `
			SELECT id, machine_id, event, data, created_at, created_by
			FROM machine_events
			WHERE machine_id = ?
			ORDER BY created_at DESC
			LIMIT ?
		`
	}

	rows, err := db.Query(query, machineID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*models.MachineEvent
	for rows.Next() {
		var event models.MachineEvent
		err := rows.Scan(
			&event.ID,
			&event.MachineID,
			&event.Event,
			&event.Data,
			&event.CreatedAt,
			&event.CreatedBy,
		)
		if err != nil {
			return nil, err
		}

		events = append(events, &event)
	}

	return events, nil
}

// ListAllEvents lists all events (for audit purposes)
func (db *DB) ListAllEvents(limit int) ([]*models.MachineEvent, error) {
	query := `
		SELECT id, machine_id, event, data, created_at, created_by
		FROM machine_events
		ORDER BY created_at DESC
		LIMIT $1
	`

	if db.driver == "sqlite3" {
		query = `
			SELECT id, machine_id, event, data, created_at, created_by
			FROM machine_events
			ORDER BY created_at DESC
			LIMIT ?
		`
	}

	rows, err := db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*models.MachineEvent
	for rows.Next() {
		var event models.MachineEvent
		err := rows.Scan(
			&event.ID,
			&event.MachineID,
			&event.Event,
			&event.Data,
			&event.CreatedAt,
			&event.CreatedBy,
		)
		if err != nil {
			return nil, err
		}

		events = append(events, &event)
	}

	return events, nil
}

// EmitMachineEvent is a helper to create an event and trigger webhooks
func (db *DB) EmitMachineEvent(machineID, eventType string, data interface{}, createdBy *string) error {
	dataJSON, err := json.Marshal(data)
	if err != nil {
		return err
	}

	event := &models.MachineEvent{
		MachineID: machineID,
		Event:     eventType,
		Data:      dataJSON,
		CreatedBy: createdBy,
	}

	return db.CreateMachineEvent(event)
}
