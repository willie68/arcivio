package fieldgroup

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/rs/xid"
)

const maxNameLen = 64

// Service is the field-group use case.
type Service struct {
	store Store
	ids   func() string
}

// New creates the field-group service.
func New(store Store) *Service {
	return &Service{store: store, ids: func() string { return xid.New().String() }}
}

// List returns every field group, ordered by name.
func (s *Service) List(ctx context.Context) ([]Group, error) {
	groups, err := s.store.List(ctx)
	if err != nil {
		return nil, err
	}
	if groups == nil {
		return []Group{}, nil
	}
	return groups, nil
}

// Get loads one field group by id.
func (s *Service) Get(ctx context.Context, id string) (*Group, error) {
	if strings.TrimSpace(id) == "" {
		return nil, ErrNotFound
	}
	return s.store.GetByID(ctx, id)
}

// Create stores a new field group.
func (s *Service) Create(ctx context.Context, in Input) (*Group, error) {
	group, err := normalize(in)
	if err != nil {
		return nil, err
	}
	if err := s.ensureNameFree(ctx, group.Name, ""); err != nil {
		return nil, err
	}
	group.ID = s.ids()
	if err := s.store.Create(ctx, group); err != nil {
		return nil, err
	}
	return &group, nil
}

// Update replaces name, texts and fields of an existing group.
func (s *Service) Update(ctx context.Context, id string, in Input) (*Group, error) {
	current, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	group, err := normalize(in)
	if err != nil {
		return nil, err
	}
	if err := s.ensureNameFree(ctx, group.Name, current.ID); err != nil {
		return nil, err
	}
	group.ID = current.ID
	if err := s.store.Update(ctx, group); err != nil {
		return nil, err
	}
	return &group, nil
}

// Delete removes a field group.
func (s *Service) Delete(ctx context.Context, id string) error {
	if _, err := s.Get(ctx, id); err != nil {
		return err
	}
	return s.store.Delete(ctx, id)
}

func (s *Service) ensureNameFree(ctx context.Context, name, exceptID string) error {
	existing, err := s.store.GetByName(ctx, name)
	if errors.Is(err, ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if existing.ID != exceptID {
		return ErrAlreadyExists
	}
	return nil
}

func normalize(in Input) (Group, error) {
	name, err := normalizeName(in.Name)
	if err != nil {
		return Group{}, err
	}
	fields := make([]Field, 0, len(in.Fields))
	seen := map[string]struct{}{}
	for _, field := range in.Fields {
		next, err := normalizeField(field)
		if err != nil {
			return Group{}, err
		}
		key := strings.ToLower(next.Name)
		if _, ok := seen[key]; ok {
			return Group{}, ErrInvalid
		}
		seen[key] = struct{}{}
		fields = append(fields, next)
	}
	return Group{
		Name:        name,
		Labels:      trimText(in.Labels),
		Description: trimText(in.Description),
		Fields:      fields,
	}, nil
}

func normalizeField(field Field) (Field, error) {
	name, err := normalizeName(field.Name)
	if err != nil {
		return Field{}, err
	}
	if !ValidValueType(field.ValueType) {
		return Field{}, ErrInvalid
	}
	return Field{
		Name:        name,
		Labels:      trimText(field.Labels),
		Description: trimText(field.Description),
		ValueType:   field.ValueType,
	}, nil
}

func normalizeName(raw string) (string, error) {
	name := strings.TrimSpace(raw)
	if name == "" || utf8.RuneCountInString(name) > maxNameLen {
		return "", ErrInvalid
	}
	for i, r := range name {
		switch {
		case r >= 'A' && r <= 'Z', r >= 'a' && r <= 'z':
		case i > 0 && (r >= '0' && r <= '9' || r == '_'):
		default:
			return "", ErrInvalid
		}
	}
	return name, nil
}

func trimText(text Text) Text {
	return Text{De: strings.TrimSpace(text.De), En: strings.TrimSpace(text.En)}
}
