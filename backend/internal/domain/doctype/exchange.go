package doctype

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/willie68/arcivio/internal/domain/fieldgroup"
	"github.com/willie68/arcivio/internal/domain/models"
)

const (
	kindFieldGroup   = "fieldGroup"
	kindDocumentType = "documentType"
	reasonSameID     = "same-id"
	reasonName       = "name"
	actionOverwrite  = "overwrite"
	actionRename     = "rename"
	actionSkip       = "skip"
)

// Exchange is a JSON catalog of document types and the field groups they use.
// Readonly system groups are omitted. Document types still reference them by id.
type Exchange struct {
	DocumentTypes []ExchangeType
	FieldGroups   []ExchangeGroup
}

// ExchangeType is one document type. FieldGroups lists field-group ids in order.
type ExchangeType struct {
	ID          string
	Name        string
	Labels      Text
	Description Text
	FieldGroups []string
}

// ExchangeGroup is one field group definition.
type ExchangeGroup struct {
	ID          string
	Name        string
	Labels      Text
	Description Text
	Fields      []ExchangeField
}

// ExchangeField is one field inside an exchanged group.
type ExchangeField struct {
	Name        string
	Labels      Text
	Description Text
	ValueType   string
	Mandatory   bool
}

// Change is one differing property, already shortened for the confirmation dialog.
type Change struct {
	Field  string
	Before string
	After  string
}

// Conflict is an import that must be confirmed before it is written.
type Conflict struct {
	Kind          string
	Reason        string
	ID            string
	Exists        bool
	Name          string
	ExistingID    string
	ExistingName  string
	SuggestedName string
	Changes       []Change
}

// Decision answers one conflict. Action is overwrite, rename or skip.
// Name is the technical name used when Action is rename.
type Decision struct {
	Kind   string
	ID     string
	Action string
	Name   string
}

// ImportResult counts what an import created or updated.
type ImportResult struct {
	DocumentTypes int
	FieldGroups   int
}

// Export returns every document type and the field groups it depends on.
// Readonly system groups stay out of FieldGroups.
func (s *Service) Export(ctx context.Context) (Exchange, error) {
	types, err := s.List(ctx)
	if err != nil {
		return Exchange{}, err
	}
	catalog, err := s.groups.List(ctx)
	if err != nil {
		return Exchange{}, err
	}
	byID := indexGroupsByID(catalog)

	included := map[string]fieldgroup.FieldGroup{}
	exportedTypes := make([]ExchangeType, 0, len(types))
	for _, docType := range types {
		for _, id := range docType.FieldGroups {
			group, ok := byID[id]
			if !ok {
				return Exchange{}, ErrUnknownGroup
			}
			if !group.Readonly {
				included[group.ID] = group
			}
		}
		groups := docType.FieldGroups
		if groups == nil {
			groups = []string{}
		}
		exportedTypes = append(exportedTypes, ExchangeType{
			ID:          docType.ID,
			Name:        docType.Name,
			Labels:      docType.Labels,
			Description: docType.Description,
			FieldGroups: append([]string(nil), groups...),
		})
	}
	sort.Slice(exportedTypes, func(i, j int) bool {
		return exportedTypes[i].Name < exportedTypes[j].Name
	})

	exportedGroups := make([]ExchangeGroup, 0, len(included))
	for _, group := range included {
		exportedGroups = append(exportedGroups, toExchangeGroup(group))
	}
	sort.Slice(exportedGroups, func(i, j int) bool {
		return exportedGroups[i].Name < exportedGroups[j].Name
	})
	return Exchange{DocumentTypes: exportedTypes, FieldGroups: exportedGroups}, nil
}

// Preview lists import conflicts. Matching uses the id.
// The same id is reported when the stored record would change.
// A different id with the same technical name is reported as a name clash.
func (s *Service) Preview(ctx context.Context, ex Exchange) ([]Conflict, error) {
	view, err := s.prepare(ctx, ex)
	if err != nil {
		return nil, err
	}
	conflicts := view.conflicts()
	if conflicts == nil {
		conflicts = []Conflict{}
	}
	return conflicts, nil
}

