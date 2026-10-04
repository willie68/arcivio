package shttp

import (
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
)

func TestProvideAndLifecycle(t *testing.T) {
	inj := do.New()
	do.ProvideValue(inj, Config{
		Servicename: "coverage",
		Port:        0,
		ServiceURL:  "http://127.0.0.1",
	})
	assert.NoError(t, Provide(inj))
	sh := do.MustInvoke[*SHttp](inj)
	assert.False(t, sh.Started())

	router := chi.NewRouter()
	router.Get("/", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	sh.StartServers(router, router)
	time.Sleep(50 * time.Millisecond)
	assert.True(t, sh.Started())
	sh.ShutdownServers()
	assert.False(t, sh.Started())
}

func TestStartHTTPSGeneratedCert(t *testing.T) {
	sh, err := NewSHttp(Config{
		Servicename: "coverage",
		Port:        0,
		Sslport:     0,
		ServiceURL:  "https://localhost",
	})
	assert.NoError(t, err)
	assert.False(t, sh.useSSL)

	sh, err = NewSHttp(Config{
		Servicename: "coverage",
		Port:        0,
		Sslport:     18443,
		ServiceURL:  "https://localhost",
	})
	assert.NoError(t, err)
	assert.True(t, sh.useSSL)
	router := chi.NewRouter()
	sh.StartServers(router, router)
	time.Sleep(80 * time.Millisecond)
	assert.True(t, sh.Started())
	sh.ShutdownServers()
}

func TestTLSGenerationAndFiles(t *testing.T) {
	gc := &generateCertificate{
		ServiceName: "arcivio",
		Host:        "localhost,127.0.0.1",
		IPs:         []string{"127.0.0.1", "not-an-ip"},
		DNSnames:    []string{"arcivio.local"},
		ValidFor:    time.Hour,
		EcdsaCurve:  "P256",
		IsCA:        true,
	}
	cfg, err := gc.GenerateTLSConfig()
	assert.NoError(t, err)
	assert.NotEmpty(t, cfg.Certificates)

	gc.EcdsaCurve = ""
	gc.Ed25519Key = true
	_, err = gc.GenerateTLSConfig()
	assert.NoError(t, err)

	gc.Ed25519Key = false
	gc.ValidFrom = "not-a-date"
	_, err = gc.GenerateTLSConfig()
	assert.Error(t, err)

	gc.ValidFrom = ""
	gc.EcdsaCurve = "nope"
	bad, _ := gc.GenerateTLSConfig()
	assert.Nil(t, bad)
	assert.Nil(t, gc.publicKey("not-a-key"))

	dir := t.TempDir()
	certFile := filepath.Join(dir, "cert.pem")
	keyFile := filepath.Join(dir, "key.pem")
	sh := &SHttp{cfn: Config{Certificate: certFile, Key: keyFile}}
	_, err = sh.TLSFromFiles()
	assert.Error(t, err)

	pair, err := (&generateCertificate{ServiceName: "arcivio", Host: "localhost", ValidFor: time.Hour, EcdsaCurve: "P256"}).GenerateTLSConfig()
	assert.NoError(t, err)
	keyBytes, err := x509.MarshalPKCS8PrivateKey(pair.Certificates[0].PrivateKey)
	assert.NoError(t, err)
	assert.NoError(t, os.WriteFile(certFile, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: pair.Certificates[0].Certificate[0]}), 0o600))
	assert.NoError(t, os.WriteFile(keyFile, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyBytes}), 0o600))
	loaded, err := sh.TLSFromFiles()
	assert.NoError(t, err)
	assert.NotEmpty(t, loaded.Certificates)
}
