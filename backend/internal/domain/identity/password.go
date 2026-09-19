package identity

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	argon2Time    = 3
	argon2Memory  = 64 * 1024
	argon2Threads = 4
	argon2KeyLen  = 32
	argon2SaltLen = 16
)

// Argon2Hasher implements PasswordHasher with Argon2id (PHC encoded).
type Argon2Hasher struct {
	time    uint32
	memory  uint32
	threads uint8
	keyLen  uint32
	saltLen uint32
}

// NewArgon2Hasher returns a hasher with OWASP-aligned Argon2id parameters.
func NewArgon2Hasher() *Argon2Hasher {
	return &Argon2Hasher{
		time:    argon2Time,
		memory:  argon2Memory,
		threads: argon2Threads,
		keyLen:  argon2KeyLen,
		saltLen: argon2SaltLen,
	}
}

// NewArgon2HasherWithParams is intended for tests with cheaper parameters.
func NewArgon2HasherWithParams(time, memory uint32, threads uint8, keyLen, saltLen uint32) *Argon2Hasher {
	return &Argon2Hasher{
		time:    time,
		memory:  memory,
		threads: threads,
		keyLen:  keyLen,
		saltLen: saltLen,
	}
}

// Hash returns a PHC-encoded Argon2id hash. The password is never stored as given.
func (h *Argon2Hasher) Hash(password string) (string, error) {
	salt := make([]byte, h.saltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate salt: %w", err)
	}
	sum := argon2.IDKey([]byte(password), salt, h.time, h.memory, h.threads, h.keyLen)
	return encodeArgon2(h.time, h.memory, h.threads, salt, sum), nil
}

// Verify checks password against a PHC-encoded Argon2id hash.
func (h *Argon2Hasher) Verify(encodedHash, password string) error {
	time, memory, threads, salt, hash, err := decodeArgon2(encodedHash)
	if err != nil {
		return ErrInvalidCredentials
	}
	sum := argon2.IDKey([]byte(password), salt, time, memory, threads, uint32(len(hash)))
	if subtle.ConstantTimeCompare(hash, sum) != 1 {
		return ErrInvalidCredentials
	}
	return nil
}

func encodeArgon2(time, memory uint32, threads uint8, salt, hash []byte) string {
	b64 := base64.RawStdEncoding
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, memory, time, threads, b64.EncodeToString(salt), b64.EncodeToString(hash))
}

func decodeArgon2(encoded string) (time, memory uint32, threads uint8, salt, hash []byte, err error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return 0, 0, 0, nil, nil, fmt.Errorf("invalid argon2 hash")
	}
	var version int
	if _, err = fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return 0, 0, 0, nil, nil, err
	}
	if version != argon2.Version {
		return 0, 0, 0, nil, nil, fmt.Errorf("unsupported argon2 version")
	}
	if _, err = fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &time, &threads); err != nil {
		return 0, 0, 0, nil, nil, err
	}
	b64 := base64.RawStdEncoding
	salt, err = b64.DecodeString(parts[4])
	if err != nil {
		return 0, 0, 0, nil, nil, err
	}
	hash, err = b64.DecodeString(parts[5])
	if err != nil {
		return 0, 0, 0, nil, nil, err
	}
	return time, memory, threads, salt, hash, nil
}

func randomSecret(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
