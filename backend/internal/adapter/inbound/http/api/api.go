package api

import (
	"github.com/go-chi/chi/v5"
	"github.com/willie68/arcivio/internal/infrastructure/logging"
)

// MetricsEndpoint endpoint subpath  for metrics
const MetricsEndpoint = "/metrics"

var logger = logging.New("api")

// Handler a http REST interface handler
type Handler interface {
	// Routes get the routes
	Routes() (string, *chi.Mux)
}
