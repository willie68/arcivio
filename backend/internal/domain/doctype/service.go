package doctype

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/rs/xid"
	"github.com/willie68/arcivio/internal/domain/fieldgroup"
)

const maxNameLen = 64

// Groups looks up and stores the field groups a document type may reference.
type Groups interface {
	List(ctx context.Context) ([]fieldgroup.Group, error)
	Get(ctx context.Context, id string) (*fieldgroup.Group, error)
	Create(ctx context.Context, in fieldgroup.Input) (*fieldgroup.Group, error)
	CreateWithID(ctx context.Context, id string, in fieldgroup.Input) (*fieldgroup.Group, error)
	Update(ctx context.Context, id string, in fieldgroup.Input) (*fieldgroup.Group, error)
}

// Service is the document-type use case.
type Service struct {
	store  Store
	groups Groups
	ids    func() string
}

// New creates the document-type service.
func New(store Store, groups Groups) *Service {
	return &Service{
		store:  store,
		groups: groups,
		ids:    func() string { return xid.New().String() },
	}
}

// List returns every document type, ordered by name.
func (s *Service) List(ctx context.Context) ([]Type, error) {
	types, err := s.store.List(ctx)
	if err != nil {
		return nil, err
	}
	if types == nil {
		return []Type{}, nil
	}
	return types, nil
}

// Get loads one document type by id.
func (s *Service) Get(ctx context.Context, id string) (*Type, error) {
	if strings.TrimSpace(id) == "" {
		return nil, ErrNotFound
	}
	return s.store.GetByID(ctx, id)
}

// Create stores a new document type. The system field group must be included.
func (s *Service) Create(ctx context.Context, in Input) (*Type, error) {
	docType, err := s.normalize(ctx, in)
	if err != nil {
		return nil, err
	}
	if err := s.ensureNameFree(ctx, docType.Name, ""); err != nil {
		return nil, err
	}
	docType.ID = s.ids()
	if err := s.store.Create(ctx, docType); err != nil {
		return nil, err
	}
	return &docType, nil
}

// Update replaces name, texts and the field-group composition.
func (s *Service) Update(ctx context.Context, id string, in Input) (*Type, error) {
	current, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	docType, err := s.normalize(ctx, in)
	if err != nil {
		return nil, err
	}
	if err := s.ensureNameFree(ctx, docType.Name, current.ID); err != nil {
		return nil, err
	}
	docType.ID = current.ID
	if err := s.store.Update(ctx, docType); err != nil {
		return nil, err
	}
	return &docType, nil
}

// Delete removes a document type.
func (s *Service) Delete(ctx context.Context, id string) error {
	if _, err := s.Get(ctx, id); err != nil {
		return err
	}
	return s.store.Delete(ctx, id)
}

func (s *Service) normalize(ctx context.Context, in Input) (Type, error) {
	name, err := normalizeName(in.Name)
	if err != nil {
		return Type{}, err
	}
	groups, err := s.normalizeGroups(ctx, in.FieldGroups)
	if err != nil {
		return Type{}, err
	}
	return Type{
		Name:        name,
		Labels:      trimText(in.Labels),
		Description: trimText(in.Description),
		FieldGroups: groups,
	}, nil
}

func (s *Service) normalizeGroups(ctx context.Context, ids []string) ([]string, error) {
	groups := make([]string, 0, len(ids))
	seen := map[string]struct{}{}
	hasSystem := false
	for _, raw := range ids {
		id := strings.TrimSpace(raw)
		if id == "" {
			return nil, ErrInvalid
		}
		if _, ok := seen[id]; ok {
			return nil, ErrInvalid
		}
		seen[id] = struct{}{}
		if _, err := s.groups.Get(ctx, id); err != nil {
			if errors.Is(err, fieldgroup.ErrNotFound) {
				return nil, ErrUnknownGroup
			}
			return nil, err
		}
		if id == fieldgroup.SystemGroupID {
			hasSystem = true
		}
		groups = append(groups, id)
	}
	if !hasSystem {
		return nil, ErrSystemGroup
	}
	return groups, nil
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

func normalizeName(raw string) (string, error) {
	name := strings.TrimSpace(raw)
	if name == "" || utf8.RuneCountInString(name) > maxNameLen {
		return "", ErrInvalid
	}
	for i, r := range name {
		switch {
		case r >= 'A' && r <= 'Z', r >= 'a' && r <= 'z':
		case i > 0 && (r >= '0' && r <= '9' || r == '_' || r == ' ' || r == '(' || r == ')'):
		default:
			return "", ErrInvalid
		}
	}
	return name, nil
}

func trimText(text Text) Text {
	return Text{De: strings.TrimSpace(text.De), En: strings.TrimSpace(text.En)}
}
