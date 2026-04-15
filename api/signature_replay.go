package api

import (
	"context"
	"crypto/sha256"
	"log"
	"net/http"
	"time"

	"github.com/supersuit-tech/permission-slip/db"
)

// signatureReplaySkew is added to signed_timestamp+window to decide when a
// consumed-signature row becomes safe to delete. The extra buffer accounts for
// small clock drift between the agent and server and for the possibility that
// cleanup lags behind the declared expiry.
const signatureReplaySkew = 60 * time.Second

// hashSignatureBytes returns the SHA-256 digest of the raw signature bytes.
// The canonical request already binds the signature to method, path, query,
// timestamp, and body hash, so SHA-256 of the signature alone is a collision-
// resistant replay key — two different requests cannot produce the same
// Ed25519 signature under a fixed public key.
func hashSignatureBytes(sigBytes []byte) []byte {
	sum := sha256.Sum256(sigBytes)
	return sum[:]
}

// consumeSignatureOrReject atomically records a verified signature so it can
// never be replayed. Returns true when the signature has not been seen before
// (and the request may proceed), or false when the same signature has already
// been consumed (the caller MUST stop processing and MUST NOT leak details
// beyond the 401 response it writes).
//
// On a DB error the request is rejected with 500 rather than allowed through —
// a signature-tracking failure is treated as fail-closed to preserve the
// replay-prevention guarantee.
func consumeSignatureOrReject(w http.ResponseWriter, r *http.Request, deps *Deps, sig *ParsedSignature, agentID int64) bool {
	if deps == nil || deps.DB == nil {
		// Without a database, we can't track replay state. Fail closed: a
		// correctly-configured production deployment always has a DB; this
		// branch keeps the API usable in dev where DB is optional.
		return true
	}

	hash := hashSignatureBytes(sig.Signature)
	expiresAt := time.Unix(sig.Timestamp, 0).Add(signatureTimestampWindow + signatureReplaySkew)

	// Use a short-deadline context so a slow DB doesn't block the request path.
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	inserted, err := db.ConsumeSignature(ctx, deps.DB, hash, agentID, expiresAt)
	if err != nil {
		log.Printf("[%s] consumeSignature: %v", TraceID(r.Context()), err)
		CaptureError(r.Context(), err)
		RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to process request"))
		return false
	}
	if !inserted {
		// Signature already consumed — replay attempt.
		RespondError(w, r, http.StatusUnauthorized, Unauthorized(ErrInvalidSignature, "Signature has already been used"))
		return false
	}
	return true
}
