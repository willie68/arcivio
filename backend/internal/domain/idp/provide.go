package idp

import (
	"fmt"

	"github.com/samber/do/v2"
	"github.com/willie68/arcivio/internal/config"
	"github.com/willie68/arcivio/internal/infrastructure/logging"
)

var logger = logging.New("idp")

// Provide wires the internal OIDC IdP.
func Provide(inj do.Injector) error {
	cfg := do.MustInvoke[config.Config](inj)
	ident := do.MustInvokeAs[identityService](inj)
	prov, err := New(ConfigFrom(cfg), ident)
	if err != nil {
		return fmt.Errorf("idp: %w", err)
	}
	logger.Info(fmt.Sprintf("internal oidc idp issuer=%s client_id=%s redirect_uris=%v", prov.cfg.Issuer, prov.cfg.ClientID, prov.cfg.RedirectURIs))
	do.ProvideValue(inj, prov)
	return nil
}
