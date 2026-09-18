package bootstrap

import (
	"github.com/samber/do/v2"
	"github.com/willie68/arcivio/internal/adapter/outbound/store"
	"github.com/willie68/arcivio/internal/config"
	"github.com/willie68/arcivio/internal/domain/document"
	"github.com/willie68/arcivio/internal/infrastructure/health"
	"github.com/willie68/arcivio/internal/infrastructure/logging"
	"github.com/willie68/arcivio/internal/infrastructure/shttp"
)

var (
	logger = logging.New("services")
)

// InitServices initialise the service system
func InitServices(inj do.Injector, cfg config.Config) error {
	logger.Debug("initialise services")
	do.ProvideValue(inj, cfg)
	do.ProvideValue(inj, cfg.Storage)
	do.ProvideValue(inj, cfg.HealthSystem)
	do.ProvideValue(inj, cfg.HTTP)

	if err := InitHelperServices(inj); err != nil {
		return err
	}
	if err := store.Provide(inj); err != nil {
		return err
	}
	if err := document.Provide(inj); err != nil {
		return err
	}
	return InitRESTService(inj)
}

// InitHelperServices initialise the helper services like Healthsystem
func InitHelperServices(inj do.Injector) error {
	return health.Provide(inj)
}

// InitRESTService initialise REST Services
func InitRESTService(inj do.Injector) error {
	return shttp.Provide(inj)
}

// ShutdownServices shutting down all services, that support do.Shutdowner interface
func ShutdownServices(inj do.Injector) {
	inj.Shutdown()
}
