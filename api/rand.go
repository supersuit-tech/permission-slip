package api

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
)

// safeCodeCharset is the character set for human-readable codes (invite codes,
// confirmation codes, etc.). It excludes visually ambiguous characters: 0/O and
// 1/I. The length (32) divides evenly into 256, so modular selection from a
// random byte introduces no bias.
const safeCodeCharset = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

// generateRandomBytes returns n cryptographically random bytes.
func generateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}
	return b, nil
}

// generatePrefixedID returns a random hex-encoded ID with the given prefix.
// byteLen controls the number of random bytes (the hex string will be 2×byteLen
// characters long, plus the prefix).
func generatePrefixedID(prefix string, byteLen int) (string, error) {
	b, err := generateRandomBytes(byteLen)
	if err != nil {
		return "", err
	}
	return prefix + hex.EncodeToString(b), nil
}

// generateRandomCode returns n random characters from safeCodeCharset.
func generateRandomCode(n int) (string, error) {
	b, err := generateRandomBytes(n)
	if err != nil {
		return "", err
	}
	code := make([]byte, n)
	for i, v := range b {
		code[i] = safeCodeCharset[int(v)%len(safeCodeCharset)]
	}
	return string(code), nil
}

// hashCodeHex returns a hex-encoded hash of a code string.
// When hmacKey is non-empty, it uses HMAC-SHA256 so that database read access
// alone is not sufficient to reverse the hash. When hmacKey is empty, it falls
// back to plain SHA-256.
func hashCodeHex(code, hmacKey string) string {
	if hmacKey != "" {
		mac := hmac.New(sha256.New, []byte(hmacKey))
		mac.Write([]byte(code))
		return hex.EncodeToString(mac.Sum(nil))
	}
	h := sha256.Sum256([]byte(code))
	return hex.EncodeToString(h[:])
}
