package fieldgroup

import "context"

// Store persists field group definitions in the instance database.
type Store interface {
	List(ctx context.Context) ([]Group, error)
	GetByID(ctx context.Context, id string) (*Group, error)
	GetByName(ctx context.Context, name string) (*Group, error)
	Create(ctx context.Context, group Group) error
	Update(ctx context.Context, group Group) error
	Delete(ctx context.Context, id string) error
}
