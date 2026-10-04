package fieldgroup

import "github.com/samber/do/v2"

// Provide wires the field-group service from Store.
func Provide(inj do.Injector) error {
	do.ProvideValue(inj, New(do.MustInvokeAs[Store](inj)))
	return nil
}