// Import writes a catalog. Conflicts are applied only when a decision says so.
func (s *Service) Import(ctx context.Context, ex Exchange, decisions []Decision) (ImportResult, error) {
	view, err := s.prepare(ctx, ex)
	if err != nil {
		return ImportResult{}, err
	}
	chosen, err := view.resolve(decisions)
	if err != nil {
		return ImportResult{}, err
	}

	groupCount := 0
	for _, group := range view.groups {
		choice, ok := chosen[conflictKey(kindFieldGroup, group.ID)]
		if ok && choice.action == actionSkip {
			continue
		}
		local, exists := view.groupsByID[group.ID]
		if exists && local.Readonly {
			continue
		}
		in := toFieldGroupInput(group)
		if choice.name != "" {
			in.Name = choice.name
		}
		if exists {
			if choice.action == "" {
				continue
			}
			if _, err := s.groups.Update(ctx, group.ID, in); err != nil {
				return ImportResult{}, err
			}
			groupCount++
			continue
		}
		if _, err := s.groups.CreateWithID(ctx, group.ID, in); err != nil {
			return ImportResult{}, err
		}
		groupCount++
	}

	typeCount := 0
	for _, docType := range view.types {
		choice, ok := chosen[conflictKey(kindDocumentType, docType.ID)]
		if ok && choice.action == actionSkip {
			continue
		}
		in := Input{
			Name:        docType.Name,
			Labels:      docType.Labels,
			Description: docType.Description,
			FieldGroups: docType.FieldGroups,
		}
		if choice.name != "" {
			in.Name = choice.name
		}
		if _, exists := view.typesByID[docType.ID]; exists {
			if choice.action == "" {
				continue
			}
			if _, err := s.Update(ctx, docType.ID, in); err != nil {
				return ImportResult{}, err
			}
			typeCount++
			continue
		}
		if _, err := s.createAs(ctx, docType.ID, in); err != nil {
			return ImportResult{}, err
		}
		typeCount++
	}
	return ImportResult{DocumentTypes: typeCount, FieldGroups: groupCount}, nil
}

func (s *Service) createAs(ctx context.Context, id string, in Input) (*Type, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, ErrInvalid
	}
	if _, err := s.Get(ctx, id); err == nil {
		return nil, ErrAlreadyExists
	} else if !errors.Is(err, ErrNotFound) {
		return nil, err
	}
	docType, err := s.normalize(ctx, in)
	if err != nil {
		return nil, err
	}
	if err := s.ensureNameFree(ctx, docType.Name, ""); err != nil {
		return nil, err
	}
	docType.ID = id
	if err := s.store.Create(ctx, docType); err != nil {
		return nil, err
	}
	return &docType, nil
}

type importView struct {
	groups       []ExchangeGroup
	types        []ExchangeType
	groupsByID   map[string]fieldgroup.FieldGroup
	groupsByName map[string]fieldgroup.FieldGroup
	typesByID    map[string]Type
	typesByName  map[string]Type
}

type nameChoice struct {
	action string
	name   string
}

func (s *Service) prepare(ctx context.Context, ex Exchange) (importView, error) {
	groups, err := s.groups.List(ctx)
	if err != nil {
		return importView{}, err
	}
	types, err := s.List(ctx)
	if err != nil {
		return importView{}, err
	}
	view := importView{
		groupsByID:   indexGroupsByID(groups),
		groupsByName: indexGroupsByName(groups),
		typesByID:    indexTypesByID(types),
		typesByName:  indexTypesByName(types),
	}
	incomingGroups, err := normalizeExchangeGroups(ex.FieldGroups, view.groupsByID)
	if err != nil {
		return importView{}, err
	}
	incomingTypes, err := normalizeExchangeTypes(ex.DocumentTypes, view.groupsByID, incomingGroups)
	if err != nil {
		return importView{}, err
	}
	view.groups = incomingGroups
	view.types = incomingTypes
	return view, nil
}

