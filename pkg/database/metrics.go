package database

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/3whiskeywhiskey/metal-enrollment/pkg/models"
	"github.com/google/uuid"
)

// CreateMachineMetrics creates a new metrics record
func (db *DB) CreateMachineMetrics(metrics *models.MachineMetrics) error {
	metrics.ID = uuid.New().String()

	query := `
		INSERT INTO machine_metrics (
			id, machine_id, timestamp, cpu_usage_percent, memory_used_bytes, memory_total_bytes,
			disk_used_bytes, disk_total_bytes, network_rx_bytes, network_tx_bytes,
			load_average_1, load_average_5, load_average_15, temperature, power_state, uptime
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	if db.driver == "postgres" {
		query = `
			INSERT INTO machine_metrics (
				id, machine_id, timestamp, cpu_usage_percent, memory_used_bytes, memory_total_bytes,
				disk_used_bytes, disk_total_bytes, network_rx_bytes, network_tx_bytes,
				load_average_1, load_average_5, load_average_15, temperature, power_state, uptime
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		`
	}

	_, err := db.Exec(query,
		metrics.ID,
		metrics.MachineID,
		metrics.Timestamp,
		metrics.CPUUsagePercent,
		metrics.MemoryUsedBytes,
		metrics.MemoryTotalBytes,
		metrics.DiskUsedBytes,
		metrics.DiskTotalBytes,
		metrics.NetworkRxBytes,
		metrics.NetworkTxBytes,
		metrics.LoadAverage1,
		metrics.LoadAverage5,
		metrics.LoadAverage15,
		metrics.Temperature,
		metrics.PowerState,
		metrics.Uptime,
	)

	if err != nil {
		return fmt.Errorf("failed to create machine metrics: %w", err)
	}

	return nil
}

// GetLatestMetrics retrieves the most recent metrics for a machine
func (db *DB) GetLatestMetrics(machineID string) (*models.MachineMetrics, error) {
	metrics := &models.MachineMetrics{}
	var temperature sql.NullFloat64

	query := `
		SELECT id, machine_id, timestamp, cpu_usage_percent, memory_used_bytes, memory_total_bytes,
		       disk_used_bytes, disk_total_bytes, network_rx_bytes, network_tx_bytes,
		       load_average_1, load_average_5, load_average_15, temperature, power_state, uptime
		FROM machine_metrics
		WHERE machine_id = ?
		ORDER BY timestamp DESC
		LIMIT 1
	`

	if db.driver == "postgres" {
		query = `
			SELECT id, machine_id, timestamp, cpu_usage_percent, memory_used_bytes, memory_total_bytes,
			       disk_used_bytes, disk_total_bytes, network_rx_bytes, network_tx_bytes,
			       load_average_1, load_average_5, load_average_15, temperature, power_state, uptime
			FROM machine_metrics
			WHERE machine_id = $1
			ORDER BY timestamp DESC
			LIMIT 1
		`
	}

	err := db.QueryRow(query, machineID).Scan(
		&metrics.ID,
		&metrics.MachineID,
		&metrics.Timestamp,
		&metrics.CPUUsagePercent,
		&metrics.MemoryUsedBytes,
		&metrics.MemoryTotalBytes,
		&metrics.DiskUsedBytes,
		&metrics.DiskTotalBytes,
		&metrics.NetworkRxBytes,
		&metrics.NetworkTxBytes,
		&metrics.LoadAverage1,
		&metrics.LoadAverage5,
		&metrics.LoadAverage15,
		&temperature,
		&metrics.PowerState,
		&metrics.Uptime,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get latest metrics: %w", err)
	}

	if temperature.Valid {
		temp := temperature.Float64
		metrics.Temperature = &temp
	}

	return metrics, nil
}

// ListMetrics retrieves metrics for a machine within a time range
func (db *DB) ListMetrics(machineID string, since time.Time, limit int) ([]*models.MachineMetrics, error) {
	query := `
		SELECT id, machine_id, timestamp, cpu_usage_percent, memory_used_bytes, memory_total_bytes,
		       disk_used_bytes, disk_total_bytes, network_rx_bytes, network_tx_bytes,
		       load_average_1, load_average_5, load_average_15, temperature, power_state, uptime
		FROM machine_metrics
		WHERE machine_id = ? AND timestamp >= ?
		ORDER BY timestamp DESC
		LIMIT ?
	`

	if db.driver == "postgres" {
		query = `
			SELECT id, machine_id, timestamp, cpu_usage_percent, memory_used_bytes, memory_total_bytes,
			       disk_used_bytes, disk_total_bytes, network_rx_bytes, network_tx_bytes,
			       load_average_1, load_average_5, load_average_15, temperature, power_state, uptime
			FROM machine_metrics
			WHERE machine_id = $1 AND timestamp >= $2
			ORDER BY timestamp DESC
			LIMIT $3
		`
	}

	rows, err := db.Query(query, machineID, since, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list metrics: %w", err)
	}
	defer rows.Close()

	var metricsList []*models.MachineMetrics
	for rows.Next() {
		metrics := &models.MachineMetrics{}
		var temperature sql.NullFloat64

		err := rows.Scan(
			&metrics.ID,
			&metrics.MachineID,
			&metrics.Timestamp,
			&metrics.CPUUsagePercent,
			&metrics.MemoryUsedBytes,
			&metrics.MemoryTotalBytes,
			&metrics.DiskUsedBytes,
			&metrics.DiskTotalBytes,
			&metrics.NetworkRxBytes,
			&metrics.NetworkTxBytes,
			&metrics.LoadAverage1,
			&metrics.LoadAverage5,
			&metrics.LoadAverage15,
			&temperature,
			&metrics.PowerState,
			&metrics.Uptime,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan metrics: %w", err)
		}

		if temperature.Valid {
			temp := temperature.Float64
			metrics.Temperature = &temp
		}

		metricsList = append(metricsList, metrics)
	}

	return metricsList, nil
}

// DeleteOldMetrics removes metrics older than the specified duration
func (db *DB) DeleteOldMetrics(before time.Time) error {
	query := "DELETE FROM machine_metrics WHERE timestamp < ?"
	if db.driver == "postgres" {
		query = "DELETE FROM machine_metrics WHERE timestamp < $1"
	}

	_, err := db.Exec(query, before)
	if err != nil {
		return fmt.Errorf("failed to delete old metrics: %w", err)
	}

	return nil
}
