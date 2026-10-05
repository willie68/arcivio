package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/willie68/arcivio/internal/domain/doctype"
)

const migration = "006_document_types"

type definition struct {
	Labels      textJSON `json:"labels"`
	Description textJSON `json:"description"`
	FieldGroups []string `json:"fieldGroups"`
}

type textJSON struct {
	De string `json:"de"`
	En string `json:"en"`
}

// Repo stores document types in the instance SQLite database.
type Repo struct {
	db *sql.DB
}

// New migrates the document_types table and returns a Store.
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
CREATE TABLE IF NOT EXISTS document_types (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL UNIQUE COLLATE NOCASE,
	definition TEXT NOT NULL,
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL
)`); err != nil {
		return fmt.Errorf("create document_types: %w", err)
	}
	var n int
	if err := r.db.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE version = ?`, migration).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	if _, err := r.db.Exec(
		`INSERT INTO schema_migrations (version, applied_at) VALUES (?, ?)`,
		migration, time.Now().UTC().Format(time.RFC3339),
	); err != nil {
		return err
	}
	return nil
}

// List implements doctype.Store.
func (r *Repo) List(ctx context.Context) ([]doctype.Type, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, name, definition FROM document_types ORDER BY name COLLATE NOCASE`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	types := make([]doctype.Type, 0)
	for rows.Next() {
		docType, err := scanType(rows)
		if err != nil {
			return nil, err
		}
		types = append(types, docType)
	}
	return types, rows.Err()
}

// GetByID implements doctype.Store.
func (r *Repo) GetByID(ctx context.Context, id string) (*doctype.Type, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, name, definition FROM document_types WHERE id = ?`, id)
	docType, err := scanType(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, doctype.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &docType, nil
}

// GetByName implements doctype.Store. The match is case-insensitive.
func (r *Repo) GetByName(ctx context.Context, name string) (*doctype.Type, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, name, definition FROM document_types WHERE name = ? COLLATE NOCASE`, name)
	docType, err := scanType(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, doctype.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &docType, nil
}

// Create implements doctype.Store.
func (r *Repo) Create(ctx context.Context, docType doctype.Type) error {
	raw, err := marshalDefinition(docType)
	if err != nil {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, err = r.db.ExecContext(ctx, `
INSERT INTO document_types (id, name, definition, created_at, updated_at)
VALUES (?, ?, ?, ?, ?)`, docType.ID, docType.Name, raw, now, now)
	if err != nil {
		return mapWriteErr(err)
	}
	return nil
}

// Update implements doctype.Store.
func (r *Repo) Update(ctx context.Context, docType doctype.Type) error {
	raw, err := marshalDefinition(docType)
	if err != nil {
		return err
	}
	res, err := r.db.ExecContext(ctx, `
UPDATE document_types
SET name = ?, definition = ?, updated_at = ?
WHERE id = ?`, docType.Name, raw, time.Now().UTC().Format(time.RFC3339), docType.ID)
	if err != nil {
		return mapWriteErr(err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return doctype.ErrNotFound
	}
	return nil
}

// Delete implements doctype.Store.
func (r *Repo) Delete(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM document_types WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return doctype.ErrNotFound
	}
	return nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanType(row scanner) (doctype.Type, error) {
	var docType doctype.Type
	var raw string
	if err := row.Scan(&docType.ID, &docType.Name, &raw); err != nil {
		return doctype.Type{}, err
	}
	var def definition
	if err := json.Unmarshal([]byte(raw), &def); err != nil {
		return doctype.Type{}, fmt.Errorf("decode document type %s: %w", docType.ID, err)
	}
	docType.Labels = doctype.Text{De: def.Labels.De, En: def.Labels.En}
	docType.Description = doctype.Text{De: def.Description.De, En: def.Description.En}
	docType.FieldGroups = def.FieldGroups
	if docType.FieldGroups == nil {
		docType.FieldGroups = []string{}
	}
	return docType, nil
}

func marshalDefinition(docType doctype.Type) (string, error) {
	groups := docType.FieldGroups
	if groups == nil {
		groups = []string{}
	}
	raw, err := json.Marshal(definition{
		Labels:      textJSON{De: docType.Labels.De, En: docType.Labels.En},
		Description: textJSON{De: docType.Description.De, En: docType.Description.En},
		FieldGroups: groups,
	})
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func mapWriteErr(err error) error {
	if strings.Contains(err.Error(), "UNIQUE constraint failed") {
		return doctype.ErrAlreadyExists
	}
	return err
}
