package database

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/3whiskeywhiskey/metal-enrollment/pkg/models"
	"github.com/google/uuid"
)

// CreateWebhook creates a new webhook
func (db *DB) CreateWebhook(webhook *models.Webhook) error {
	webhook.ID = uuid.New().String()
	webhook.CreatedAt = time.Now()
	webhook.UpdatedAt = time.Now()

	eventsJSON, err := json.Marshal(webhook.Events)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO webhooks (id, name, url, events, secret, active, headers, timeout, max_retries, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	if db.driver == "sqlite3" {
		query = `
			INSERT INTO webhooks (id, name, url, events, secret, active, headers, timeout, max_retries, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`
	}

	_, err = db.Exec(query,
		webhook.ID,
		webhook.Name,
		webhook.URL,
		string(eventsJSON),
		webhook.Secret,
		webhook.Active,
		webhook.Headers,
		webhook.Timeout,
		webhook.MaxRetries,
		webhook.CreatedAt,
		webhook.UpdatedAt,
	)

	return err
}

// GetWebhook retrieves a webhook by ID
func (db *DB) GetWebhook(id string) (*models.Webhook, error) {
	var webhook models.Webhook
	var eventsJSON string

	query := `
		SELECT id, name, url, events, secret, active, headers, timeout, max_retries,
		       last_success, last_failure, created_at, updated_at
		FROM webhooks
		WHERE id = $1
	`

	if db.driver == "sqlite3" {
		query = `
			SELECT id, name, url, events, secret, active, headers, timeout, max_retries,
			       last_success, last_failure, created_at, updated_at
			FROM webhooks
			WHERE id = ?
		`
	}

	err := db.QueryRow(query, id).Scan(
		&webhook.ID,
		&webhook.Name,
		&webhook.URL,
		&eventsJSON,
		&webhook.Secret,
		&webhook.Active,
		&webhook.Headers,
		&webhook.Timeout,
		&webhook.MaxRetries,
		&webhook.LastSuccess,
		&webhook.LastFailure,
		&webhook.CreatedAt,
		&webhook.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal([]byte(eventsJSON), &webhook.Events); err != nil {
		return nil, err
	}

	return &webhook, nil
}

// ListWebhooks lists all webhooks
func (db *DB) ListWebhooks() ([]*models.Webhook, error) {
	query := `
		SELECT id, name, url, events, secret, active, headers, timeout, max_retries,
		       last_success, last_failure, created_at, updated_at
		FROM webhooks
		ORDER BY created_at DESC
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var webhooks []*models.Webhook
	for rows.Next() {
		var webhook models.Webhook
		var eventsJSON string

		err := rows.Scan(
			&webhook.ID,
			&webhook.Name,
			&webhook.URL,
			&eventsJSON,
			&webhook.Secret,
			&webhook.Active,
			&webhook.Headers,
			&webhook.Timeout,
			&webhook.MaxRetries,
			&webhook.LastSuccess,
			&webhook.LastFailure,
			&webhook.CreatedAt,
			&webhook.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		if err := json.Unmarshal([]byte(eventsJSON), &webhook.Events); err != nil {
			return nil, err
		}

		webhooks = append(webhooks, &webhook)
	}

	return webhooks, nil
}

// UpdateWebhook updates a webhook
func (db *DB) UpdateWebhook(webhook *models.Webhook) error {
	webhook.UpdatedAt = time.Now()

	eventsJSON, err := json.Marshal(webhook.Events)
	if err != nil {
		return err
	}

	query := `
		UPDATE webhooks
		SET name = $1, url = $2, events = $3, secret = $4, active = $5,
		    headers = $6, timeout = $7, max_retries = $8, updated_at = $9
		WHERE id = $10
	`

	if db.driver == "sqlite3" {
		query = `
			UPDATE webhooks
			SET name = ?, url = ?, events = ?, secret = ?, active = ?,
			    headers = ?, timeout = ?, max_retries = ?, updated_at = ?
			WHERE id = ?
		`
	}

	_, err = db.Exec(query,
		webhook.Name,
		webhook.URL,
		string(eventsJSON),
		webhook.Secret,
		webhook.Active,
		webhook.Headers,
		webhook.Timeout,
		webhook.MaxRetries,
		webhook.UpdatedAt,
		webhook.ID,
	)

	return err
}

// DeleteWebhook deletes a webhook
func (db *DB) DeleteWebhook(id string) error {
	query := `DELETE FROM webhooks WHERE id = $1`
	if db.driver == "sqlite3" {
		query = `DELETE FROM webhooks WHERE id = ?`
	}

	_, err := db.Exec(query, id)
	return err
}

// GetWebhooksByEvent retrieves all active webhooks for a specific event
func (db *DB) GetWebhooksByEvent(event string) ([]*models.Webhook, error) {
	query := `
		SELECT id, name, url, events, secret, active, headers, timeout, max_retries,
		       last_success, last_failure, created_at, updated_at
		FROM webhooks
		WHERE active = true
	`

	if db.driver == "sqlite3" {
		query += ` AND json_array_length(events) > 0`
	}

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var webhooks []*models.Webhook
	for rows.Next() {
		var webhook models.Webhook
		var eventsJSON string

		err := rows.Scan(
			&webhook.ID,
			&webhook.Name,
			&webhook.URL,
			&eventsJSON,
			&webhook.Secret,
			&webhook.Active,
			&webhook.Headers,
			&webhook.Timeout,
			&webhook.MaxRetries,
			&webhook.LastSuccess,
			&webhook.LastFailure,
			&webhook.CreatedAt,
			&webhook.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		if err := json.Unmarshal([]byte(eventsJSON), &webhook.Events); err != nil {
			return nil, err
		}

		// Filter by event
		for _, e := range webhook.Events {
			if e == event || e == "*" {
				webhooks = append(webhooks, &webhook)
				break
			}
		}
	}

	return webhooks, nil
}

// CreateWebhookDelivery creates a new webhook delivery record
func (db *DB) CreateWebhookDelivery(delivery *models.WebhookDelivery) error {
	delivery.ID = uuid.New().String()
	delivery.CreatedAt = time.Now()

	query := `
		INSERT INTO webhook_deliveries (id, webhook_id, event, payload, status_code, response, error, attempts, success, created_at, completed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	if db.driver == "sqlite3" {
		query = `
			INSERT INTO webhook_deliveries (id, webhook_id, event, payload, status_code, response, error, attempts, success, created_at, completed_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`
	}

	_, err := db.Exec(query,
		delivery.ID,
		delivery.WebhookID,
		delivery.Event,
		delivery.Payload,
		delivery.StatusCode,
		delivery.Response,
		delivery.Error,
		delivery.Attempts,
		delivery.Success,
		delivery.CreatedAt,
		delivery.CompletedAt,
	)

	return err
}

// ListWebhookDeliveries lists deliveries for a webhook
func (db *DB) ListWebhookDeliveries(webhookID string, limit int) ([]*models.WebhookDelivery, error) {
	query := `
		SELECT id, webhook_id, event, payload, status_code, response, error, attempts, success, created_at, completed_at
		FROM webhook_deliveries
		WHERE webhook_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	if db.driver == "sqlite3" {
		query = `
			SELECT id, webhook_id, event, payload, status_code, response, error, attempts, success, created_at, completed_at
			FROM webhook_deliveries
			WHERE webhook_id = ?
			ORDER BY created_at DESC
			LIMIT ?
		`
	}

	rows, err := db.Query(query, webhookID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var deliveries []*models.WebhookDelivery
	for rows.Next() {
		var delivery models.WebhookDelivery
		err := rows.Scan(
			&delivery.ID,
			&delivery.WebhookID,
			&delivery.Event,
			&delivery.Payload,
			&delivery.StatusCode,
			&delivery.Response,
			&delivery.Error,
			&delivery.Attempts,
			&delivery.Success,
			&delivery.CreatedAt,
			&delivery.CompletedAt,
		)
		if err != nil {
			return nil, err
		}

		deliveries = append(deliveries, &delivery)
	}

	return deliveries, nil
}

// UpdateWebhookDeliveryStatus updates the webhook last success/failure timestamps
func (db *DB) UpdateWebhookDeliveryStatus(webhookID string, success bool) error {
	now := time.Now()
	var query string

	if success {
		query = `UPDATE webhooks SET last_success = $1 WHERE id = $2`
		if db.driver == "sqlite3" {
			query = `UPDATE webhooks SET last_success = ? WHERE id = ?`
		}
	} else {
		query = `UPDATE webhooks SET last_failure = $1 WHERE id = $2`
		if db.driver == "sqlite3" {
			query = `UPDATE webhooks SET last_failure = ? WHERE id = ?`
		}
	}

	_, err := db.Exec(query, now, webhookID)
	return err
}
