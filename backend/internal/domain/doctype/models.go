package doctype

import "errors"

// Text is a UI string in German and English.
type Text struct {
	De string
	En string
}

// Type is a document type: an ordered composition of field groups.
type Type struct {
	ID          string
	Name        string
	Labels      Text
	Description Text
	// FieldGroups lists field-group ids. The system group is always present.
	FieldGroups []string
}

// Input is a create or replace of a document type.
type Input struct {
	Name        string
	Labels      Text
	Description Text
	FieldGroups []string
}

var (
	ErrNotFound      = errors.New("document type not found")
	ErrAlreadyExists = errors.New("document type already exists")
	ErrInvalid       = errors.New("invalid document type")
	ErrUnknownGroup  = errors.New("unknown field group")
	ErrSystemGroup   = errors.New("document type requires the system field group")
	ErrConflict      = errors.New("import needs a decision")
)
