package apiv1

import (
	"fmt"
	"io/fs"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/render"
	"github.com/samber/do/v2"
	httpSwagger "github.com/swaggo/http-swagger"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/willie68/arcivio/internal/adapter/inbound/http/auth"
	idphandler "github.com/willie68/arcivio/internal/adapter/inbound/http/idp"
	"github.com/willie68/arcivio/internal/config"
	"github.com/willie68/arcivio/internal/domain/idp"
	"github.com/willie68/arcivio/internal/infrastructure/health"
	"github.com/willie68/arcivio/internal/infrastructure/logging"
	"github.com/willie68/arcivio/internal/shared/utils/httputils"
	"github.com/willie68/arcivio/pkg/web"
)

// APIVersion the actual implemented api version
const APIVersion = "1"

// BaseURL is the url all endpoints will be available under
var BaseURL = fmt.Sprintf("/api/v%s", APIVersion)

var logger = logging.New("apiv1")

func token(r *http.Request) (string, error) {
	tk := r.Header.Get("Authorization")
	tk = strings.TrimPrefix(tk, "Bearer ")
	return tk, nil
}

// APIRoutes configuring the api routes for the main REST API
func APIRoutes(inj do.Injector, cfn config.Config) (*chi.Mux, error) {
	logger.Info(fmt.Sprintf("baseurl : %s", BaseURL))
	router := chi.NewRouter()
	setDefaultHandler(router, cfn)

	// building the routes
	router.Route("/", func(r chi.Router) {
		r.Mount(health.NewHealthHandler(inj).Routes())
		if cfn.Metrics.Enable {
			r.Mount("/metrics", promhttp.Handler())
		}
	})
	router.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))
	router.Get("/", serveSPAIndex)
	router.Get("/login", serveSPAIndex)
	router.Get("/callback", serveSPAIndex)
	router.Get("/change-password", serveSPAIndex)
	clientFS, err := fs.Sub(web.WebClientAssets, "client")
	if err != nil {
		return nil, err
	}
	httputils.FileServer(router, "/client", http.FS(clientFS))

	idpProv, err := do.Invoke[*idp.Provider](inj)
	if err != nil {
		return nil, fmt.Errorf("internal idp is required: %w", err)
	}
	router.Mount(idphandler.New(idpProv).Routes())

	var jwtErr error
	router.Route(BaseURL, func(r chi.Router) {
		if strings.EqualFold(cfn.Auth.Type, "jwt") {
			if err := setJWTHandler(r, inj, cfn); err != nil {
				jwtErr = err
				return
			}
		}
		r.Get("/me", newMeHandler(inj).GetMe)
	})
	if jwtErr != nil {
		return nil, jwtErr
	}

	logger.Info(fmt.Sprintf("%s api routes", config.Servicename))

	walkFunc := func(method string, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		logger.Info(fmt.Sprintf("api route: %s %s", method, route))
		return nil
	}

	if err := chi.Walk(router, walkFunc); err != nil {
		logger.Warn(fmt.Sprintf("could not walk api routes. %v", err))
	}
	return router, nil
}

func setJWTHandler(router chi.Router, inj do.Injector, cfn config.Config) error {
	jwtConfig, err := auth.ParseJWTConfig(cfn.Auth)
	if err != nil {
		return err
	}
	logger.Info(fmt.Sprintf("jwt config: %v", jwtConfig))
	jwtAuth := auth.JWTAuth{
		Config: jwtConfig,
	}
	if jwtConfig.Validate {
		prov, err := do.Invoke[*idp.Provider](inj)
		if err != nil {
			return fmt.Errorf("jwt validate requires internal idp: %w", err)
		}
		jwtAuth.Verifier = prov
	}
	router.Use(
		auth.Verifier(&jwtAuth),
		auth.Authenticator,
	)
	return nil
}

func setDefaultHandler(router *chi.Mux, cfn config.Config) {
	router.Use(
		render.SetContentType(render.ContentTypeJSON),
		middleware.Logger,
		middleware.Recoverer,
		cors.Handler(cors.Options{
			// AllowedOrigins: []string{"https://foo.com"}, // Use this to allow specific origin hosts
			AllowedOrigins: []string{"*"},
			// AllowOriginFunc:  func(r *http.Request, origin string) bool { return true },
			AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "X-mcs-username", "X-mcs-password", "X-mcs-profile"},
			ExposedHeaders:   []string{"Link"},
			AllowCredentials: true,
			MaxAge:           300, // Maximum value not ignored by any of major browsers
		}),
	)
	if cfn.OpenTelemetry.Endpoint != "" {
		router.Use(otelMiddleware())
	}
}

// HealthRoutes returning the health routes
func HealthRoutes(inj do.Injector, cfn config.Config) *chi.Mux {
	router := chi.NewRouter()
	router.Use(
		render.SetContentType(render.ContentTypeJSON),
		middleware.Logger,
		middleware.Recoverer,
	)
	if cfn.OpenTelemetry.Endpoint != "" {
		router.Use(otelMiddleware())
	}

	router.Route("/", func(r chi.Router) {
		r.Mount(health.NewHealthHandler(inj).Routes())
		if cfn.Metrics.Enable {
			r.Mount("/metrics", promhttp.Handler())
		}
		if cfn.Profiling.Enable {
			// Define the routes for serving profiling data
			r.Mount("/debug", middleware.Profiler())
		}
	})

	logger.Info("health api routes")
	walkFunc := func(method string, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		logger.Info(fmt.Sprintf("health route: %s %s", method, route))
		return nil
	}
	if err := chi.Walk(router, walkFunc); err != nil {
		logger.Warn(fmt.Sprintf("could not walk health routes. %s", err.Error()))
	}

	return router
}

func serveSPAIndex(w http.ResponseWriter, r *http.Request) {
	data, err := web.WebClientAssets.ReadFile("client/index.html")
	if err != nil {
		http.Error(w, "spa not embedded", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(data)
}

func otelMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			spanName := r.Method + " " + r.URL.Path
			otelhttp.NewHandler(next, spanName).ServeHTTP(w, r)
		})
	}
}
