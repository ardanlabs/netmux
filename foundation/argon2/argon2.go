package argon2

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Set of error variables.
var (
	ErrInvalidHash         = errors.New("the encoded hash is not in the correct format")
	ErrIncompatibleVersion = errors.New("incompatible version of argon2")
)

const (
	saltLength  = 16
	keyLength   = 32
	memory      = 64 * 1024
	iterations  = 3
	parallelism = 2
)

// GenerateHash takes a string value and encodes that value into our own
// unique format.
func GenerateHash(value string) (string, error) {
	salt, err := generateRandomBytes(saltLength)
	if err != nil {
		return "", fmt.Errorf("generateRandomBytes: %w", err)
	}

	key := argon2.IDKey([]byte(value), salt, iterations, memory, parallelism, keyLength)

	encodedSalt := base64.RawStdEncoding.EncodeToString(salt)
	encodedKey := base64.RawStdEncoding.EncodeToString(key)

	hash := fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s", argon2.Version, memory, iterations, parallelism, encodedSalt, encodedKey)

	return hash, nil
}

// DecodeAndCompare takes the value that was encoded and the hash that our
// algorithm produced and validates the hash was derived from the value.
func DecodeAndCompare(value string, hash string) (match bool, err error) {
	salt, orgKey, err := decodeHash(hash)
	if err != nil {
		return false, fmt.Errorf("decodeHash: %w", err)
	}

	newKey := argon2.IDKey([]byte(value), salt, iterations, memory, parallelism, keyLength)

	// We are using the subtle.ConstantTimeCompare() function for this
	// to help prevent timing attacks.
	if subtle.ConstantTimeCompare(orgKey, newKey) == 1 {
		return true, nil
	}

	return false, nil
}

// =============================================================================

func generateRandomBytes(n uint32) ([]byte, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return nil, fmt.Errorf("rand.Read: %w", err)
	}

	return b, nil
}

func decodeHash(encodedHash string) (salt []byte, key []byte, err error) {
	vals := strings.Split(encodedHash, "$")
	if len(vals) != 6 {
		return nil, nil, ErrInvalidHash
	}

	var version int
	if _, err := fmt.Sscanf(vals[2], "v=%d", &version); err != nil {
		return nil, nil, fmt.Errorf("fmt.Sscanf: %w", err)
	}

	if version != argon2.Version {
		return nil, nil, ErrIncompatibleVersion
	}

	var cfg struct {
		memory      uint32
		iterations  uint32
		parallelism uint8
	}
	if _, err := fmt.Sscanf(vals[3], "m=%d,t=%d,p=%d", &cfg.memory, &cfg.iterations, &cfg.parallelism); err != nil {
		return nil, nil, fmt.Errorf("fmt.Sscanf: %w", err)
	}

	if cfg.memory != memory {
		return nil, nil, ErrInvalidHash
	}

	if cfg.iterations != iterations {
		return nil, nil, ErrInvalidHash
	}

	if cfg.parallelism != parallelism {
		return nil, nil, ErrInvalidHash
	}

	salt, err = base64.RawStdEncoding.Strict().DecodeString(vals[4])
	if err != nil {
		return nil, nil, err
	}

	key, err = base64.RawStdEncoding.Strict().DecodeString(vals[5])
	if err != nil {
		return nil, nil, err
	}

	return salt, key, nil
}
