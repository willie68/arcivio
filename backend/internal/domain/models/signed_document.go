package models

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"

	"github.com/gowebpki/jcs"
)

const (
	// SigAlgRS256 is RSA PKCS#1 v1.5 with SHA-256 over RFC 8785 canonical JSON.
	SigAlgRS256 = "RS256"
)

var (
	ErrNilDocument      = errors.New("document is nil")
	ErrMissingKeyID     = errors.New("missing key id")
	ErrInvalidKey       = errors.New("invalid key")
	ErrUnsupportedKey   = errors.New("unsupported key type")
	ErrMissingSignature = errors.New("missing document signature")
	ErrInvalidSignature = errors.New("invalid document signature")
)

// PublicKeyLookup resolves a PEM-encoded verify key (public key, certificate, or private key) by key id.
type PublicKeyLookup func(keyID string) (publicKeyPEM string, err error)

// DocumentSignature is the RFC 8785 / RS256 sidecar of a SignedDocument.
type DocumentSignature struct {
	Algorithm string `json:"alg"`
	KeyID     string `json:"keyId"`
	Signature string `json:"signature"`
}

// SignedDocument is a Document plus a detached signature over its canonical JSON.
// JSON of a SignedDocument unmarshals into Document (unknown field signature is ignored).
type SignedDocument struct {
	Document
	Signature DocumentSignature `json:"signature"`
}

// Sign signs the document with a PEM-encoded RSA private key.
func (d *Document) Sign(keyID, privateKeyPEM string) (*SignedDocument, error) {
	return SignDocument(d, keyID, privateKeyPEM)
}

// SignDocument canonicalizes doc (RFC 8785) and signs it with the given RSA private key PEM.
func SignDocument(doc *Document, keyID, privateKeyPEM string) (*SignedDocument, error) {
	if doc == nil {
		return nil, ErrNilDocument
	}
	if keyID == "" {
		return nil, ErrMissingKeyID
	}
	key, err := parseRSAPrivateKey(privateKeyPEM)
	if err != nil {
		return nil, err
	}
	canonical, err := CanonicalDocumentJSON(doc)
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(canonical)
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, sum[:])
	if err != nil {
		return nil, fmt.Errorf("sign document: %w", err)
	}
	return &SignedDocument{
		Document: *doc,
		Signature: DocumentSignature{
			Algorithm: SigAlgRS256,
			KeyID:     keyID,
			Signature: base64.RawURLEncoding.EncodeToString(sig),
		},
	}, nil
}

// GetKeyID returns the id of the key that produced the signature.
func (d *SignedDocument) GetKeyID() string {
	if d == nil {
		return ""
	}
	return d.Signature.KeyID
}

// Check verifies the signature with a PEM-encoded RSA public key, certificate, or private key.
func (d *SignedDocument) Check(keyPEM string) error {
	if d == nil || d.Signature.Signature == "" {
		return ErrMissingSignature
	}
	if d.Signature.Algorithm != "" && d.Signature.Algorithm != SigAlgRS256 {
		return fmt.Errorf("%w: %s", ErrInvalidSignature, d.Signature.Algorithm)
	}
	pub, err := parseRSAPublicKey(keyPEM)
	if err != nil {
		return err
	}
	canonical, err := CanonicalDocumentJSON(&d.Document)
	if err != nil {
		return err
	}
	sig, err := base64.RawURLEncoding.DecodeString(d.Signature.Signature)
	if err != nil {
		return ErrInvalidSignature
	}
	sum := sha256.Sum256(canonical)
	if err := rsa.VerifyPKCS1v15(pub, crypto.SHA256, sum[:], sig); err != nil {
		return ErrInvalidSignature
	}
	return nil
}

// CheckWith looks up the verify key by GetKeyID and then checks the signature.
func (d *SignedDocument) CheckWith(lookup PublicKeyLookup) error {
	if lookup == nil {
		return ErrInvalidKey
	}
	keyPEM, err := lookup(d.GetKeyID())
	if err != nil {
		return err
	}
	return d.Check(keyPEM)
}

// CanonicalDocumentJSON returns the RFC 8785 canonical JSON of the document body (no signature).
func CanonicalDocumentJSON(doc *Document) ([]byte, error) {
	if doc == nil {
		return nil, ErrNilDocument
	}
	raw, err := json.Marshal(doc)
	if err != nil {
		return nil, err
	}
	canonical, err := jcs.Transform(raw)
	if err != nil {
		return nil, fmt.Errorf("canonicalize document json: %w", err)
	}
	return canonical, nil
}

func parseRSAPrivateKey(pemStr string) (*rsa.PrivateKey, error) {
	block, err := decodePEM(pemStr)
	if err != nil {
		return nil, err
	}
	if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		rsaKey, ok := key.(*rsa.PrivateKey)
		if !ok {
			return nil, ErrUnsupportedKey
		}
		return rsaKey, nil
	}
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidKey, err)
	}
	return key, nil
}

func parseRSAPublicKey(pemStr string) (*rsa.PublicKey, error) {
	block, err := decodePEM(pemStr)
	if err != nil {
		return nil, err
	}
	switch block.Type {
	case "CERTIFICATE":
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrInvalidKey, err)
		}
		pub, ok := cert.PublicKey.(*rsa.PublicKey)
		if !ok {
			return nil, ErrUnsupportedKey
		}
		return pub, nil
	case "PRIVATE KEY", "RSA PRIVATE KEY":
		priv, err := parseRSAPrivateKey(pemStr)
		if err != nil {
			return nil, err
		}
		return &priv.PublicKey, nil
	}
	if pub, err := x509.ParsePKIXPublicKey(block.Bytes); err == nil {
		rsaPub, ok := pub.(*rsa.PublicKey)
		if !ok {
			return nil, ErrUnsupportedKey
		}
		return rsaPub, nil
	}
	if pub, err := x509.ParsePKCS1PublicKey(block.Bytes); err == nil {
		return pub, nil
	}
	return nil, ErrInvalidKey
}

func decodePEM(pemStr string) (*pem.Block, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, ErrInvalidKey
	}
	return block, nil
}
