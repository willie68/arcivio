package roles

import "github.com/samber/do/v2"

// Provide registers the built-in role catalog.
func Provide(inj do.Injector) error {
	do.ProvideValue(inj, NewService())
	return nil
}
