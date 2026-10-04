package fieldgroup

import "github.com/samber/do/v2"

// Provide wires the field-group service from Store.
// When a Usage is already registered, deleting a group that a document type
// still references is rejected.
func Provide(inj do.Injector) error {
	svc := New(do.MustInvokeAs[Store](inj))
	if usage, err := do.InvokeAs[Usage](inj); err == nil {
		svc.BindUsage(usage)
	}
	do.ProvideValue(inj, svc)
	return nil
}
