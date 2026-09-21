package apiv1

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
	"github.com/willie68/arcivio/internal/adapter/outbound/store"
	"github.com/willie68/arcivio/internal/bootstrap"
	"github.com/willie68/arcivio/internal/config"
	"github.com/willie68/arcivio/internal/infrastructure/health"
	"github.com/willie68/arcivio/internal/infrastructure/shttp"
)

type routeServiceName struct{}

func (routeServiceName) ServiceName() string { return "arcivio" }

func TestToken(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer abc")
	tk, err := token(req)
	assert.NoError(t, err)
	assert.Equal(t, "abc", tk)
}

func TestAPIRoutesAndHealthRoutes(t *testing.T) {
	inj := do.New()
	cfg := config.Config{
		Storage:      store.Config{Type: "sqlite", Path: filepath.Join(t.TempDir(), "t.db")},
		HealthSystem: health.Config{Period: 30, StartDelay: 0},
		HTTP: shttp.Config{
			Port:        0,
			Servicename: "arcivio",
			ServiceURL:  "https://127.0.0.1:9443",
			DNSNames:    []string{"localhost"},
			IPAddresses: []string{"127.0.0.1"},
			Sslport:     9443,
		},
		Auth: config.Authentication{
			Type:       "jwt",
			Properties: map[string]any{"validate": true},
		},
		Metrics: config.Metrics{Enable: true},
	}
	assert.NoError(t, bootstrap.InitServices(inj, cfg))
	do.ProvideValue(inj, routeServiceName{})
	t.Cleanup(func() { bootstrap.ShutdownServices(inj) })

	router, err := APIRoutes(inj, cfg)
	assert.NoError(t, err)
	assert.NotNil(t, router)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/livez", nil)
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "Arcivio")

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/client/", nil)
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "Arcivio")

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/addresses/", nil)
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/me/password", nil)
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/auth/.well-known/openid-configuration", nil)
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"code_challenge_methods_supported"`)

	hr := HealthRoutes(inj, config.Config{Metrics: config.Metrics{Enable: true}, Profiling: config.Profiling{Enable: true}})
	assert.NotNil(t, hr)
}
