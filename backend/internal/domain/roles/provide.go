package roles

import "github.com/samber/do/v2"

func Provide(inj do.Injector) error {
	do.ProvideValue(inj, NewService())
	return nil
}
