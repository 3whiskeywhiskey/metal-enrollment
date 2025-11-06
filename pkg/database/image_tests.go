package database

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/3whiskeywhiskey/metal-enrollment/pkg/models"
	"github.com/google/uuid"
)

// CreateImageTest creates a new image test record
func (db *DB) CreateImageTest(test *models.ImageTest) error {
	test.ID = uuid.New().String()
	test.CreatedAt = time.Now()

	query := `
		INSERT INTO image_tests (
			id, image_path, image_type, test_type, status, result, error, machine_id, created_at, completed_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	if db.driver == "postgres" {
		query = `
			INSERT INTO image_tests (
				id, image_path, image_type, test_type, status, result, error, machine_id, created_at, completed_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		`
	}

	_, err := db.Exec(query,
		test.ID,
		test.ImagePath,
		test.ImageType,
		test.TestType,
		test.Status,
		test.Result,
		test.Error,
		test.MachineID,
		test.CreatedAt,
		test.CompletedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create image test: %w", err)
	}

	return nil
}

// UpdateImageTest updates an image test record
func (db *DB) UpdateImageTest(test *models.ImageTest) error {
	query := `
		UPDATE image_tests SET
			status = ?, result = ?, error = ?, completed_at = ?
		WHERE id = ?
	`

	if db.driver == "postgres" {
		query = `
			UPDATE image_tests SET
				status = $1, result = $2, error = $3, completed_at = $4
			WHERE id = $5
		`
	}

	_, err := db.Exec(query,
		test.Status,
		test.Result,
		test.Error,
		test.CompletedAt,
		test.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update image test: %w", err)
	}

	return nil
}

// GetImageTest retrieves an image test by ID
func (db *DB) GetImageTest(id string) (*models.ImageTest, error) {
	test := &models.ImageTest{}
	var result, errorMsg sql.NullString
	var machineID sql.NullString
	var completedAt sql.NullTime

	query := `
		SELECT id, image_path, image_type, test_type, status, result, error, machine_id, created_at, completed_at
		FROM image_tests WHERE id = ?
	`

	if db.driver == "postgres" {
		query = `
			SELECT id, image_path, image_type, test_type, status, result, error, machine_id, created_at, completed_at
			FROM image_tests WHERE id = $1
		`
	}

	err := db.QueryRow(query, id).Scan(
		&test.ID,
		&test.ImagePath,
		&test.ImageType,
		&test.TestType,
		&test.Status,
		&result,
		&errorMsg,
		&machineID,
		&test.CreatedAt,
		&completedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get image test: %w", err)
	}

	if result.Valid {
		test.Result = result.String
	}
	if errorMsg.Valid {
		test.Error = errorMsg.String
	}
	if machineID.Valid {
		mid := machineID.String
		test.MachineID = &mid
	}
	if completedAt.Valid {
		test.CompletedAt = &completedAt.Time
	}

	return test, nil
}

// ListImageTests retrieves image tests
func (db *DB) ListImageTests(imageType string, limit int) ([]*models.ImageTest, error) {
	var query string
	var args []interface{}

	if imageType != "" {
		query = `
			SELECT id, image_path, image_type, test_type, status, result, error, machine_id, created_at, completed_at
			FROM image_tests
			WHERE image_type = ?
			ORDER BY created_at DESC
			LIMIT ?
		`
		if db.driver == "postgres" {
			query = `
				SELECT id, image_path, image_type, test_type, status, result, error, machine_id, created_at, completed_at
				FROM image_tests
				WHERE image_type = $1
				ORDER BY created_at DESC
				LIMIT $2
			`
		}
		args = []interface{}{imageType, limit}
	} else {
		query = `
			SELECT id, image_path, image_type, test_type, status, result, error, machine_id, created_at, completed_at
			FROM image_tests
			ORDER BY created_at DESC
			LIMIT ?
		`
		if db.driver == "postgres" {
			query = `
				SELECT id, image_path, image_type, test_type, status, result, error, machine_id, created_at, completed_at
				FROM image_tests
				ORDER BY created_at DESC
				LIMIT $1
			`
		}
		args = []interface{}{limit}
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list image tests: %w", err)
	}
	defer rows.Close()

	var tests []*models.ImageTest
	for rows.Next() {
		test := &models.ImageTest{}
		var result, errorMsg sql.NullString
		var machineID sql.NullString
		var completedAt sql.NullTime

		err := rows.Scan(
			&test.ID,
			&test.ImagePath,
			&test.ImageType,
			&test.TestType,
			&test.Status,
			&result,
			&errorMsg,
			&machineID,
			&test.CreatedAt,
			&completedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan image test: %w", err)
		}

		if result.Valid {
			test.Result = result.String
		}
		if errorMsg.Valid {
			test.Error = errorMsg.String
		}
		if machineID.Valid {
			mid := machineID.String
			test.MachineID = &mid
		}
		if completedAt.Valid {
			test.CompletedAt = &completedAt.Time
		}

		tests = append(tests, test)
	}

	return tests, nil
}
