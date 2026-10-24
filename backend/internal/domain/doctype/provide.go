package doctype

import "github.com/samber/do/v2"

// Provide wires the document-type service from Store and the field-group catalog.
func Provide(inj do.Injector) error {
	do.ProvideValue(inj, New(do.MustInvokeAs[Store](inj), do.MustInvokeAs[Groups](inj)))
	return nil
}
