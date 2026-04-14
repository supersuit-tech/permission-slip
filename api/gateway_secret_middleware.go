package api

import (
	"crypto/sha256"
	"crypto/subtle"
	"log"
	"net/http"
)

// GatewaySecretMiddleware returns middleware that rejects requests without a
// valid X-Gateway-Secret header. When secret is empty, the middleware is a
// no-op — all requests pass through unchanged.
//
// This is designed for private deployments behind Cloudflare Tunnel: set the
// GATEWAY_SECRET env var to a long random string and configure clients (mobile
// app, curl, etc.) to send the same value in the X-Gateway-Secret header.
//
// Middleware ordering: in main.go this runs inside SecurityHeadersMiddleware
// (which only sets response headers and never blocks), but outside CORS,
// routing, and auth — so unauthorized requests are rejected before any
// application processing. True CORS preflight requests (OPTIONS with an
// Access-Control-Request-Method header) are exempt because browsers send them
// without custom headers; other OPTIONS requests are still gated.
//
// The header and configured secret are SHA-256 hashed before comparison so
// that the fixed-length digests (always 32 bytes) feed ConstantTimeCompare.
// This avoids leaking the secret length via an early length-mismatch return
// inside ConstantTimeCompare.
func GatewaySecretMiddleware(secret string) func(http.Handler) http.Handler {
	if secret != "" {
		log.Println("Gateway secret: enabled — requests without a valid X-Gateway-Secret header will be rejected")
	}

	return func(next http.Handler) http.Handler {
		if secret == "" {
			return next
		}

		secretSum := sha256.Sum256([]byte(secret))

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Exempt only genuine CORS preflights — OPTIONS requests that
			// include Access-Control-Request-Method. Bare OPTIONS requests
			// still require the gateway secret so nothing slips through.
			if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
				next.ServeHTTP(w, r)
				return
			}

			providedSum := sha256.Sum256([]byte(r.Header.Get("X-Gateway-Secret")))
			if subtle.ConstantTimeCompare(providedSum[:], secretSum[:]) != 1 {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
