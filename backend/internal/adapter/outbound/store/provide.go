package store

import (
	"fmt"

	"github.com/samber/do/v2"
	"github.com/willie68/arcivio/internal/adapter/outbound/store/sqlite"
)

// Provide registers the configured store as document.Store.
func Provide(inj do.Injector) error {
	cfg := do.MustInvoke[Config](inj)
	switch cfg.Type {
	case "sqlite", "":
		st, err := sqlite.New(cfg.Path)
		if err != nil {
			return err
		}
		do.ProvideValue(inj, st)
		return nil
	default:
		return fmt.Errorf("unknown storage type %q", cfg.Type)
	}
}
