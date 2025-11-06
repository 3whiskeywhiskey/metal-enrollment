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
		db.createPowerOperationsTable(),
		db.createMachineMetricsTable(),
		db.createImageTestsTable(),
		db.createWebhooksTable(),
		db.createWebhookDeliveriesTable(),
		db.createMachineTemplatesTable(),
		db.createMachineEventsTable(),
	}

	for i, migration := range migrations {
		if _, err := db.Exec(migration); err != nil {
			return fmt.Errorf("migration %d failed: %w", i, err)
		}
	}

	// Run additional migrations for schema updates
	if err := db.addBMCInfoColumn(); err != nil {
		return fmt.Errorf("failed to add bmc_info column: %w", err)
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

func (db *DB) createPowerOperationsTable() string {
	return `
		CREATE TABLE IF NOT EXISTS power_operations (
			id TEXT PRIMARY KEY,
			machine_id TEXT NOT NULL,
			operation TEXT NOT NULL,
			status TEXT NOT NULL,
			result TEXT,
			error TEXT,
			initiated_by TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL,
			completed_at TIMESTAMP,
			FOREIGN KEY (machine_id) REFERENCES machines(id) ON DELETE CASCADE
		)
	`
}

func (db *DB) createMachineMetricsTable() string {
	return `
		CREATE TABLE IF NOT EXISTS machine_metrics (
			id TEXT PRIMARY KEY,
			machine_id TEXT NOT NULL,
			timestamp TIMESTAMP NOT NULL,
			cpu_usage_percent REAL NOT NULL,
			memory_used_bytes BIGINT NOT NULL,
			memory_total_bytes BIGINT NOT NULL,
			disk_used_bytes BIGINT NOT NULL,
			disk_total_bytes BIGINT NOT NULL,
			network_rx_bytes BIGINT NOT NULL,
			network_tx_bytes BIGINT NOT NULL,
			load_average_1 REAL NOT NULL,
			load_average_5 REAL NOT NULL,
			load_average_15 REAL NOT NULL,
			temperature REAL,
			power_state TEXT NOT NULL,
			uptime BIGINT NOT NULL,
			FOREIGN KEY (machine_id) REFERENCES machines(id) ON DELETE CASCADE
		)
	`
}

func (db *DB) createImageTestsTable() string {
	return `
		CREATE TABLE IF NOT EXISTS image_tests (
			id TEXT PRIMARY KEY,
			image_path TEXT NOT NULL,
			image_type TEXT NOT NULL,
			test_type TEXT NOT NULL,
			status TEXT NOT NULL,
			result TEXT,
			error TEXT,
			machine_id TEXT,
			created_at TIMESTAMP NOT NULL,
			completed_at TIMESTAMP,
			FOREIGN KEY (machine_id) REFERENCES machines(id) ON DELETE SET NULL
		)
	`
}

// addBMCInfoColumn adds the bmc_info column to machines table if it doesn't exist
func (db *DB) addBMCInfoColumn() error {
	jsonType := "TEXT"
	if db.driver == "postgres" {
		jsonType = "JSONB"
	}

	// For SQLite, check if column exists first
	if db.driver == "sqlite3" {
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('machines') WHERE name='bmc_info'").Scan(&count)
		if err != nil {
			return err
		}
		if count > 0 {
			return nil // Column already exists
		}

		_, err = db.Exec(fmt.Sprintf("ALTER TABLE machines ADD COLUMN bmc_info %s", jsonType))
		return err
	}

	// For PostgreSQL
	_, err := db.Exec(fmt.Sprintf(`
		ALTER TABLE machines
		ADD COLUMN IF NOT EXISTS bmc_info %s
	`, jsonType))
	return err
}

func (db *DB) createWebhooksTable() string {
	jsonType := "TEXT"
	if db.driver == "postgres" {
		jsonType = "JSONB"
	}

	return fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS webhooks (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			url TEXT NOT NULL,
			events %s NOT NULL,
			secret TEXT,
			active BOOLEAN NOT NULL DEFAULT TRUE,
			headers %s,
			timeout INTEGER NOT NULL DEFAULT 30,
			max_retries INTEGER NOT NULL DEFAULT 3,
			last_success TIMESTAMP,
			last_failure TIMESTAMP,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL
		)
	`, jsonType, jsonType)
}

func (db *DB) createWebhookDeliveriesTable() string {
	return `
		CREATE TABLE IF NOT EXISTS webhook_deliveries (
			id TEXT PRIMARY KEY,
			webhook_id TEXT NOT NULL,
			event TEXT NOT NULL,
			payload TEXT NOT NULL,
			status_code INTEGER NOT NULL,
			response TEXT,
			error TEXT,
			attempts INTEGER NOT NULL DEFAULT 1,
			success BOOLEAN NOT NULL,
			created_at TIMESTAMP NOT NULL,
			completed_at TIMESTAMP,
			FOREIGN KEY (webhook_id) REFERENCES webhooks(id) ON DELETE CASCADE
		)
	`
}

func (db *DB) createMachineTemplatesTable() string {
	jsonType := "TEXT"
	if db.driver == "postgres" {
		jsonType = "JSONB"
	}

	return fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS machine_templates (
			id TEXT PRIMARY KEY,
			name TEXT UNIQUE NOT NULL,
			description TEXT,
			nixos_config TEXT NOT NULL,
			bmc_config %s,
			tags %s,
			variables %s,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL,
			created_by TEXT NOT NULL
		)
	`, jsonType, jsonType, jsonType)
}

func (db *DB) createMachineEventsTable() string {
	jsonType := "TEXT"
	if db.driver == "postgres" {
		jsonType = "JSONB"
	}

	return fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS machine_events (
			id TEXT PRIMARY KEY,
			machine_id TEXT NOT NULL,
			event TEXT NOT NULL,
			data %s,
			created_at TIMESTAMP NOT NULL,
			created_by TEXT,
			FOREIGN KEY (machine_id) REFERENCES machines(id) ON DELETE CASCADE
		)
	`, jsonType)
}
