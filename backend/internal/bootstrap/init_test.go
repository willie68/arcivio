package bootstrap

import (
	"path/filepath"
	"testing"

	"github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
	"github.com/willie68/arcivio/internal/adapter/outbound/store"
	"github.com/willie68/arcivio/internal/config"
	"github.com/willie68/arcivio/internal/infrastructure/health"
	"github.com/willie68/arcivio/internal/infrastructure/shttp"
)

func TestInitServices(t *testing.T) {
	inj := do.New()
	cfg := config.Config{
		Storage:      store.Config{Type: "sqlite", Path: filepath.Join(t.TempDir(), "t.db")},
		HealthSystem: health.Config{Period: 30, StartDelay: 0},
		HTTP:         shttp.Config{Port: 0, Servicename: "test"},
	}
	assert.NoError(t, InitServices(inj, cfg))
	ShutdownServices(inj)
}
