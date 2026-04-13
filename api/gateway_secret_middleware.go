package api

import (
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
// The check runs as the outermost middleware so unauthorized requests are
// rejected with minimal processing (before CORS, security headers, routing,
// auth, etc.). CORS preflight (OPTIONS) requests are exempt because browsers
// send them without custom headers.
func GatewaySecretMiddleware(secret string) func(http.Handler) http.Handler {
	if secret != "" {
		log.Println("Gateway secret: enabled — requests without a valid X-Gateway-Secret header will be rejected")
	}

	return func(next http.Handler) http.Handler {
		if secret == "" {
			return next
		}

		secretBytes := []byte(secret)

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Allow CORS preflight through — browsers send OPTIONS without
			// custom headers, so blocking them would break cross-origin
			// requests entirely.
			if r.Method == http.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}

			provided := r.Header.Get("X-Gateway-Secret")
			if subtle.ConstantTimeCompare([]byte(provided), secretBytes) != 1 {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
