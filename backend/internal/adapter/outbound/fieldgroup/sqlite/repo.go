package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/willie68/arcivio/internal/domain/fieldgroup"
	"github.com/willie68/arcivio/internal/domain/models"
)

const migration = "005_field_groups"

type definition struct {
	Labels      textJSON    `json:"labels"`
	Description textJSON    `json:"description"`
	Readonly    bool        `json:"readonly"`
	Fields      []fieldJSON `json:"fields"`
}

type textJSON struct {
	De string `json:"de"`
	En string `json:"en"`
}

type fieldJSON struct {
	Name        string   `json:"name"`
	Labels      textJSON `json:"labels"`
	Description textJSON `json:"description"`
	ValueType   string   `json:"valueType"`
	Mandatory   bool     `json:"mandatory"`
}

// Repo stores field groups in the instance SQLite database.
type Repo struct {
	db *sql.DB
}

// New migrates the field_groups table and returns a Store.
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
CREATE TABLE IF NOT EXISTS field_groups (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL UNIQUE COLLATE NOCASE,
	definition TEXT NOT NULL,
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL
)`); err != nil {
		return fmt.Errorf("create field_groups: %w", err)
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

// List implements fieldgroup.Store.
func (r *Repo) List(ctx context.Context) ([]fieldgroup.FieldGroup, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, name, definition FROM field_groups ORDER BY name COLLATE NOCASE`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	groups := make([]fieldgroup.FieldGroup, 0)
	for rows.Next() {
		group, err := scanGroup(rows)
		if err != nil {
			return nil, err
		}
		groups = append(groups, group)
	}
	return groups, rows.Err()
}

// GetByID implements fieldgroup.Store.
func (r *Repo) GetByID(ctx context.Context, id string) (*fieldgroup.FieldGroup, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, name, definition FROM field_groups WHERE id = ?`, id)
	group, err := scanGroup(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fieldgroup.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &group, nil
}

// GetByName implements fieldgroup.Store. The match is case-insensitive.
func (r *Repo) GetByName(ctx context.Context, name string) (*fieldgroup.FieldGroup, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, name, definition FROM field_groups WHERE name = ? COLLATE NOCASE`, name)
	group, err := scanGroup(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fieldgroup.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &group, nil
}

// Create implements fieldgroup.Store.
func (r *Repo) Create(ctx context.Context, group fieldgroup.FieldGroup) error {
	raw, err := marshalDefinition(group)
	if err != nil {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, err = r.db.ExecContext(ctx, `
INSERT INTO field_groups (id, name, definition, created_at, updated_at)
VALUES (?, ?, ?, ?, ?)`, group.ID, group.Name, raw, now, now)
	if err != nil {
		return mapWriteErr(err)
	}
	return nil
}

// Update implements fieldgroup.Store.
func (r *Repo) Update(ctx context.Context, group fieldgroup.FieldGroup) error {
	raw, err := marshalDefinition(group)
	if err != nil {
		return err
	}
	res, err := r.db.ExecContext(ctx, `
UPDATE field_groups
SET name = ?, definition = ?, updated_at = ?
WHERE id = ?`, group.Name, raw, time.Now().UTC().Format(time.RFC3339), group.ID)
	if err != nil {
		return mapWriteErr(err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fieldgroup.ErrNotFound
	}
	return nil
}

// Delete implements fieldgroup.Store.
func (r *Repo) Delete(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM field_groups WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fieldgroup.ErrNotFound
	}
	return nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanGroup(row scanner) (fieldgroup.FieldGroup, error) {
	var group fieldgroup.FieldGroup
	var raw string
	if err := row.Scan(&group.ID, &group.Name, &raw); err != nil {
		return fieldgroup.FieldGroup{}, err
	}
	var def definition
	if err := json.Unmarshal([]byte(raw), &def); err != nil {
		return fieldgroup.FieldGroup{}, fmt.Errorf("decode field group %s: %w", group.ID, err)
	}
	group.Labels = toText(def.Labels)
	group.Description = toText(def.Description)
	group.Readonly = def.Readonly
	group.Fields = make([]fieldgroup.Field, 0, len(def.Fields))
	for _, field := range def.Fields {
		group.Fields = append(group.Fields, fieldgroup.Field{
			Name:        field.Name,
			Labels:      toText(field.Labels),
			Description: toText(field.Description),
			ValueType:   field.ValueType,
			Mandatory:   field.Mandatory,
		})
	}
	return group, nil
}

func marshalDefinition(group fieldgroup.FieldGroup) (string, error) {
	fields := make([]fieldJSON, 0, len(group.Fields))
	for _, field := range group.Fields {
		fields = append(fields, fieldJSON{
			Name:        field.Name,
			Labels:      fromText(field.Labels),
			Description: fromText(field.Description),
			ValueType:   field.ValueType,
			Mandatory:   field.Mandatory,
		})
	}
	raw, err := json.Marshal(definition{
		Labels:      fromText(group.Labels),
		Description: fromText(group.Description),
		Readonly:    group.Readonly,
		Fields:      fields,
	})
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func toText(text textJSON) models.LabelText {
	return models.LabelText{De: text.De, En: text.En}
}

func fromText(text models.LabelText) textJSON {
	return textJSON{De: text.De, En: text.En}
}

func mapWriteErr(err error) error {
	if strings.Contains(err.Error(), "UNIQUE constraint failed") {
		return fieldgroup.ErrAlreadyExists
	}
	return err
}
