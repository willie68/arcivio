package idp

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func loadOrCreateKey(cfg Config) (*rsa.PrivateKey, error) {
	if pemData := strings.TrimSpace(cfg.PrivateKeyPEM); pemData != "" {
		return parseRSAPrivateKey([]byte(pemData))
	}
	if cfg.PrivateKeyFile != "" {
		data, err := os.ReadFile(cfg.PrivateKeyFile)
		if err == nil {
			return parseRSAPrivateKey(data)
		}
		if !os.IsNotExist(err) {
			return nil, err
		}
		key, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			return nil, err
		}
		if err := writePrivateKey(cfg.PrivateKeyFile, key); err != nil {
			return nil, err
		}
		return key, nil
	}
	return rsa.GenerateKey(rand.Reader, 2048)
}

func writePrivateKey(path string, key *rsa.PrivateKey) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return err
	}
	block := &pem.Block{Type: "PRIVATE KEY", Bytes: der}
	return os.WriteFile(path, pem.EncodeToMemory(block), 0o600)
}

func parseRSAPrivateKey(pemBytes []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, fmt.Errorf("invalid private key pem")
	}
	if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		rsaKey, ok := key.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("private key is not RSA")
		}
		return rsaKey, nil
	}
	return x509.ParsePKCS1PrivateKey(block.Bytes)
}

func keyID(pub *rsa.PublicKey) string {
	sum := sha256.Sum256(pub.N.Bytes())
	return base64.RawURLEncoding.EncodeToString(sum[:8])
}

func jwksFor(pub *rsa.PublicKey, kid string) map[string]any {
	e := big.NewInt(int64(pub.E)).Bytes()
	return map[string]any{
		"keys": []map[string]string{
			{
				"kty": "RSA",
				"use": "sig",
				"alg": "RS256",
				"kid": kid,
				"n":   base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
				"e":   base64.RawURLEncoding.EncodeToString(e),
			},
		},
	}
}

func signRS256(key *rsa.PrivateKey, kid string, claims map[string]any) (string, error) {
	header := map[string]any{"alg": "RS256", "typ": "JWT", "kid": kid}
	hJSON, err := json.Marshal(header)
	if err != nil {
		return "", err
	}
	pJSON, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	enc := base64.RawURLEncoding.EncodeToString
	signingInput := enc(hJSON) + "." + enc(pJSON)
	sum := sha256.Sum256([]byte(signingInput))
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, sum[:])
	if err != nil {
		return "", err
	}
	return signingInput + "." + enc(sig), nil
}

func verifyRS256(pub *rsa.PublicKey, token string) (map[string]any, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("malformed jwt")
	}
	headerJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, fmt.Errorf("jwt header: %w", err)
	}
	var header map[string]any
	if err := json.Unmarshal(headerJSON, &header); err != nil {
		return nil, fmt.Errorf("jwt header: %w", err)
	}
	if alg, _ := header["alg"].(string); alg != "RS256" {
		return nil, fmt.Errorf("unexpected jwt alg")
	}
	payloadJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("jwt payload: %w", err)
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, fmt.Errorf("jwt signature: %w", err)
	}
	sum := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	if err := rsa.VerifyPKCS1v15(pub, crypto.SHA256, sum[:], sig); err != nil {
		return nil, fmt.Errorf("invalid jwt signature")
	}
	var claims map[string]any
	if err := json.Unmarshal(payloadJSON, &claims); err != nil {
		return nil, fmt.Errorf("jwt payload: %w", err)
	}
	return claims, nil
}

func unixTime(t time.Time) int64 {
	return t.Unix()
}

func claimString(claims map[string]any, key string) string {
	v, _ := claims[key].(string)
	return v
}

func claimTime(v any) (time.Time, bool) {
	switch t := v.(type) {
	case float64:
		return time.Unix(int64(t), 0), true
	case int64:
		return time.Unix(t, 0), true
	case int:
		return time.Unix(int64(t), 0), true
	case json.Number:
		n, err := t.Int64()
		if err != nil {
			return time.Time{}, false
		}
		return time.Unix(n, 0), true
	default:
		return time.Time{}, false
	}
}
