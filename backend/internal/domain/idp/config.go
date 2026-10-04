package idp

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/willie68/arcivio/internal/config"
)

// Default OIDC client settings for the built-in web app.
const (
	DefaultClientID  = "arcivio-web"
	DefaultLoginPath = "/login"
	cookieName       = "arcivio_auth"
)

// Config is the internal OIDC IdP configuration.
type Config struct {
	Issuer          string
	ClientID        string
	Audience        string
	RedirectURIs    []string
	LoginPath       string
	PrivateKeyFile  string
	PrivateKeyPEM   string
	AccessTokenTTL  time.Duration
	IDTokenTTL      time.Duration
	AuthCodeTTL     time.Duration
	AuthRequestTTL  time.Duration
}

// ConfigFrom maps service config onto IdP settings.
func ConfigFrom(cfg config.Config) Config {
	props := cfg.Auth.Properties
	clientID := optionalString(props, "clientID", DefaultClientID)
	audience := optionalString(props, "audience", clientID)

	issuer := optionalString(props, "issuer", "")
	if issuer == "" {
		base := strings.TrimRight(cfg.HTTP.ServiceURL, "/")
		if base == "" {
			base = "https://localhost"
		}
		issuer = base + "/auth"
	}

	loginPath := optionalString(props, "loginPath", DefaultLoginPath)
	keyFile := optionalString(props, "privateKeyFile", "")
	if keyFile == "" {
		dir := filepath.Dir(cfg.Storage.Path)
		if dir == "." || dir == "" {
			dir = "data"
		}
		keyFile = filepath.Join(dir, "idp", "private.pem")
	}

	c := Config{
		Issuer:         issuer,
		ClientID:       clientID,
		Audience:       audience,
		RedirectURIs:   collectRedirectURIs(cfg, props),
		LoginPath:      loginPath,
		PrivateKeyFile: keyFile,
		PrivateKeyPEM:  optionalString(props, "privateKey", ""),
		AccessTokenTTL: optionalDurationSeconds(props, "accessTokenTTL", time.Hour),
		IDTokenTTL:     optionalDurationSeconds(props, "idTokenTTL", time.Hour),
		AuthCodeTTL:    10 * time.Minute,
		AuthRequestTTL: 10 * time.Minute,
	}
	return c
}

type redirectSet struct {
	seen map[string]struct{}
	out  []string
}

func (s *redirectSet) add(raw string) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return
	}
	if u.Path == "" || u.Path == "/" {
		u.Path = "/callback"
	}
	normalized := u.String()
	if _, ok := s.seen[normalized]; ok {
		return
	}
	s.seen[normalized] = struct{}{}
	s.out = append(s.out, normalized)
}

func addConfiguredRedirects(set *redirectSet, props map[string]any) {
	v, ok := props["redirectURIs"]
	if !ok {
		return
	}
	switch list := v.(type) {
	case []any:
		for _, item := range list {
			if s, ok := item.(string); ok {
				set.add(s)
			}
		}
	case []string:
		for _, s := range list {
			set.add(s)
		}
	}
}

func addHostCallbacks(set *redirectSet, scheme string, port int, hosts []string) {
	for _, host := range hosts {
		set.add(fmt.Sprintf("%s://%s:%d/callback", scheme, host, port))
	}
}

func collectRedirectURIs(cfg config.Config, props map[string]any) []string {
	set := &redirectSet{seen: map[string]struct{}{}}
	if props != nil {
		addConfiguredRedirects(set, props)
	}
	set.add(strings.TrimRight(cfg.HTTP.ServiceURL, "/") + "/callback")
	port := cfg.HTTP.Sslport
	if port <= 0 {
		port = cfg.HTTP.Port
	}
	scheme := "https"
	if cfg.HTTP.Sslport <= 0 {
		scheme = "http"
	}
	addHostCallbacks(set, scheme, port, cfg.HTTP.DNSNames)
	addHostCallbacks(set, scheme, port, cfg.HTTP.IPAddresses)
	addHostCallbacks(set, "http", 5173, []string{"localhost", "127.0.0.1", "[::1]"})
	return set.out
}

func optionalString(props map[string]any, key, def string) string {
	if props == nil {
		return def
	}
	v, ok := props[key]
	if !ok || v == nil {
		return def
	}
	s, ok := v.(string)
	if !ok || s == "" {
		return def
	}
	return s
}

func optionalDurationSeconds(props map[string]any, key string, def time.Duration) time.Duration {
	if props == nil {
		return def
	}
	v, ok := props[key]
	if !ok || v == nil {
		return def
	}
	var n int64
	switch t := v.(type) {
	case int:
		n = int64(t)
	case int64:
		n = t
	default:
		return def
	}
	if n <= 0 {
		return def
	}
	return time.Duration(n) * time.Second
}

// AllowsRedirect reports whether uri is registered or a local Vite dev callback.
func (c Config) AllowsRedirect(uri string) bool {
	uri = strings.TrimSpace(uri)
	for _, allowed := range c.RedirectURIs {
		if allowed == uri {
			return true
		}
	}
	return isLoopbackDevRedirect(uri)
}

// isLoopbackDevRedirect allows the Vite dev server (any port) as OIDC redirect.
func isLoopbackDevRedirect(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	if !strings.EqualFold(u.Scheme, "http") {
		return false
	}
	path := strings.TrimRight(u.Path, "/")
	if path != "/callback" && path != "/client/callback" {
		return false
	}
	switch strings.ToLower(u.Hostname()) {
	case "localhost", "127.0.0.1", "::1":
		return true
	default:
		return false
	}
}