func (v importView) conflicts() []Conflict {
	conflicts := make([]Conflict, 0)
	takenGroups := nameSet(v.groupsByName)
	for _, group := range v.groups {
		takenGroups[strings.ToLower(group.Name)] = struct{}{}
	}
	for _, group := range v.groups {
		local, exists := v.groupsByID[group.ID]
		if exists && local.Readonly {
			continue
		}
		owner, nameTaken := v.groupsByName[strings.ToLower(group.Name)]
		takenByOther := nameTaken && (!exists || owner.ID != group.ID)
		if exists && !takenByOther && sameGroup(local, group) {
			continue
		}
		conflict := Conflict{
			Kind:   kindFieldGroup,
			ID:     group.ID,
			Exists: exists,
			Name:   group.Name,
		}
		if takenByOther {
			conflict.Reason = reasonName
			conflict.ExistingID = owner.ID
			conflict.ExistingName = owner.Name
			conflict.SuggestedName = nextName(group.Name, takenGroups)
			if exists {
				conflict.Changes = groupChanges(local, group)
			} else {
				conflict.Changes = groupChanges(owner, group)
			}
		} else {
			conflict.Reason = reasonSameID
			conflict.ExistingID = local.ID
			conflict.ExistingName = local.Name
			conflict.Changes = groupChanges(local, group)
		}
		conflicts = append(conflicts, conflict)
	}

	takenTypes := nameSet(v.typesByName)
	for _, docType := range v.types {
		takenTypes[strings.ToLower(docType.Name)] = struct{}{}
	}
	for _, docType := range v.types {
		local, exists := v.typesByID[docType.ID]
		owner, nameTaken := v.typesByName[strings.ToLower(docType.Name)]
		takenByOther := nameTaken && (!exists || owner.ID != docType.ID)
		if exists && !takenByOther && sameType(local, docType) {
			continue
		}
		conflict := Conflict{
			Kind:   kindDocumentType,
			ID:     docType.ID,
			Exists: exists,
			Name:   docType.Name,
		}
		if takenByOther {
			conflict.Reason = reasonName
			conflict.ExistingID = owner.ID
			conflict.ExistingName = owner.Name
			conflict.SuggestedName = nextName(docType.Name, takenTypes)
			if exists {
				conflict.Changes = typeChanges(local, docType, v.groupsByID, v.groups)
			} else {
				conflict.Changes = typeChanges(owner, docType, v.groupsByID, v.groups)
			}
		} else {
			conflict.Reason = reasonSameID
			conflict.ExistingID = local.ID
			conflict.ExistingName = local.Name
			conflict.Changes = typeChanges(local, docType, v.groupsByID, v.groups)
		}
		conflicts = append(conflicts, conflict)
	}
	return conflicts
}

func (v importView) resolve(decisions []Decision) (map[string]nameChoice, error) {
	byKey := map[string]Decision{}
	for _, decision := range decisions {
		byKey[conflictKey(decision.Kind, decision.ID)] = decision
	}
	chosen := map[string]nameChoice{}
	usedGroups := map[string]string{}
	for name, group := range v.groupsByName {
		usedGroups[name] = group.ID
	}
	for _, group := range v.groups {
		if _, exists := v.groupsByID[group.ID]; exists {
			continue
		}
		if owner, taken := v.groupsByName[strings.ToLower(group.Name)]; taken && owner.ID != group.ID {
			continue
		}
		usedGroups[strings.ToLower(group.Name)] = group.ID
	}
	usedTypes := map[string]string{}
	for name, docType := range v.typesByName {
		usedTypes[name] = docType.ID
	}
	for _, docType := range v.types {
		if _, exists := v.typesByID[docType.ID]; exists {
			continue
		}
		if owner, taken := v.typesByName[strings.ToLower(docType.Name)]; taken && owner.ID != docType.ID {
			continue
		}
		usedTypes[strings.ToLower(docType.Name)] = docType.ID
	}

	for _, conflict := range v.conflicts() {
		decision, ok := byKey[conflictKey(conflict.Kind, conflict.ID)]
		if !ok || !validAction(conflict, decision.Action) {
			return nil, ErrConflict
		}
		choice := nameChoice{action: decision.Action}
		if decision.Action == actionRename {
			name, err := normalizeName(decision.Name)
			if err != nil {
				return nil, err
			}
			choice.name = name
		} else if decision.Action == actionOverwrite {
			choice.name = conflict.Name
		}
		if choice.name != "" {
			used := usedGroups
			if conflict.Kind == kindDocumentType {
				used = usedTypes
			}
			if owner, taken := used[strings.ToLower(choice.name)]; taken && owner != conflict.ID {
				return nil, ErrAlreadyExists
			}
			used[strings.ToLower(choice.name)] = conflict.ID
		}
		chosen[conflictKey(conflict.Kind, conflict.ID)] = choice
	}
	return chosen, nil
}

func validAction(conflict Conflict, action string) bool {
	switch conflict.Reason {
	case reasonSameID:
		return action == actionOverwrite || action == actionSkip
	case reasonName:
		return action == actionRename || action == actionSkip
	default:
		return false
	}
}

