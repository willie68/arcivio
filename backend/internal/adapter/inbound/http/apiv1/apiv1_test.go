package apiv1

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
	"github.com/willie68/arcivio/internal/config"
	"github.com/willie68/arcivio/internal/infrastructure/health"
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
	hsvc, err := health.NewHealthSystem(inj, health.Config{Period: 30, StartDelay: 0})
	assert.NoError(t, err)
	do.ProvideValue(inj, hsvc)
	do.ProvideValue(inj, routeServiceName{})

	cfg := config.Config{
		Metrics: config.Metrics{Enable: true},
	}
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
	assert.Equal(t, http.StatusNotFound, rec.Code)

	hr := HealthRoutes(inj, config.Config{Metrics: config.Metrics{Enable: true}, Profiling: config.Profiling{Enable: true}})
	assert.NotNil(t, hr)
}
