package document

import "github.com/samber/do/v2"

// Provide wires Documents from the Store port.
func Provide(inj do.Injector) error {
	do.ProvideValue(inj, NewDocuments(do.MustInvokeAs[Store](inj)))
	return nil
}
