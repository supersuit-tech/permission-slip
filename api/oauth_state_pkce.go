// AES-256-GCM helpers for sealing the PKCE code verifier inside the OAuth state JWT.
// JWTs are signed but not encrypted; without this, the verifier would be readable from the base64url-encoded payload.
package api

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
)

// oauthStatePKCENonceSize is the GCM nonce length for sealing PKCE verifiers in state JWTs.
const oauthStatePKCENonceSize = 12

func oauthStateAEAD(secret string) (cipher.AEAD, error) {
	sum := sha256.Sum256([]byte(secret))
	block, err := aes.NewCipher(sum[:])
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

// sealOAuthStatePKCE encrypts the PKCE verifier for storage inside the signed state JWT.
// The JWT is only signed (not encrypted), so the verifier must not appear in plaintext in the payload.
func sealOAuthStatePKCE(secret, verifier string) (string, error) {
	if verifier == "" {
		return "", nil
	}
	aead, err := oauthStateAEAD(secret)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, oauthStatePKCENonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	sealed := aead.Seal(nonce, nonce, []byte(verifier), nil)
	return base64.RawURLEncoding.EncodeToString(sealed), nil
}

func openOAuthStatePKCE(secret, encoded string) (string, error) {
	if encoded == "" {
		return "", nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("decode pkce blob: %w", err)
	}
	if len(raw) < oauthStatePKCENonceSize {
		return "", fmt.Errorf("pkce blob too short")
	}
	aead, err := oauthStateAEAD(secret)
	if err != nil {
		return "", err
	}
	nonce := raw[:oauthStatePKCENonceSize]
	ciphertext := raw[oauthStatePKCENonceSize:]
	plain, err := aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt pkce blob: %w", err)
	}
	return string(plain), nil
}
