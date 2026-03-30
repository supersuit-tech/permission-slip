// Package webpush implements the Web Push notification channel using VAPID
// authentication. It provides VAPID key management (env var requirement in
// production, auto-generation with database persistence in development) and
// a notify.Sender that delivers push messages to all of a user's registered
// browser subscriptions.
package webpush

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"strings"

	wplib "github.com/SherClockHolmes/webpush-go"
	"github.com/supersuit-tech/permission-slip/db"
)

// VAPID key storage keys in the server_config table.
const (
	configKeyVAPIDPublicKey  = "vapid_public_key"
	configKeyVAPIDPrivateKey = "vapid_private_key"
)

// VAPIDKeys holds a VAPID key pair for Web Push authentication.
type VAPIDKeys struct {
	PublicKey  string
	PrivateKey string
}

// PublicKeyFingerprint returns a short SHA-256 fingerprint of the public key
// for log messages. Operators can use this to verify the correct key is loaded
// without exposing the full key.
func (k *VAPIDKeys) PublicKeyFingerprint() string {
	h := sha256.Sum256([]byte(k.PublicKey))
	return fmt.Sprintf("%x", h[:4])
}

// InitVAPIDKeys loads or generates VAPID keys.
//
// In production (devMode=false):
//   - Both env vars set → validates and uses them
//   - Neither set → returns nil (Web Push disabled)
//   - Only one set → returns error (misconfiguration)
//
// In development (devMode=true), keys are resolved with this priority:
//  1. Environment variables (VAPID_PUBLIC_KEY, VAPID_PRIVATE_KEY)
//  2. Database (server_config table) — previously auto-generated
//  3. Generate new keys and persist to database
//
// Returns (nil, nil) when Web Push is not configured.
func InitVAPIDKeys(ctx context.Context, d db.DBTX, devMode bool) (*VAPIDKeys, error) {
	// Priority 1: Environment variables (both modes).
	// Trim whitespace to handle common copy-paste issues (trailing newlines, etc.).
	pubKey := strings.TrimSpace(os.Getenv("VAPID_PUBLIC_KEY"))
	privKey := strings.TrimSpace(os.Getenv("VAPID_PRIVATE_KEY"))
	if pubKey != "" && privKey != "" {
		if err := validateVAPIDKeyFormat(pubKey, privKey); err != nil {
			return nil, fmt.Errorf("invalid VAPID keys from environment: %w", err)
		}
		keys := &VAPIDKeys{PublicKey: pubKey, PrivateKey: privKey}
		log.Printf("Web Push: using VAPID keys from environment variables (fingerprint: %s)", keys.PublicKeyFingerprint())
		return keys, nil
	}

	// In production, do not fall through to DB or auto-generation.
	// If neither key is set, Web Push is simply not configured (return nil).
	// If only one is set, that's a misconfiguration.
	if !devMode {
		if pubKey == "" && privKey == "" {
			log.Println("Web Push: disabled (VAPID_PUBLIC_KEY and VAPID_PRIVATE_KEY not set)")
			return nil, nil
		}
		// Partial configuration — one key set but not the other.
		if pubKey == "" {
			return nil, fmt.Errorf("VAPID_PUBLIC_KEY is not set but VAPID_PRIVATE_KEY is; both are required for Web Push")
		}
		return nil, fmt.Errorf("VAPID_PRIVATE_KEY is not set but VAPID_PUBLIC_KEY is; both are required for Web Push")
	}

	// --- Development mode only below this point ---

	if d == nil {
		return nil, nil
	}

	// Priority 2: Database (dev only)
	dbPub, err := db.GetServerConfig(ctx, d, configKeyVAPIDPublicKey)
	if err != nil {
		return nil, err
	}
	dbPriv, err := db.GetServerConfig(ctx, d, configKeyVAPIDPrivateKey)
	if err != nil {
		return nil, err
	}
	if dbPub != "" && dbPriv != "" {
		if err := validateVAPIDKeyFormat(dbPub, dbPriv); err != nil {
			return nil, fmt.Errorf("VAPID keys in database are corrupt (dev mode; delete server_config rows and restart to regenerate): %w", err)
		}
		keys := &VAPIDKeys{PublicKey: dbPub, PrivateKey: dbPriv}
		log.Printf("Web Push: using auto-generated VAPID keys from database (development mode, fingerprint: %s)", keys.PublicKeyFingerprint())
		return keys, nil
	}

	// Priority 3: Generate and persist (dev only)
	privKey, pubKey, err = wplib.GenerateVAPIDKeys()
	if err != nil {
		return nil, err
	}

	if err := db.SetServerConfig(ctx, d, configKeyVAPIDPublicKey, pubKey); err != nil {
		return nil, err
	}
	if err := db.SetServerConfig(ctx, d, configKeyVAPIDPrivateKey, privKey); err != nil {
		return nil, err
	}

	keys := &VAPIDKeys{PublicKey: pubKey, PrivateKey: privKey}
	log.Printf("Web Push: auto-generated new VAPID key pair and saved to database (development mode, fingerprint: %s)", keys.PublicKeyFingerprint())
	log.Println("Web Push: for production, set VAPID_PUBLIC_KEY, VAPID_PRIVATE_KEY, and VAPID_SUBJECT env vars (generate keys with: make generate-vapid-keys)")

	return keys, nil
}

// decodeBase64URL decodes a base64url string, tolerating both padded and
// unpadded input. VAPID keys from different tools may include or omit padding.
func decodeBase64URL(s string) ([]byte, error) {
	// Try raw (unpadded) first — this is the most common for VAPID keys.
	if b, err := base64.RawURLEncoding.DecodeString(s); err == nil {
		return b, nil
	}
	// Fall back to padded.
	return base64.URLEncoding.DecodeString(s)
}

// validateVAPIDKeyFormat performs format validation on VAPID keys.
// VAPID keys are base64url-encoded P-256 EC keys: the public key is 65 bytes
// (uncompressed point), the private key is 32 bytes (scalar).
func validateVAPIDKeyFormat(pubKey, privKey string) error {
	pubBytes, err := decodeBase64URL(pubKey)
	if err != nil {
		return fmt.Errorf("VAPID_PUBLIC_KEY is not valid base64url: %w", err)
	}
	privBytes, err := decodeBase64URL(privKey)
	if err != nil {
		return fmt.Errorf("VAPID_PRIVATE_KEY is not valid base64url: %w", err)
	}

	// P-256 uncompressed public key: 1-byte prefix (0x04) + 32-byte X + 32-byte Y = 65 bytes.
	if len(pubBytes) != 65 {
		return fmt.Errorf("VAPID_PUBLIC_KEY decodes to %d bytes (expected 65 for an uncompressed P-256 public key)", len(pubBytes))
	}
	if pubBytes[0] != 0x04 {
		return fmt.Errorf("VAPID_PUBLIC_KEY has invalid prefix byte 0x%02x (expected 0x04 for uncompressed P-256 point)", pubBytes[0])
	}
	// P-256 private key scalar: 32 bytes.
	if len(privBytes) != 32 {
		return fmt.Errorf("VAPID_PRIVATE_KEY decodes to %d bytes (expected 32 for a P-256 private key)", len(privBytes))
	}

	return nil
}
