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

const (
	usersMigration     = "002_identity_users"
	lastLoginMigration = "003_identity_users_last_login"
	profileMigration   = "004_identity_users_profile"
)

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
	if n == 0 {
		if _, err := r.db.Exec(
			`INSERT INTO schema_migrations (version, applied_at) VALUES (?, ?)`,
			usersMigration, time.Now().UTC().Format(time.RFC3339),
		); err != nil {
			return err
		}
	}
	if err := r.migrateLastLogin(); err != nil {
		return err
	}
	return r.migrateProfile()
}

func (r *Repo) migrateLastLogin() error {
	var n int
	if err := r.db.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE version = ?`, lastLoginMigration).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	if _, err := r.db.Exec(`ALTER TABLE users ADD COLUMN last_login TEXT`); err != nil {
		return fmt.Errorf("add last_login: %w", err)
	}
	if _, err := r.db.Exec(
		`INSERT INTO schema_migrations (version, applied_at) VALUES (?, ?)`,
		lastLoginMigration, time.Now().UTC().Format(time.RFC3339),
	); err != nil {
		return err
	}
	return nil
}

func (r *Repo) migrateProfile() error {
	var n int
	if err := r.db.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE version = ?`, profileMigration).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	for _, stmt := range []string{
		`ALTER TABLE users ADD COLUMN first_name TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE users ADD COLUMN last_name TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE users ADD COLUMN email TEXT NOT NULL DEFAULT ''`,
	} {
		if _, err := r.db.Exec(stmt); err != nil {
			return fmt.Errorf("add profile columns: %w", err)
		}
	}
	if _, err := r.db.Exec(
		`INSERT INTO schema_migrations (version, applied_at) VALUES (?, ?)`,
		profileMigration, time.Now().UTC().Format(time.RFC3339),
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
	return scanUser(r.db.QueryRowContext(ctx, userSelect+` WHERE id = ?`, id))
}

// GetByUsername implements identity.UserStore.
func (r *Repo) GetByUsername(ctx context.Context, username string) (*identity.User, error) {
	return scanUser(r.db.QueryRowContext(ctx, userSelect+` WHERE username = ?`, strings.ToLower(username)))
}

const userSelect = `SELECT id, username, password_hash, roles, must_change_password, created_at, updated_at, last_login, first_name, last_name, email FROM users`

// List implements identity.UserStore.
func (r *Repo) List(ctx context.Context, offset, limit int, sort string, desc bool, prefix string) ([]identity.User, int, error) {
	where, args := prefixWhere(prefix)
	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	queryArgs := append(append([]any{}, args...), limit, offset)
	rows, err := r.db.QueryContext(ctx, userSelect+where+listOrderSQL(sort, desc), queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := make([]identity.User, 0)
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, *u)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

func listOrderSQL(sort string, desc bool) string {
	canonical, ok := identity.CanonicalUserSort(sort)
	if !ok {
		canonical = identity.SortUsername
		desc = false
	}
	column := "username COLLATE NOCASE"
	switch canonical {
	case identity.SortFirstName:
		column = "first_name COLLATE NOCASE"
	case identity.SortLastName:
		column = "last_name COLLATE NOCASE"
	case identity.SortEmail:
		column = "email COLLATE NOCASE"
	case identity.SortRoles:
		column = "roles COLLATE NOCASE"
	case identity.SortLastLogin:
		column = "last_login"
	case identity.SortMustChange:
		column = "must_change_password"
	}
	direction := "ASC"
	if desc {
		direction = "DESC"
	}
	return " ORDER BY " + column + " " + direction + ", username COLLATE NOCASE ASC LIMIT ? OFFSET ?"
}

func prefixWhere(prefix string) (string, []any) {
	prefix = identity.NormalizePrefix(prefix)
	if prefix == "" {
		return "", nil
	}
	pattern := likePrefix(prefix)
	return ` WHERE username LIKE ? ESCAPE '\' OR first_name LIKE ? ESCAPE '\' OR last_name LIKE ? ESCAPE '\' OR email LIKE ? ESCAPE '\'`,
		[]any{pattern, pattern, pattern, pattern}
}

func likePrefix(value string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return replacer.Replace(value) + "%"
}

func scanUser(row interface{ Scan(dest ...any) error }) (*identity.User, error) {
	var u identity.User
	var rolesJSON string
	var must int
	var created, updated string
	var lastLogin sql.NullString
	err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &rolesJSON, &must, &created, &updated, &lastLogin, &u.FirstName, &u.LastName, &u.Email)
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
	if lastLogin.Valid && lastLogin.String != "" {
		if t, err := time.Parse(time.RFC3339, lastLogin.String); err == nil {
			u.LastLogin = &t
		}
	}
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
INSERT INTO users (id, username, password_hash, roles, must_change_password, created_at, updated_at, first_name, last_name, email)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		user.ID, user.Username, user.PasswordHash, string(rolesJSON), must,
		user.CreatedAt.UTC().Format(time.RFC3339), user.UpdatedAt.UTC().Format(time.RFC3339),
		user.FirstName, user.LastName, user.Email,
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
UPDATE users SET username = ?, first_name = ?, last_name = ?, email = ?, password_hash = ?, roles = ?, must_change_password = ?, updated_at = ?
WHERE id = ?`,
		user.Username, user.FirstName, user.LastName, user.Email,
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

// RecordLastLogin implements identity.UserStore.
func (r *Repo) RecordLastLogin(ctx context.Context, userID string, at time.Time) error {
	res, err := r.db.ExecContext(ctx, `UPDATE users SET last_login = ? WHERE id = ?`,
		at.UTC().Format(time.RFC3339), userID)
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

// Delete implements identity.UserStore.
func (r *Repo) Delete(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM users WHERE id = ?`, id)
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

// CountWithRole implements identity.UserStore.
func (r *Repo) CountWithRole(ctx context.Context, role string) (int, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT roles FROM users`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	n := 0
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return 0, err
		}
		var roles []string
		if err := json.Unmarshal([]byte(raw), &roles); err != nil {
			return 0, err
		}
		for _, got := range roles {
			if got == role {
				n++
				break
			}
		}
	}
	return n, rows.Err()
}
