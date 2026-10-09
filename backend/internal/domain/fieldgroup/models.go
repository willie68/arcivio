package fieldgroup

import (
	"errors"

	"github.com/willie68/arcivio/internal/domain/models"
)

// SystemGroupID is the shipped field group every document type includes.
const SystemGroupID = "system"

// Supported value types. Names are stable API values.

// Field is one field definition inside a group.
// Name is unique within the group, compared case-insensitively.
type Field struct {
	Name        string           `json:"name"`
	Labels      models.LabelText `json:"labels"`
	Description models.LabelText `json:"description"`
	ValueType   string           `json:"valueType"`
	// Mandatory means the field must have a value before the document is saved.
	Mandatory bool `json:"mandatory"`
}

// FieldGroup is a reusable set of field definitions, stored in the instance database.
// Name is unique among groups, compared case-insensitively.
type FieldGroup struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	Labels      models.LabelText `json:"labels"`
	Description models.LabelText `json:"description"`
	// Readonly groups are shipped with the system and cannot be changed or deleted.
	Readonly bool    `json:"readonly"`
	Fields   []Field `json:"fields"`
}

// Input is a create or replace of a field group.
type Input struct {
	Name        string
	Labels      models.LabelText
	Description models.LabelText
	Fields      []Field
}

var (
	ErrNotFound      = errors.New("field group not found")
	ErrAlreadyExists = errors.New("field group already exists")
	ErrInvalid       = errors.New("invalid field group")
	ErrReadonly      = errors.New("field group is read-only")
)

// ValueTypes is the allowed set, in UI order.
func ValueTypes() []string {
	return []string{
		models.ValueTypeBool,
		models.ValueTypeInt,
		models.ValueTypeDecimal,
		models.ValueTypeDouble,
		models.ValueTypeText,
		models.ValueTypeMultiline,
		models.ValueTypeDateTime,
		models.ValueTypeIntRef,
		models.ValueTypeFile,
	}
}

// ValidValueType reports whether valueType is one of the allowed types.
func ValidValueType(valueType string) bool {
	switch valueType {
	case models.ValueTypeBool, models.ValueTypeInt, models.ValueTypeDecimal, models.ValueTypeDouble, models.ValueTypeText, models.ValueTypeMultiline, models.ValueTypeDateTime, models.ValueTypeIntRef, models.ValueTypeFile:
		return true
	default:
		return false
	}
}
