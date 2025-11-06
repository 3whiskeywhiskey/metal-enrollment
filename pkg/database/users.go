package database

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/3whiskeywhiskey/metal-enrollment/pkg/models"
	"github.com/google/uuid"
)

// CreateUser creates a new user
func (db *DB) CreateUser(username, email, passwordHash string, role models.UserRole) (*models.User, error) {
	user := &models.User{
		ID:           uuid.New().String(),
		Username:     username,
		Email:        email,
		PasswordHash: passwordHash,
		Role:         role,
		Active:       true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	query := `
		INSERT INTO users (id, username, email, password_hash, role, active, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	if db.driver == "postgres" {
		query = `
			INSERT INTO users (id, username, email, password_hash, role, active, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`
	}

	_, err := db.Exec(query,
		user.ID,
		user.Username,
		user.Email,
		user.PasswordHash,
		user.Role,
		user.Active,
		user.CreatedAt,
		user.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

// GetUser retrieves a user by ID
func (db *DB) GetUser(id string) (*models.User, error) {
	user := &models.User{}
	var lastLoginAt sql.NullTime

	query := `
		SELECT id, username, email, password_hash, role, active, created_at, updated_at, last_login_at
		FROM users WHERE id = ?
	`

	if db.driver == "postgres" {
		query = `
			SELECT id, username, email, password_hash, role, active, created_at, updated_at, last_login_at
			FROM users WHERE id = $1
		`
	}

	err := db.QueryRow(query, id).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.Active,
		&user.CreatedAt,
		&user.UpdatedAt,
		&lastLoginAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if lastLoginAt.Valid {
		user.LastLoginAt = &lastLoginAt.Time
	}

	return user, nil
}

// GetUserByUsername retrieves a user by username
func (db *DB) GetUserByUsername(username string) (*models.User, error) {
	user := &models.User{}
	var lastLoginAt sql.NullTime

	query := `
		SELECT id, username, email, password_hash, role, active, created_at, updated_at, last_login_at
		FROM users WHERE username = ?
	`

	if db.driver == "postgres" {
		query = `
			SELECT id, username, email, password_hash, role, active, created_at, updated_at, last_login_at
			FROM users WHERE username = $1
		`
	}

	err := db.QueryRow(query, username).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.Active,
		&user.CreatedAt,
		&user.UpdatedAt,
		&lastLoginAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if lastLoginAt.Valid {
		user.LastLoginAt = &lastLoginAt.Time
	}

	return user, nil
}

// ListUsers retrieves all users
func (db *DB) ListUsers() ([]*models.User, error) {
	query := `
		SELECT id, username, email, password_hash, role, active, created_at, updated_at, last_login_at
		FROM users
		ORDER BY created_at DESC
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		user := &models.User{}
		var lastLoginAt sql.NullTime

		err := rows.Scan(
			&user.ID,
			&user.Username,
			&user.Email,
			&user.PasswordHash,
			&user.Role,
			&user.Active,
			&user.CreatedAt,
			&user.UpdatedAt,
			&lastLoginAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}

		if lastLoginAt.Valid {
			user.LastLoginAt = &lastLoginAt.Time
		}

		users = append(users, user)
	}

	return users, nil
}

// UpdateUser updates a user record
func (db *DB) UpdateUser(user *models.User) error {
	user.UpdatedAt = time.Now()

	query := `
		UPDATE users SET
			email = ?, password_hash = ?, role = ?, active = ?, updated_at = ?, last_login_at = ?
		WHERE id = ?
	`

	if db.driver == "postgres" {
		query = `
			UPDATE users SET
				email = $1, password_hash = $2, role = $3, active = $4, updated_at = $5, last_login_at = $6
			WHERE id = $7
		`
	}

	_, err := db.Exec(query,
		user.Email,
		user.PasswordHash,
		user.Role,
		user.Active,
		user.UpdatedAt,
		user.LastLoginAt,
		user.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}

// DeleteUser deletes a user record
func (db *DB) DeleteUser(id string) error {
	query := "DELETE FROM users WHERE id = ?"
	if db.driver == "postgres" {
		query = "DELETE FROM users WHERE id = $1"
	}

	_, err := db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	return nil
}

// UpdateLastLogin updates the last login timestamp for a user
func (db *DB) UpdateLastLogin(userID string) error {
	now := time.Now()
	query := "UPDATE users SET last_login_at = ? WHERE id = ?"

	if db.driver == "postgres" {
		query = "UPDATE users SET last_login_at = $1 WHERE id = $2"
	}

	_, err := db.Exec(query, now, userID)
	if err != nil {
		return fmt.Errorf("failed to update last login: %w", err)
	}

	return nil
}
