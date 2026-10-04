package documenttype

import "errors"

// Text is a UI string in German and English.
type Text struct {
	De string
	En string
}

// Type is a document type assembled from field groups, in order.
// Name is unique among types, compared case-insensitively.
type Type struct {
	ID            string
	Name          string
	Labels        Text
	Description   Text
	FieldGroupIDs []string
}

// Input is a create or replace of a document type.
type Input struct {
	Name          string
	Labels        Text
	Description   Text
	FieldGroupIDs []string
}

// Document type failures returned by the service and the store.
var (
	ErrNotFound      = errors.New("document type not found")
	ErrAlreadyExists = errors.New("document type already exists")
	ErrInvalid       = errors.New("invalid document type")
)
