package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
)

const refreshTokenRandLen = 32

// NewOpaqueRefreshToken returns a URL-safe opaque refresh token (plaintext for the client).
func NewOpaqueRefreshToken() (string, error) {
	b := make([]byte, refreshTokenRandLen)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("refresh entropy: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// HashRefreshToken returns a deterministic hex-encoded SHA-256 of the plaintext refresh token.
func HashRefreshToken(plaintext string) string {
	sum := sha256.Sum256([]byte(plaintext))
	return hex.EncodeToString(sum[:])
}
