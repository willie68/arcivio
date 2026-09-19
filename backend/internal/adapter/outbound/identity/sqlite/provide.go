package sqlite

import (
	"github.com/samber/do/v2"
	storesqlite "github.com/willie68/arcivio/internal/adapter/outbound/store/sqlite"
)

// Provide registers identity.UserStore on the shared SQLite database.
func Provide(inj do.Injector) error {
	st := do.MustInvoke[*storesqlite.Store](inj)
	repo, err := New(st.DB())
	if err != nil {
		return err
	}
	do.ProvideValue(inj, repo)
	return nil
}
