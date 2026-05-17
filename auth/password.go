package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Argon2id parameters aligned with OWASP memory-hard password recommendations.
const (
	argon2idVersion = 0x13
	argonTime       = 3
	argonMemoryKiB  = 64 * 1024 // 64 MiB
	argonThreads    = 4
	argonKeyLen     = 32
	saltLen         = 16
)

// ErrInvalidPasswordHash is returned when a stored hash cannot be parsed.
var ErrInvalidPasswordHash = errors.New("invalid password hash encoding")

// HashPassword returns a PHC-style Argon2id string suitable for the users.password_hash column.
func HashPassword(password string) (string, error) {
	if password == "" {
		return "", errors.New("empty password")
	}
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("salt: %w", err)
	}
	key := argon2.IDKey([]byte(password), salt, argonTime, argonMemoryKiB, argonThreads, argonKeyLen)
	encSalt := base64.RawStdEncoding.EncodeToString(salt)
	encHash := base64.RawStdEncoding.EncodeToString(key)
	// $argon2id$v=19$m=65536,t=3,p=4$<salt>$<hash>
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2idVersion, argonMemoryKiB, argonTime, argonThreads, encSalt, encHash), nil
}

// VerifyPassword checks password against a hash produced by HashPassword.
func VerifyPassword(password, encodedHash string) (bool, error) {
	if encodedHash == "" {
		return false, ErrInvalidPasswordHash
	}
	// Legacy / test stub rows (not Argon2id) always fail verification.
	if !strings.HasPrefix(encodedHash, "$argon2id$") {
		return false, nil
	}
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false, ErrInvalidPasswordHash
	}
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2idVersion {
		return false, ErrInvalidPasswordHash
	}
	var mem uint32
	var time uint32
	var threads uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &mem, &time, &threads); err != nil {
		return false, ErrInvalidPasswordHash
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, ErrInvalidPasswordHash
	}
	expected, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, ErrInvalidPasswordHash
	}
	if len(expected) != argonKeyLen {
		return false, ErrInvalidPasswordHash
	}
	computed := argon2.IDKey([]byte(password), salt, uint32(time), uint32(mem), uint8(threads), uint32(len(expected)))
	if subtle.ConstantTimeCompare(computed, expected) == 1 {
		return true, nil
	}
	return false, nil
}
