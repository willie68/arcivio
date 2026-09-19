package identity

import (
	"context"
	"fmt"

	"github.com/samber/do/v2"
	"github.com/willie68/arcivio/internal/infrastructure/logging"
)

var logger = logging.New("identity")

// Provide wires the identity service from UserStore and bootstraps the first admin.
func Provide(inj do.Injector) error {
	users := do.MustInvokeAs[UserStore](inj)
	svc := New(users, NewArgon2Hasher())
	created, err := svc.Bootstrap(context.Background())
	if err != nil {
		return fmt.Errorf("identity bootstrap: %w", err)
	}
	if created {
		logger.Warn("created bootstrap user admin with default password; mustChangePassword is set")
	}
	do.ProvideValue(inj, svc)
	return nil
}
