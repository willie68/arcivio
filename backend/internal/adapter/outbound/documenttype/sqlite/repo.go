package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/willie68/arcivio/internal/domain/documenttype"
)

const migration = "006_document_types"

type definition struct {
	Labels        textJSON `json:"labels"`
	Description   textJSON `json:"description"`
	FieldGroupIDs []string `json:"fieldGroupIds"`
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

// List implements documenttype.Store.
func (r *Repo) List(ctx context.Context) ([]documenttype.Type, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, name, definition FROM document_types ORDER BY name COLLATE NOCASE`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]documenttype.Type, 0)
	for rows.Next() {
		item, err := scanType(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// GetByID implements documenttype.Store.
func (r *Repo) GetByID(ctx context.Context, id string) (*documenttype.Type, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, name, definition FROM document_types WHERE id = ?`, id)
	item, err := scanType(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, documenttype.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

// GetByName implements documenttype.Store. The match is case-insensitive.
func (r *Repo) GetByName(ctx context.Context, name string) (*documenttype.Type, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, name, definition FROM document_types WHERE name = ? COLLATE NOCASE`, name)
	item, err := scanType(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, documenttype.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

// Create implements documenttype.Store.
func (r *Repo) Create(ctx context.Context, docType documenttype.Type) error {
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

// Update implements documenttype.Store.
func (r *Repo) Update(ctx context.Context, docType documenttype.Type) error {
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
		return documenttype.ErrNotFound
	}
	return nil
}

// Delete implements documenttype.Store.
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
		return documenttype.ErrNotFound
	}
	return nil
}

// UsesGroup implements documenttype.Store and fieldgroup.Usage.
func (r *Repo) UsesGroup(ctx context.Context, groupID string) (bool, error) {
	items, err := r.List(ctx)
	if err != nil {
		return false, err
	}
	for _, item := range items {
		for _, id := range item.FieldGroupIDs {
			if id == groupID {
				return true, nil
			}
		}
	}
	return false, nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanType(row scanner) (documenttype.Type, error) {
	var item documenttype.Type
	var raw string
	if err := row.Scan(&item.ID, &item.Name, &raw); err != nil {
		return documenttype.Type{}, err
	}
	var def definition
	if err := json.Unmarshal([]byte(raw), &def); err != nil {
		return documenttype.Type{}, fmt.Errorf("decode document type %s: %w", item.ID, err)
	}
	item.Labels = documenttype.Text{De: def.Labels.De, En: def.Labels.En}
	item.Description = documenttype.Text{De: def.Description.De, En: def.Description.En}
	item.FieldGroupIDs = append([]string(nil), def.FieldGroupIDs...)
	if item.FieldGroupIDs == nil {
		item.FieldGroupIDs = []string{}
	}
	return item, nil
}

func marshalDefinition(docType documenttype.Type) (string, error) {
	ids := docType.FieldGroupIDs
	if ids == nil {
		ids = []string{}
	}
	raw, err := json.Marshal(definition{
		Labels:        textJSON{De: docType.Labels.De, En: docType.Labels.En},
		Description:   textJSON{De: docType.Description.De, En: docType.Description.En},
		FieldGroupIDs: ids,
	})
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func mapWriteErr(err error) error {
	if strings.Contains(err.Error(), "UNIQUE constraint failed") {
		return documenttype.ErrAlreadyExists
	}
	return err
}
