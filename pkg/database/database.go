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
	}

	for i, migration := range migrations {
		if _, err := db.Exec(migration); err != nil {
			return fmt.Errorf("migration %d failed: %w", i, err)
		}
	}

	return nil
}

func (db *DB) createMachinesTable() string {
	autoIncrement := "AUTOINCREMENT"
	jsonType := "TEXT"

	if db.driver == "postgres" {
		autoIncrement = ""
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
	autoIncrement := "AUTOINCREMENT"

	if db.driver == "postgres" {
		autoIncrement = ""
	}

	return fmt.Sprintf(`
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
	`, autoIncrement)
}