func normalizeExchangeGroups(raw []ExchangeGroup, local map[string]fieldgroup.FieldGroup) ([]ExchangeGroup, error) {
	groups := make([]ExchangeGroup, 0, len(raw))
	seenID := map[string]struct{}{}
	seenName := map[string]struct{}{}
	for _, group := range raw {
		id := strings.TrimSpace(group.ID)
		if id == "" {
			return nil, ErrInvalid
		}
		if _, ok := seenID[id]; ok {
			return nil, ErrInvalid
		}
		seenID[id] = struct{}{}
		if current, ok := local[id]; ok && current.Readonly {
			continue
		}
		name, err := normalizeName(group.Name)
		if err != nil {
			return nil, err
		}
		key := strings.ToLower(name)
		if _, ok := seenName[key]; ok {
			return nil, ErrInvalid
		}
		seenName[key] = struct{}{}
		group.ID = id
		group.Name = name
		group.Labels = trimText(group.Labels)
		group.Description = trimText(group.Description)
		groups = append(groups, group)
	}
	return groups, nil
}

func normalizeExchangeTypes(raw []ExchangeType, local map[string]fieldgroup.FieldGroup, incoming []ExchangeGroup) ([]ExchangeType, error) {
	available := map[string]struct{}{}
	for id := range local {
		available[id] = struct{}{}
	}
	for _, group := range incoming {
		available[group.ID] = struct{}{}
	}
	types := make([]ExchangeType, 0, len(raw))
	seenID := map[string]struct{}{}
	seenName := map[string]struct{}{}
	for _, docType := range raw {
		id := strings.TrimSpace(docType.ID)
		if id == "" {
			return nil, ErrInvalid
		}
		if _, ok := seenID[id]; ok {
			return nil, ErrInvalid
		}
		seenID[id] = struct{}{}
		name, err := normalizeName(docType.Name)
		if err != nil {
			return nil, err
		}
		key := strings.ToLower(name)
		if _, ok := seenName[key]; ok {
			return nil, ErrInvalid
		}
		seenName[key] = struct{}{}
		ids := make([]string, 0, len(docType.FieldGroups))
		hasSystem := false
		used := map[string]struct{}{}
		for _, rawID := range docType.FieldGroups {
			groupID := strings.TrimSpace(rawID)
			if groupID == "" {
				return nil, ErrInvalid
			}
			if _, ok := used[groupID]; ok {
				return nil, ErrInvalid
			}
			used[groupID] = struct{}{}
			if _, ok := available[groupID]; !ok {
				return nil, ErrUnknownGroup
			}
			if groupID == fieldgroup.SystemGroupID {
				hasSystem = true
			}
			ids = append(ids, groupID)
		}
		if !hasSystem {
			return nil, ErrSystemGroup
		}
		types = append(types, ExchangeType{
			ID:          id,
			Name:        name,
			Labels:      trimText(docType.Labels),
			Description: trimText(docType.Description),
			FieldGroups: ids,
		})
	}
	return types, nil
}

func sameGroup(local fieldgroup.FieldGroup, incoming ExchangeGroup) bool {
	return len(groupChanges(local, incoming)) == 0
}

func sameType(local Type, incoming ExchangeType) bool {
	if local.Name != incoming.Name || local.Labels != incoming.Labels || local.Description != incoming.Description {
		return false
	}
	return strings.Join(local.FieldGroups, "\n") == strings.Join(incoming.FieldGroups, "\n")
}

func groupChanges(local fieldgroup.FieldGroup, incoming ExchangeGroup) []Change {
	changes := make([]Change, 0)
	addChange(&changes, "name", local.Name, incoming.Name)
	addChange(&changes, "labelDe", local.Labels.De, incoming.Labels.De)
	addChange(&changes, "labelEn", local.Labels.En, incoming.Labels.En)
	addChange(&changes, "descriptionDe", local.Description.De, incoming.Description.De)
	addChange(&changes, "descriptionEn", local.Description.En, incoming.Description.En)
	addChange(&changes, "fields", summarizeFields(toExchangeGroup(local).Fields), summarizeFields(incoming.Fields))
	return changes
}

