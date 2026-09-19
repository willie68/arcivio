package idp

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"regexp"
)

var verifierRE = regexp.MustCompile(`^[A-Za-z0-9\-._~]{43,128}$`)

const pkceS256 = "S256"

func challengeS256(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func verifyPKCE(verifier, challenge, method string) error {
	if method != pkceS256 {
		return fmt.Errorf("unsupported code_challenge_method")
	}
	if !verifierRE.MatchString(verifier) {
		return fmt.Errorf("invalid code_verifier")
	}
	if challengeS256(verifier) != challenge {
		return fmt.Errorf("code_verifier does not match code_challenge")
	}
	return nil
}

func randomURLToken(nBytes int) (string, error) {
	b := make([]byte, nBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
