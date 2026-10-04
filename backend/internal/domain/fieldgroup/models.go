package fieldgroup

import "errors"

// Supported value types. Names are stable API values.
const (
	ValueBool      = "bool"
	ValueInt       = "int"
	ValueDecimal   = "decimal"
	ValueDouble    = "double"
	ValueText      = "text"
	ValueMultiline = "multiline"
	ValueDateTime  = "datetime"
)

// Text is a UI string in German and English.
type Text struct {
	De string
	En string
}

// Field is one field definition inside a group.
// Name is unique within the group, compared case-insensitively.
type Field struct {
	Name        string
	Labels      Text
	Description Text
	ValueType   string
}

// Group is a reusable set of field definitions, stored in the instance database.
// Name is unique among groups, compared case-insensitively.
type Group struct {
	ID          string
	Name        string
	Labels      Text
	Description Text
	Fields      []Field
}

// Input is a create or replace of a field group.
type Input struct {
	Name        string
	Labels      Text
	Description Text
	Fields      []Field
}

// Field group failures returned by the service and the store.
var (
	ErrNotFound      = errors.New("field group not found")
	ErrAlreadyExists = errors.New("field group already exists")
	ErrInvalid       = errors.New("invalid field group")
	ErrInUse         = errors.New("field group is used by a document type")
)

// ValueTypes is the allowed set, in UI order.
func ValueTypes() []string {
	return []string{
		ValueBool,
		ValueInt,
		ValueDecimal,
		ValueDouble,
		ValueText,
		ValueMultiline,
		ValueDateTime,
	}
}

// ValidValueType reports whether valueType is one of the simple types.
func ValidValueType(valueType string) bool {
	switch valueType {
	case ValueBool, ValueInt, ValueDecimal, ValueDouble, ValueText, ValueMultiline, ValueDateTime:
		return true
	default:
		return false
	}
}
