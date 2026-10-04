package documenttype

import (
	"github.com/samber/do/v2"
	"github.com/willie68/arcivio/internal/domain/fieldgroup"
)

// Provide wires the document-type service from Store and the field-group catalog.
func Provide(inj do.Injector) error {
	do.ProvideValue(inj, New(do.MustInvokeAs[Store](inj), do.MustInvokeAs[fieldgroup.Store](inj)))
	return nil
}
