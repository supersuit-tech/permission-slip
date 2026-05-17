package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/supersuit-tech/permission-slip/auth"
	"github.com/supersuit-tech/permission-slip/db"
)

type userIDKey struct{}
type emailKey struct{}
type profileKey struct{}

// RequireSession returns middleware that validates HS256 access JWTs issued by this service.
func RequireSession(deps *Deps) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
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

			if deps.JWTSigningSecret == "" {
				log.Printf("[%s] RequireSession: JWT signing secret not configured", TraceID(r.Context()))
				RespondError(w, r, http.StatusInternalServerError, InternalError("Session authentication not configured"))
				return
			}

			unverified, _, err := jwt.NewParser().ParseUnverified(tokenString, jwt.MapClaims{})
			if err != nil {
				RespondError(w, r, http.StatusUnauthorized, Unauthorized(ErrInvalidToken, "Malformed token"))
				return
			}
			if unverified.Method == nil || unverified.Method.Alg() != "HS256" {
				RespondError(w, r, http.StatusUnauthorized, Unauthorized(ErrInvalidToken, "Unsupported signing algorithm"))
				return
			}

			secret := []byte(deps.JWTSigningSecret)
			sub, em, err := auth.VerifyAccessToken(secret, tokenString)
			if err != nil || sub == "" {
				RespondError(w, r, http.StatusUnauthorized, Unauthorized(ErrInvalidToken, "Invalid or expired session token"))
				return
			}

			ctx := context.WithValue(r.Context(), userIDKey{}, sub)
			if em != "" {
				ctx = context.WithValue(ctx, emailKey{}, em)
			}
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
				CaptureError(r.Context(), fmt.Errorf("RequireProfile: database not configured"))
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
				q := r.URL.Query()
				q.Del("access_token")
				r.URL.RawQuery = q.Encode()
				r.RequestURI = r.URL.RequestURI()
			}
		}
		next.ServeHTTP(w, r)
	})
}
