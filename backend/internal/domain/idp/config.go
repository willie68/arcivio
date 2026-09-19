package idp

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/willie68/arcivio/internal/config"
)

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

func collectRedirectURIs(cfg config.Config, props map[string]any) []string {
	seen := make(map[string]struct{})
	var out []string
	add := func(raw string) {
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
		s := u.String()
		if _, ok := seen[s]; ok {
			return
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}

	if v, ok := props["redirectURIs"]; ok {
		switch list := v.(type) {
		case []any:
			for _, item := range list {
				if s, ok := item.(string); ok {
					add(s)
				}
			}
		case []string:
			for _, s := range list {
				add(s)
			}
		}
	}

	add(strings.TrimRight(cfg.HTTP.ServiceURL, "/") + "/callback")
	port := cfg.HTTP.Sslport
	if port <= 0 {
		port = cfg.HTTP.Port
	}
	scheme := "https"
	if cfg.HTTP.Sslport <= 0 {
		scheme = "http"
	}
	for _, host := range cfg.HTTP.DNSNames {
		add(fmt.Sprintf("%s://%s:%d/callback", scheme, host, port))
	}
	for _, ip := range cfg.HTTP.IPAddresses {
		add(fmt.Sprintf("%s://%s:%d/callback", scheme, ip, port))
	}
	for _, host := range []string{"localhost", "127.0.0.1", "[::1]"} {
		add(fmt.Sprintf("http://%s:5173/callback", host))
	}
	return out
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
