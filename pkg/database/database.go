package database

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

// Config holds database configuration
type Config struct {
	Driver string
	DSN    string
}

// DB wraps the database connection
type DB struct {
	*sql.DB
	driver string
}

// New creates a new database connection
func New(cfg Config) (*DB, error) {
	db, err := sql.Open(cfg.Driver, cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Verify connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DB{DB: db, driver: cfg.Driver}, nil
}

// Driver returns the database driver name
func (db *DB) Driver() string {
	return db.driver
}

// Migrate runs database migrations
func (db *DB) Migrate() error {
	migrations := []string{
		db.createMachinesTable(),
		db.createBuildsTable(),
		db.createUsersTable(),
		db.createAPIKeysTable(),
		db.createGroupsTable(),
		db.createGroupMembershipsTable(),
	}

	for i, migration := range migrations {
		if _, err := db.Exec(migration); err != nil {
			return fmt.Errorf("migration %d failed: %w", i, err)
		}
	}

	return nil
}

func (db *DB) createMachinesTable() string {
	jsonType := "TEXT"

	if db.driver == "postgres" {
		jsonType = "JSONB"
	}

	return fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS machines (
			id TEXT PRIMARY KEY,
			service_tag TEXT UNIQUE NOT NULL,
			mac_address TEXT NOT NULL,
			status TEXT NOT NULL,
			hostname TEXT,
			description TEXT,
			hardware %s,
			nixos_config TEXT,
			last_build_id TEXT,
			last_build_time TIMESTAMP,
			enrolled_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL,
			last_seen_at TIMESTAMP
		)
	`, jsonType)
}

func (db *DB) createBuildsTable() string {
	return `
		CREATE TABLE IF NOT EXISTS builds (
			id TEXT PRIMARY KEY,
			machine_id TEXT NOT NULL,
			status TEXT NOT NULL,
			config TEXT NOT NULL,
			log_output TEXT,
			error TEXT,
			artifact_url TEXT,
			created_at TIMESTAMP NOT NULL,
			completed_at TIMESTAMP,
			FOREIGN KEY (machine_id) REFERENCES machines(id)
		)
	`
}

func (db *DB) createUsersTable() string {
	return `
		CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			username TEXT UNIQUE NOT NULL,
			email TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			role TEXT NOT NULL,
			active BOOLEAN NOT NULL DEFAULT TRUE,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL,
			last_login_at TIMESTAMP
		)
	`
}

func (db *DB) createAPIKeysTable() string {
	return `
		CREATE TABLE IF NOT EXISTS api_keys (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			name TEXT NOT NULL,
			key TEXT UNIQUE NOT NULL,
			active BOOLEAN NOT NULL DEFAULT TRUE,
			created_at TIMESTAMP NOT NULL,
			expires_at TIMESTAMP,
			last_used_at TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)
	`
}

func (db *DB) createGroupsTable() string {
	jsonArrayType := "TEXT"
	if db.driver == "postgres" {
		jsonArrayType = "JSONB"
	}

	return fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS groups (
			id TEXT PRIMARY KEY,
			name TEXT UNIQUE NOT NULL,
			description TEXT,
			tags %s,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL
		)
	`, jsonArrayType)
}

func (db *DB) createGroupMembershipsTable() string {
	return `
		CREATE TABLE IF NOT EXISTS group_memberships (
			group_id TEXT NOT NULL,
			machine_id TEXT NOT NULL,
			added_at TIMESTAMP NOT NULL,
			PRIMARY KEY (group_id, machine_id),
			FOREIGN KEY (group_id) REFERENCES groups(id) ON DELETE CASCADE,
			FOREIGN KEY (machine_id) REFERENCES machines(id) ON DELETE CASCADE
		)
	`
}
