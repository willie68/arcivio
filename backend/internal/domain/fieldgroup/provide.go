package fieldgroup

import (
	"context"
	"fmt"

	"github.com/samber/do/v2"
)

// Provide wires the field-group service from Store and imports shipped groups.
// When a Usage is already registered, deleting a group that a document type
// still references is rejected.
func Provide(inj do.Injector) error {
	svc := New(do.MustInvokeAs[Store](inj))
	if err := svc.EnsureBuiltin(context.Background()); err != nil {
		return fmt.Errorf("field group catalog: %w", err)
	}
	if usage, err := do.InvokeAs[Usage](inj); err == nil {
		svc.BindUsage(usage)
	}
	do.ProvideValue(inj, svc)
	return nil
}
