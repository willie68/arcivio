package doctype

import "context"

// Store persists document types in the instance database.
type Store interface {
	List(ctx context.Context) ([]Type, error)
	GetByID(ctx context.Context, id string) (*Type, error)
	GetByName(ctx context.Context, name string) (*Type, error)
	Create(ctx context.Context, docType Type) error
	Update(ctx context.Context, docType Type) error
	Delete(ctx context.Context, id string) error
}
