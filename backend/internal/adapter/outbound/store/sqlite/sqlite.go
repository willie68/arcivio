package sqlite

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// Store is an embedded SQLite database used as metadata cache.
type Store struct {
	db   *sql.DB
	path string
}

// New opens (or creates) the SQLite file and applies the bootstrap schema.
func New(path string) (*Store, error) {
	if path == "" {
		path = filepath.Join("data", "arcivio.db")
	}
	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create sqlite dir: %w", err)
		}
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	if _, err := db.Exec(`PRAGMA foreign_keys = ON`); err != nil {
		_ = db.Close()
		return nil, err
	}
	if _, err := db.Exec(`
CREATE TABLE IF NOT EXISTS schema_migrations (
	version TEXT PRIMARY KEY,
	applied_at TEXT NOT NULL
)`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("migrate sqlite: %w", err)
	}
	return &Store{db: db, path: path}, nil
}

// Ping checks the database connection.
func (s *Store) Ping() error {
	if s == nil || s.db == nil {
		return fmt.Errorf("sqlite store not open")
	}
	return s.db.Ping()
}

// Shutdown implements do.Shutdowner.
func (s *Store) Shutdown() error {
	return s.Close()
}

// Close closes the database.
func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

// DB exposes the sql handle for later domains.
func (s *Store) DB() *sql.DB {
	return s.db
}