func typeChanges(local Type, incoming ExchangeType, groups map[string]fieldgroup.FieldGroup, incomingGroups []ExchangeGroup) []Change {
	names := map[string]string{}
	for id, group := range groups {
		names[id] = group.Name
	}
	for _, group := range incomingGroups {
		names[group.ID] = group.Name
	}
	changes := make([]Change, 0)
	addChange(&changes, "name", local.Name, incoming.Name)
	addChange(&changes, "labelDe", local.Labels.De, incoming.Labels.De)
	addChange(&changes, "labelEn", local.Labels.En, incoming.Labels.En)
	addChange(&changes, "descriptionDe", local.Description.De, incoming.Description.De)
	addChange(&changes, "descriptionEn", local.Description.En, incoming.Description.En)
	addChange(&changes, "fieldGroups", joinGroupNames(local.FieldGroups, names), joinGroupNames(incoming.FieldGroups, names))
	return changes
}

func addChange(changes *[]Change, field, before, after string) {
	if before == after {
		return
	}
	*changes = append(*changes, Change{Field: field, Before: before, After: after})
}

func summarizeFields(fields []ExchangeField) string {
	parts := make([]string, 0, len(fields))
	for _, field := range fields {
		part := field.Name + ":" + field.ValueType
		if field.Mandatory {
			part += "*"
		}
		parts = append(parts, part)
	}
	return strings.Join(parts, ", ")
}

func joinGroupNames(ids []string, names map[string]string) string {
	parts := make([]string, 0, len(ids))
	for _, id := range ids {
		name := names[id]
		if name == "" {
			name = id
		}
		parts = append(parts, name)
	}
	return strings.Join(parts, ", ")
}

func nextName(base string, taken map[string]struct{}) string {
	for n := 1; n < 10000; n++ {
		candidate := fmt.Sprintf("%s (%d)", base, n)
		if _, err := normalizeName(candidate); err != nil {
			return ""
		}
		if _, ok := taken[strings.ToLower(candidate)]; ok {
			continue
		}
		taken[strings.ToLower(candidate)] = struct{}{}
		return candidate
	}
	return ""
}

func nameSet[T any](indexed map[string]T) map[string]struct{} {
	out := make(map[string]struct{}, len(indexed))
	for name := range indexed {
		out[name] = struct{}{}
	}
	return out
}

func conflictKey(kind, id string) string {
	return kind + "\n" + id
}

func indexTypesByID(types []Type) map[string]Type {
	out := make(map[string]Type, len(types))
	for _, docType := range types {
		out[docType.ID] = docType
	}
	return out
}

func indexTypesByName(types []Type) map[string]Type {
	out := make(map[string]Type, len(types))
	for _, docType := range types {
		out[strings.ToLower(docType.Name)] = docType
	}
	return out
}

func indexGroupsByID(groups []fieldgroup.FieldGroup) map[string]fieldgroup.FieldGroup {
	out := make(map[string]fieldgroup.FieldGroup, len(groups))
	for _, group := range groups {
		out[group.ID] = group
	}
	return out
}

func indexGroupsByName(groups []fieldgroup.FieldGroup) map[string]fieldgroup.FieldGroup {
	out := make(map[string]fieldgroup.FieldGroup, len(groups))
	for _, group := range groups {
		out[strings.ToLower(group.Name)] = group
	}
	return out
}

func toExchangeGroup(group fieldgroup.FieldGroup) ExchangeGroup {
	fields := make([]ExchangeField, 0, len(group.Fields))
	for _, field := range group.Fields {
		fields = append(fields, ExchangeField{
			Name:        field.Name,
			Labels:      Text{De: field.Labels.De, En: field.Labels.En},
			Description: Text{De: field.Description.De, En: field.Description.En},
			ValueType:   field.ValueType,
			Mandatory:   field.Mandatory,
		})
	}
	return ExchangeGroup{
		ID:          group.ID,
		Name:        group.Name,
		Labels:      Text{De: group.Labels.De, En: group.Labels.En},
		Description: Text{De: group.Description.De, En: group.Description.En},
		Fields:      fields,
	}
}

func toFieldGroupInput(group ExchangeGroup) fieldgroup.Input {
	fields := make([]fieldgroup.Field, 0, len(group.Fields))
	for _, field := range group.Fields {
		fields = append(fields, fieldgroup.Field{
			Name:        field.Name,
			Labels:      models.LabelText{De: field.Labels.De, En: field.Labels.En},
			Description: models.LabelText{De: field.Description.De, En: field.Description.En},
			ValueType:   field.ValueType,
			Mandatory:   field.Mandatory,
		})
	}
	return fieldgroup.Input{
		Name:        group.Name,
		Labels:      models.LabelText{De: group.Labels.De, En: group.Labels.En},
		Description: models.LabelText{De: group.Description.De, En: group.Description.En},
		Fields:      fields,
	}
}
