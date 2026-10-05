package fieldgroup

import (
	"context"
	"fmt"

	"github.com/samber/do/v2"
)

// Provide wires the field-group service from Store and imports shipped groups.
func Provide(inj do.Injector) error {
	svc := New(do.MustInvokeAs[Store](inj))
	if err := svc.EnsureBuiltin(context.Background()); err != nil {
		return fmt.Errorf("field group catalog: %w", err)
	}
	do.ProvideValue(inj, svc)
	return nil
}
