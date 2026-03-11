package api

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/supersuit-tech/permission-slip-web/db"
)

// SupabaseAudAuthenticated is the expected "aud" claim in Supabase user-session JWTs.
const SupabaseAudAuthenticated = "authenticated"

type userIDKey struct{}
type emailKey struct{}
type profileKey struct{}

// JWKSCache fetches and caches EC public keys from a JWKS endpoint.
// Safe for concurrent use. Create one per process via NewJWKSCache and
// store it in Deps.JWKSCache.
type JWKSCache struct {
	mu        sync.RWMutex
	keys      map[string]*ecdsa.PublicKey // kid → public key
	fetchedAt time.Time
	url       string
}

// NewJWKSCache creates a JWKS cache that fetches keys from the given URL.
func NewJWKSCache(url string) *JWKSCache {
	return &JWKSCache{url: url}
}

const jwksCacheTTL = 5 * time.Minute

// getECKey returns the EC public key for the given kid, refreshing the cache
// from the JWKS endpoint when stale or when the kid is unknown.
func (c *JWKSCache) getECKey(ctx context.Context, kid string) (*ecdsa.PublicKey, error) {
	c.mu.RLock()
	fresh := time.Since(c.fetchedAt) < jwksCacheTTL
	k := c.keys[kid]
	c.mu.RUnlock()
	if fresh && k != nil {
		return k, nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	// Re-check after acquiring write lock.
	if time.Since(c.fetchedAt) < jwksCacheTTL {
		if k := c.keys[kid]; k != nil {
			return k, nil
		}
	}
	if err := c.fetchLocked(ctx); err != nil {
		return nil, err
	}
	k = c.keys[kid]
	if k == nil {
		return nil, fmt.Errorf("kid %q not found in JWKS after refresh", kid)
	}
	return k, nil
}

// fetchLocked fetches the JWKS and populates c.keys. Caller must hold c.mu write lock.
func (c *JWKSCache) fetchLocked(ctx context.Context) error {
	if c.url == "" {
		return fmt.Errorf("JWKS URL not configured")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url, nil)
	if err != nil {
		return fmt.Errorf("create JWKS request: %w", err)
	}
	resp, err := jwksHTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("fetch JWKS from %s: %w", c.url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("JWKS endpoint %s returned %d", c.url, resp.StatusCode)
	}

	// Minimal JWKS JSON structure — only what we need.
	var jwks struct {
		Keys []struct {
			Kid string `json:"kid"`
			Kty string `json:"kty"`
			Crv string `json:"crv"`
			X   string `json:"x"`
			Y   string `json:"y"`
		} `json:"keys"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return fmt.Errorf("decode JWKS: %w", err)
	}

	newKeys := make(map[string]*ecdsa.PublicKey, len(jwks.Keys))
	for _, k := range jwks.Keys {
		if k.Kty != "EC" || k.Crv != "P-256" {
			continue
		}
		pub, err := ecPublicKeyFromBase64url(k.X, k.Y)
		if err != nil {
			log.Printf("JWKS: skipping key kid=%q: %v", k.Kid, err)
			continue
		}
		newKeys[k.Kid] = pub
	}
	c.keys = newKeys
	c.fetchedAt = time.Now()
	log.Printf("JWKS: cached %d EC P-256 key(s) from %s", len(newKeys), c.url)
	return nil
}

// ecPublicKeyFromBase64url reconstructs an *ecdsa.PublicKey from
// base64url-encoded x and y coordinates (JWKS EC key format, RFC 7517).
func ecPublicKeyFromBase64url(xEnc, yEnc string) (*ecdsa.PublicKey, error) {
	decode := func(s string) ([]byte, error) {
		// base64url without padding
		return base64.RawURLEncoding.DecodeString(s)
	}
	xBytes, err := decode(xEnc)
	if err != nil {
		return nil, fmt.Errorf("decode x: %w", err)
	}
	yBytes, err := decode(yEnc)
	if err != nil {
		return nil, fmt.Errorf("decode y: %w", err)
	}
	return &ecdsa.PublicKey{
		Curve: elliptic.P256(),
		X:     new(big.Int).SetBytes(xBytes),
		Y:     new(big.Int).SetBytes(yBytes),
	}, nil
}

// jwksHTTPClient is a dedicated HTTP client for JWKS fetches with a short
// timeout to prevent a slow/hanging JWKS endpoint from blocking all auth
// requests (the fetch holds the jwksCache write lock).
var jwksHTTPClient = &http.Client{Timeout: 10 * time.Second}

// RequireSession returns middleware that validates Supabase session JWTs.
//
// Supports two signing algorithms:
//
//   - ES256 — Supabase CLI v2+ (asymmetric). Requires Deps.JWKSCache
//     (created once at startup via NewJWKSCache).
//
//   - HS256 — Supabase CLI v1 and test environments. Requires Deps.SupabaseJWTSecret.
//
// The algorithm is read from the incoming token's "alg" header — no config
// needed per-request. Both modes enforce aud="authenticated" and expiry.
func RequireSession(deps *Deps) func(http.Handler) http.Handler {
	var jwksMisconfigOnce sync.Once
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				// If the request has an agent signature header, the caller is
				// likely an agent hitting a dashboard endpoint by mistake.
				// Give a more helpful error message.
				if r.Header.Get(signatureHeader) != "" {
					RespondError(w, r, http.StatusUnauthorized, Unauthorized(ErrInvalidToken,
						"This endpoint requires user session authentication (Authorization: Bearer <token>), not agent signature authentication. Use the agent API endpoints (e.g. /api/v1/agents/me) instead."))
					return
				}
				RespondError(w, r, http.StatusUnauthorized, Unauthorized(ErrInvalidToken, "Missing Authorization header"))
				return
			}

			tokenString, ok := strings.CutPrefix(authHeader, "Bearer ")
			if !ok || tokenString == "" {
				RespondError(w, r, http.StatusUnauthorized, Unauthorized(ErrInvalidToken, "Authorization header must use Bearer scheme"))
				return
			}

			// Peek at the unverified header to choose the right key/algorithm.
			unverified, _, err := jwt.NewParser().ParseUnverified(tokenString, jwt.MapClaims{})
			if err != nil {
				RespondError(w, r, http.StatusUnauthorized, Unauthorized(ErrInvalidToken, "Malformed token"))
				return
			}

			var token *jwt.Token

			switch unverified.Method.Alg() {
			case "ES256":
				if deps.JWKSCache == nil {
					log.Printf("[%s] ES256 token but JWKS cache not configured", TraceID(r.Context()))
					jwksMisconfigOnce.Do(func() {
						CaptureError(r.Context(), fmt.Errorf("ES256 token received but JWKS cache not configured"))
					})
					RespondError(w, r, http.StatusInternalServerError, InternalError("Session authentication not configured"))
					return
				}
				token, err = jwt.Parse(tokenString,
					func(t *jwt.Token) (any, error) {
						if _, ok := t.Method.(*jwt.SigningMethodECDSA); !ok {
							return nil, jwt.ErrSignatureInvalid
						}
						kid, _ := t.Header["kid"].(string)
						return deps.JWKSCache.getECKey(r.Context(), kid)
					},
					jwt.WithValidMethods([]string{"ES256"}),
					jwt.WithAudience(SupabaseAudAuthenticated),
					jwt.WithExpirationRequired(),
				)

			case "HS256":
				if deps.SupabaseJWTSecret == "" {
					RespondError(w, r, http.StatusInternalServerError, InternalError("Session authentication not configured"))
					return
				}
				token, err = jwt.Parse(tokenString,
					func(t *jwt.Token) (any, error) {
						if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
							return nil, jwt.ErrSignatureInvalid
						}
						return []byte(deps.SupabaseJWTSecret), nil
					},
					jwt.WithValidMethods([]string{"HS256"}),
					jwt.WithAudience(SupabaseAudAuthenticated),
					jwt.WithExpirationRequired(),
				)

			default:
				RespondError(w, r, http.StatusUnauthorized, Unauthorized(ErrInvalidToken, "Unsupported signing algorithm"))
				return
			}

			if err != nil || !token.Valid {
				RespondError(w, r, http.StatusUnauthorized, Unauthorized(ErrInvalidToken, "Invalid or expired session token"))
				return
			}

			sub, err := token.Claims.GetSubject()
			if err != nil || sub == "" {
				RespondError(w, r, http.StatusUnauthorized, Unauthorized(ErrInvalidToken, "Token missing subject claim"))
				return
			}

			ctx := context.WithValue(r.Context(), userIDKey{}, sub)
			// Extract email from JWT for profile recovery (re-linking).
			if claims, ok := token.Claims.(jwt.MapClaims); ok {
				if email, ok := claims["email"].(string); ok && email != "" {
					ctx = context.WithValue(ctx, emailKey{}, email)
				}
			}
			// Tag Sentry events with the authenticated user so error reports
			// show which user was affected.
			SetSentryUser(ctx, sub)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// UserID returns the authenticated user's ID from the request context.
func UserID(ctx context.Context) string {
	id, _ := ctx.Value(userIDKey{}).(string)
	return id
}

// UserEmail returns the authenticated user's email from the JWT claims,
// or "" if not available.
func UserEmail(ctx context.Context) string {
	e, _ := ctx.Value(emailKey{}).(string)
	return e
}

// RequireProfile chains RequireSession → profile lookup.
// Stores the profile in context for retrieval via Profile(ctx).
func RequireProfile(deps *Deps) func(http.Handler) http.Handler {
	sessionAuth := RequireSession(deps)
	return func(next http.Handler) http.Handler {
		return sessionAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := UserID(r.Context())
			if userID == "" {
				RespondError(w, r, http.StatusUnauthorized, Unauthorized(ErrInvalidToken, "Authentication required"))
				return
			}
			if deps.DB == nil {
				log.Printf("[%s] RequireProfile: database not configured", TraceID(r.Context()))
				RespondError(w, r, http.StatusServiceUnavailable, ServiceUnavailable("Database not available"))
				return
			}
			profile, err := db.GetProfileByUserID(r.Context(), deps.DB, userID)
			if err != nil {
				log.Printf("[%s] RequireProfile: profile lookup: %v", TraceID(r.Context()), err)
				CaptureError(r.Context(), err)
				RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to verify profile"))
				return
			}
			if profile == nil {
				// Profile not found by user ID. This can happen when Supabase
				// creates a new auth identity (new UUID) for an existing email
				// — the user's old profile is orphaned under the old UUID.
				// Attempt to recover by looking up the profile via email and
				// re-linking it to the current user ID.
				if email := UserEmail(r.Context()); email != "" {
					old, findErr := db.FindProfileByAuthEmail(r.Context(), deps.DB, email)
					if findErr != nil {
						log.Printf("[%s] RequireProfile: email fallback lookup: %v", TraceID(r.Context()), findErr)
					}
					if old != nil && old.ID != userID {
						if rlErr := db.RelinkProfile(r.Context(), deps.DB, old.ID, userID); rlErr != nil {
							log.Printf("[%s] RequireProfile: re-link profile %s→%s: %v", TraceID(r.Context()), old.ID, userID, rlErr)
							// A concurrent request may have already completed the re-link.
							// Re-fetch by the new user ID before falling through to 404.
							if p, fetchErr := db.GetProfileByUserID(r.Context(), deps.DB, userID); fetchErr != nil {
								log.Printf("[%s] RequireProfile: re-fetch after concurrent re-link: %v", TraceID(r.Context()), fetchErr)
							} else {
								profile = p
							}
						} else {
							log.Printf("[%s] RequireProfile: re-linked profile %s→%s", TraceID(r.Context()), old.ID, userID)
							old.ID = userID
							profile = old
						}
					} else if old != nil {
						// old.ID == userID: a concurrent request already completed the
						// re-link. The profile is correctly linked; use it directly.
						log.Printf("[%s] RequireProfile: profile already re-linked to %s (concurrent)", TraceID(r.Context()), userID)
						profile = old
					}
				}
			}
			if profile == nil {
				RespondError(w, r, http.StatusNotFound, NotFound(ErrProfileNotFound, "Profile not found"))
				return
			}
			ctx := context.WithValue(r.Context(), profileKey{}, profile)
			next.ServeHTTP(w, r.WithContext(ctx))
		}))
	}
}

// Profile returns the authenticated user's profile from the request context.
func Profile(ctx context.Context) *db.Profile {
	p, _ := ctx.Value(profileKey{}).(*db.Profile)
	return p
}

// AllowQueryParamToken is a middleware that promotes an access_token query
// parameter into the Authorization header. This implements RFC 6750 §2.3
// for specific routes reached via browser redirect (e.g. OAuth authorize)
// where the caller cannot set headers.
//
// Apply this only to routes that genuinely need it — query-param tokens
// are more easily leaked via logs, Referer headers, and browser history
// than header-based tokens.
func AllowQueryParamToken(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "" {
			if qt := r.URL.Query().Get("access_token"); qt != "" {
				r = r.Clone(r.Context())
				r.Header.Set("Authorization", "Bearer "+qt)
				// Strip the token from the URL to prevent leaking via Referer
				// headers on subsequent redirects (RFC 6750 §5.3).
				q := r.URL.Query()
				q.Del("access_token")
				r.URL.RawQuery = q.Encode()
				// r.RequestURI is the raw request-target copied verbatim by Clone;
				// clear it so downstream handlers cannot observe the token via
				// r.RequestURI even if r.URL.RawQuery is already clean.
				r.RequestURI = r.URL.RequestURI()
			}
		}
		next.ServeHTTP(w, r)
	})
}
