package fieldgroup

import "context"

// Store persists field group definitions in the instance database.
type Store interface {
	List(ctx context.Context) ([]FieldGroup, error)
	GetByID(ctx context.Context, id string) (*FieldGroup, error)
	GetByName(ctx context.Context, name string) (*FieldGroup, error)
	Create(ctx context.Context, group FieldGroup) error
	Update(ctx context.Context, group FieldGroup) error
	Delete(ctx context.Context, id string) error
}
