package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/willie68/arcivio/internal/domain/identity"
)

const usersMigration = "002_identity_users"

// Repo is the SQLite implementation of identity.UserStore.
type Repo struct {
	db *sql.DB
}

// New migrates the users table and returns a UserStore.
func New(db *sql.DB) (*Repo, error) {
	if db == nil {
		return nil, fmt.Errorf("sqlite db is nil")
	}
	r := &Repo{db: db}
	if err := r.migrate(); err != nil {
		return nil, err
	}
	return r, nil
}

func (r *Repo) migrate() error {
	if _, err := r.db.Exec(`
CREATE TABLE IF NOT EXISTS users (
	id TEXT PRIMARY KEY,
	username TEXT NOT NULL UNIQUE COLLATE NOCASE,
	password_hash TEXT NOT NULL,
	roles TEXT NOT NULL,
	must_change_password INTEGER NOT NULL DEFAULT 1,
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL
)`); err != nil {
		return fmt.Errorf("create users: %w", err)
	}

	var n int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE version = ?`, usersMigration).Scan(&n)
	if err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	if _, err := r.db.Exec(
		`INSERT INTO schema_migrations (version, applied_at) VALUES (?, ?)`,
		usersMigration, time.Now().UTC().Format(time.RFC3339),
	); err != nil {
		return err
	}
	return nil
}

// Count implements identity.UserStore.
func (r *Repo) Count(ctx context.Context) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&n)
	return n, err
}

// GetByID implements identity.UserStore.
func (r *Repo) GetByID(ctx context.Context, id string) (*identity.User, error) {
	return r.scanUser(r.db.QueryRowContext(ctx, userSelect+` WHERE id = ?`, id))
}

// GetByUsername implements identity.UserStore.
func (r *Repo) GetByUsername(ctx context.Context, username string) (*identity.User, error) {
	return r.scanUser(r.db.QueryRowContext(ctx, userSelect+` WHERE username = ?`, strings.ToLower(username)))
}

const userSelect = `SELECT id, username, password_hash, roles, must_change_password, created_at, updated_at FROM users`

func (r *Repo) scanUser(row *sql.Row) (*identity.User, error) {
	var u identity.User
	var rolesJSON string
	var must int
	var created, updated string
	err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &rolesJSON, &must, &created, &updated)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, identity.ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(rolesJSON), &u.Roles); err != nil {
		return nil, fmt.Errorf("decode roles: %w", err)
	}
	u.MustChangePassword = must != 0
	u.CreatedAt, _ = time.Parse(time.RFC3339, created)
	u.UpdatedAt, _ = time.Parse(time.RFC3339, updated)
	return &u, nil
}

// Create implements identity.UserStore.
func (r *Repo) Create(ctx context.Context, user identity.User) error {
	rolesJSON, err := json.Marshal(user.Roles)
	if err != nil {
		return err
	}
	must := 0
	if user.MustChangePassword {
		must = 1
	}
	_, err = r.db.ExecContext(ctx, `
INSERT INTO users (id, username, password_hash, roles, must_change_password, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?)`,
		user.ID, user.Username, user.PasswordHash, string(rolesJSON), must,
		user.CreatedAt.UTC().Format(time.RFC3339), user.UpdatedAt.UTC().Format(time.RFC3339),
	)
	return err
}

// Update implements identity.UserStore.
func (r *Repo) Update(ctx context.Context, user identity.User) error {
	rolesJSON, err := json.Marshal(user.Roles)
	if err != nil {
		return err
	}
	must := 0
	if user.MustChangePassword {
		must = 1
	}
	res, err := r.db.ExecContext(ctx, `
UPDATE users SET password_hash = ?, roles = ?, must_change_password = ?, updated_at = ?
WHERE id = ?`,
		user.PasswordHash, string(rolesJSON), must, user.UpdatedAt.UTC().Format(time.RFC3339), user.ID,
	)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return identity.ErrUserNotFound
	}
	return nil
}
