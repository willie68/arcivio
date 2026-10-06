package fieldgroup

import (
	"encoding/json"
	"sort"

	_ "embed"
)

//go:embed fieldgroups.json
var builtinCatalog []byte

func builtinGroups() ([]FieldGroup, error) {
	var catalog map[string]FieldGroup
	if err := json.Unmarshal(builtinCatalog, &catalog); err != nil {
		return nil, err
	}
	groups := make([]FieldGroup, 0, len(catalog))
	for _, group := range catalog {
		group.Readonly = true
		groups = append(groups, group)
	}
	sort.Slice(groups, func(i, j int) bool {
		return groups[i].Name < groups[j].Name
	})
	return groups, nil
}
