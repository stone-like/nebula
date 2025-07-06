package memory

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// Database handles SQLite database operations
type Database struct {
	db *sql.DB
}

// NewDatabase creates a new database instance
func NewDatabase(dbPath string) (*Database, error) {
	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	database := &Database{db: db}

	// Initialize tables
	if err := database.initTables(); err != nil {
		return nil, fmt.Errorf("failed to initialize tables: %w", err)
	}

	return database, nil
}

// Close closes the database connection
func (d *Database) Close() error {
	return d.db.Close()
}

// initTables creates the necessary tables if they don't exist
func (d *Database) initTables() error {
	// Create sessions table
	sessionTableSQL := `
	CREATE TABLE IF NOT EXISTS sessions (
		id TEXT PRIMARY KEY,
		started_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		ended_at DATETIME,
		project_path TEXT NOT NULL,
		model_used TEXT NOT NULL
	);`

	if _, err := d.db.Exec(sessionTableSQL); err != nil {
		return fmt.Errorf("failed to create sessions table: %w", err)
	}

	// Create messages table
	messageTableSQL := `
	CREATE TABLE IF NOT EXISTS messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		session_id TEXT REFERENCES sessions(id),
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		role TEXT NOT NULL,
		content TEXT,
		tool_calls TEXT,
		tool_results TEXT
	);`

	if _, err := d.db.Exec(messageTableSQL); err != nil {
		return fmt.Errorf("failed to create messages table: %w", err)
	}

	// Create indexes for better performance
	indexSQL := []string{
		"CREATE INDEX IF NOT EXISTS idx_sessions_project_path ON sessions(project_path);",
		"CREATE INDEX IF NOT EXISTS idx_messages_session_id ON messages(session_id);",
		"CREATE INDEX IF NOT EXISTS idx_messages_timestamp ON messages(timestamp);",
	}

	for _, sql := range indexSQL {
		if _, err := d.db.Exec(sql); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	return nil
}

// GetDB returns the underlying database connection
func (d *Database) GetDB() *sql.DB {
	return d.db
}