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
func (s *Service) List(ctx context.Context) ([]FieldGroup, error) {
	groups, err := s.store.List(ctx)
	if err != nil {
		return nil, err
	}
	if groups == nil {
		return []FieldGroup{}, nil
	}
	return groups, nil
}

// Get loads one field group by id.
func (s *Service) Get(ctx context.Context, id string) (*FieldGroup, error) {
	if strings.TrimSpace(id) == "" {
		return nil, ErrNotFound
	}
	return s.store.GetByID(ctx, id)
}

// Create stores a new field group.
func (s *Service) Create(ctx context.Context, in Input) (*FieldGroup, error) {
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

// CreateWithID stores a new field group under a given id.
func (s *Service) CreateWithID(ctx context.Context, id string, in Input) (*FieldGroup, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, ErrInvalid
	}
	if _, err := s.Get(ctx, id); err == nil {
		return nil, ErrAlreadyExists
	} else if !errors.Is(err, ErrNotFound) {
		return nil, err
	}
	group, err := normalize(in)
	if err != nil {
		return nil, err
	}
	if err := s.ensureNameFree(ctx, group.Name, ""); err != nil {
		return nil, err
	}
	group.ID = id
	if err := s.store.Create(ctx, group); err != nil {
		return nil, err
	}
	return &group, nil
}

// Update replaces name, texts and fields of an existing group.
func (s *Service) Update(ctx context.Context, id string, in Input) (*FieldGroup, error) {
	current, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if current.Readonly {
		return nil, ErrReadonly
	}
	group, err := normalize(in)
	if err != nil {
		return nil, err
	}
	if err := s.ensureNameFree(ctx, group.Name, current.ID); err != nil {
		return nil, err
	}
	group.ID = current.ID
	group.Readonly = current.Readonly
	if err := s.store.Update(ctx, group); err != nil {
		return nil, err
	}
	return &group, nil
}

// Delete removes a field group.
func (s *Service) Delete(ctx context.Context, id string) error {
	current, err := s.Get(ctx, id)
	if err != nil {
		return err
	}
	if current.Readonly {
		return ErrReadonly
	}
	return s.store.Delete(ctx, id)
}

// EnsureBuiltin stores each shipped field group that is not already present.
func (s *Service) EnsureBuiltin(ctx context.Context) error {
	shipped, err := builtinGroups()
	if err != nil {
		return err
	}
	for _, group := range shipped {
		_, err := s.store.GetByID(ctx, group.ID)
		if err == nil {
			continue
		}
		if !errors.Is(err, ErrNotFound) {
			return err
		}
		if _, err := s.store.GetByName(ctx, group.Name); err == nil {
			return ErrAlreadyExists
		} else if !errors.Is(err, ErrNotFound) {
			return err
		}
		group.Readonly = true
		if err := s.store.Create(ctx, group); err != nil {
			return err
		}
	}
	return nil
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

func normalize(in Input) (FieldGroup, error) {
	name, err := normalizeCatalogName(in.Name)
	if err != nil {
		return FieldGroup{}, err
	}
	fields := make([]Field, 0, len(in.Fields))
	seen := map[string]struct{}{}
	for _, field := range in.Fields {
		next, err := normalizeField(field)
		if err != nil {
			return FieldGroup{}, err
		}
		key := strings.ToLower(next.Name)
		if _, ok := seen[key]; ok {
			return FieldGroup{}, ErrInvalid
		}
		seen[key] = struct{}{}
		fields = append(fields, next)
	}
	return FieldGroup{
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
		Mandatory:   field.Mandatory,
	}, nil
}

func normalizeName(raw string) (string, error) {
	return normalizeLabeledName(raw, false)
}

func normalizeCatalogName(raw string) (string, error) {
	return normalizeLabeledName(raw, true)
}

func normalizeLabeledName(raw string, catalog bool) (string, error) {
	name := strings.TrimSpace(raw)
	if name == "" || utf8.RuneCountInString(name) > maxNameLen {
		return "", ErrInvalid
	}
	for i, r := range name {
		switch {
		case r >= 'A' && r <= 'Z', r >= 'a' && r <= 'z':
		case i > 0 && (r >= '0' && r <= '9' || r == '_'):
		case catalog && i > 0 && (r == ' ' || r == '(' || r == ')'):
		default:
			return "", ErrInvalid
		}
	}
	return name, nil
}

func trimText(text Text) Text {
	return Text{De: strings.TrimSpace(text.De), En: strings.TrimSpace(text.En)}
}
