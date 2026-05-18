package api

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	"github.com/supersuit-tech/permission-slip/auth"
	"github.com/supersuit-tech/permission-slip/db"
)

const (
	refreshTokenTTL = 30 * 24 * time.Hour
	minPasswordLen  = 8
	maxPasswordLen  = 128
)

type authTokenResponse struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

type authSignupRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authLoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authRefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type authLogoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// RegisterAuthRoutes registers POST /signup, /login, /refresh, /logout on mux.
// The mux should be mounted with http.StripPrefix("/api/auth", mux) from the root server.
func RegisterAuthRoutes(mux *http.ServeMux, deps *Deps) {
	mux.Handle("POST /signup", handleAuthSignup(deps))
	mux.Handle("POST /login", handleAuthLogin(deps))
	mux.Handle("POST /refresh", handleAuthRefresh(deps))
	mux.Handle("POST /logout", handleAuthLogout(deps))
}

func authRateAllow(deps *Deps, kind string, r *http.Request) bool {
	if deps.DevMode || deps.AuthRateLimiter == nil {
		return true
	}
	ip := clientIP(r, deps.TrustedProxyHeader)
	key := "auth:" + kind + ":" + ip
	ok, _ := deps.AuthRateLimiter.Allow(key)
	return ok
}

func issueTokenPair(ctx context.Context, d db.DBTX, deps *Deps, userID, email string) (access, refresh string, expiresAt time.Time, err error) {
	now := time.Now().UTC()
	secret := []byte(deps.JWTSigningSecret)
	access, expiresAt, err = auth.IssueAccessToken(secret, userID, email, now)
	if err != nil {
		return "", "", time.Time{}, err
	}
	refresh, err = auth.NewOpaqueRefreshToken()
	if err != nil {
		return "", "", time.Time{}, err
	}
	sid := "as_" + uuid.New().String()
	hash := auth.HashRefreshToken(refresh)
	exp := now.Add(refreshTokenTTL)
	if err := db.CreateAuthSession(ctx, d, sid, userID, hash, exp); err != nil {
		return "", "", time.Time{}, err
	}
	return access, refresh, expiresAt, nil
}

func normalizeEmail(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

func validateEmailShape(email string) bool {
	if len(email) < 5 || len(email) > 254 {
		return false
	}
	at := strings.LastIndex(email, "@")
	if at <= 0 || at == len(email)-1 {
		return false
	}
	domain := email[at+1:]
	return strings.Contains(domain, ".")
}

func baseUsernameFromEmail(email string) string {
	at := strings.Index(email, "@")
	if at <= 0 {
		return "user"
	}
	local := strings.ToLower(email[:at])
	var b strings.Builder
	for _, r := range local {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '_', r == '-':
			b.WriteRune(r)
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(unicode.ToLower(r))
		default:
			b.WriteRune('_')
		}
	}
	s := b.String()
	s = strings.Trim(s, "_-")
	if len(s) < 3 {
		return "user"
	}
	if len(s) > 200 {
		s = s[:200]
	}
	return s
}

func handleAuthSignup(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.DB == nil {
			RespondError(w, r, http.StatusServiceUnavailable, ServiceUnavailable("Database not available"))
			return
		}
		if deps.JWTSigningSecret == "" {
			RespondError(w, r, http.StatusInternalServerError, InternalError("Authentication not configured"))
			return
		}
		if !authRateAllow(deps, "signup", r) {
			RespondError(w, r, http.StatusTooManyRequests, TooManyRequests("Too many signup attempts. Please try again later.", 60))
			return
		}

		var req authSignupRequest
		if !DecodeJSONOrReject(w, r, &req) {
			return
		}
		email := normalizeEmail(req.Email)
		password := strings.TrimSpace(req.Password)
		if !validateEmailShape(email) {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "Invalid email address"))
			return
		}
		if len(password) < minPasswordLen || len(password) > maxPasswordLen {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, fmt.Sprintf("Password must be between %d and %d characters", minPasswordLen, maxPasswordLen)))
			return
		}

		hash, err := auth.HashPassword(password)
		if err != nil {
			log.Printf("[%s] signup: hash password: %v", TraceID(r.Context()), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Could not create account"))
			return
		}

		uid := uuid.New().String()
		baseUser := baseUsernameFromEmail(email)

		tx, owned, err := db.BeginOrContinue(r.Context(), deps.DB)
		if err != nil {
			log.Printf("[%s] signup: begin: %v", TraceID(r.Context()), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Could not create account"))
			return
		}
		if owned {
			defer db.RollbackTx(r.Context(), tx)
		}

		if err := db.CreateUserWithPassword(r.Context(), tx, uid, email, hash); err != nil {
			if db.IsUniqueViolation(err) {
				RespondError(w, r, http.StatusConflict, Conflict(ErrConstraintViolation, "An account with this email already exists"))
				return
			}
			log.Printf("[%s] signup: insert user: %v", TraceID(r.Context()), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Could not create account"))
			return
		}

		var profile *db.Profile
		for attempt := 0; attempt < 8; attempt++ {
			username := baseUser
			if attempt > 0 {
				username = fmt.Sprintf("%s_%s", baseUser, uuid.New().String()[:8])
			}
			profile, err = db.CreateProfile(r.Context(), tx, uid, username, email, false)
			if err == nil {
				break
			}
			var oe *db.OnboardingError
			if errors.As(err, &oe) && oe.Code == db.OnboardingErrUsernameTaken {
				continue
			}
			log.Printf("[%s] signup: create profile: %v", TraceID(r.Context()), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Could not create account"))
			return
		}
		if profile == nil {
			RespondError(w, r, http.StatusInternalServerError, InternalError("Could not create account"))
			return
		}

		access, refresh, exp, err := issueTokenPair(r.Context(), tx, deps, uid, email)
		if err != nil {
			log.Printf("[%s] signup: tokens: %v", TraceID(r.Context()), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Could not create account"))
			return
		}

		if owned {
			if err := db.CommitTx(r.Context(), tx); err != nil {
				log.Printf("[%s] signup: commit: %v", TraceID(r.Context()), err)
				RespondError(w, r, http.StatusInternalServerError, InternalError("Could not create account"))
				return
			}
		}

		RespondJSON(w, http.StatusCreated, authTokenResponse{
			AccessToken:  access,
			RefreshToken: refresh,
			ExpiresAt:    exp,
		})
	}
}

func handleAuthLogin(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.DB == nil {
			RespondError(w, r, http.StatusServiceUnavailable, ServiceUnavailable("Database not available"))
			return
		}
		if deps.JWTSigningSecret == "" {
			RespondError(w, r, http.StatusInternalServerError, InternalError("Authentication not configured"))
			return
		}
		if !authRateAllow(deps, "login", r) {
			RespondError(w, r, http.StatusTooManyRequests, TooManyRequests("Too many login attempts. Please try again later.", 60))
			return
		}

		var req authLoginRequest
		if !DecodeJSONOrReject(w, r, &req) {
			return
		}
		email := normalizeEmail(req.Email)
		password := req.Password

		u, err := db.GetUserAuthByEmail(r.Context(), deps.DB, email)
		if err != nil {
			log.Printf("[%s] login: lookup: %v", TraceID(r.Context()), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Login failed"))
			return
		}
		ok := false
		if u != nil {
			ok, err = auth.VerifyPassword(password, u.PasswordHash)
			if err != nil {
				log.Printf("[%s] login: verify: %v", TraceID(r.Context()), err)
				RespondError(w, r, http.StatusInternalServerError, InternalError("Login failed"))
				return
			}
		}
		if u == nil || !ok {
			time.Sleep(80 * time.Millisecond)
			RespondError(w, r, http.StatusUnauthorized, Unauthorized(ErrInvalidToken, "Invalid email or password"))
			return
		}

		tx, owned, err := db.BeginOrContinue(r.Context(), deps.DB)
		if err != nil {
			log.Printf("[%s] login: begin: %v", TraceID(r.Context()), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Login failed"))
			return
		}
		if owned {
			defer db.RollbackTx(r.Context(), tx)
		}

		access, refresh, exp, err := issueTokenPair(r.Context(), tx, deps, u.ID, u.Email)
		if err != nil {
			log.Printf("[%s] login: tokens: %v", TraceID(r.Context()), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Login failed"))
			return
		}
		if owned {
			if err := db.CommitTx(r.Context(), tx); err != nil {
				log.Printf("[%s] login: commit: %v", TraceID(r.Context()), err)
				RespondError(w, r, http.StatusInternalServerError, InternalError("Login failed"))
				return
			}
		}

		RespondJSON(w, http.StatusOK, authTokenResponse{
			AccessToken:  access,
			RefreshToken: refresh,
			ExpiresAt:    exp,
		})
	}
}

func handleAuthRefresh(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.DB == nil {
			RespondError(w, r, http.StatusServiceUnavailable, ServiceUnavailable("Database not available"))
			return
		}
		if deps.JWTSigningSecret == "" {
			RespondError(w, r, http.StatusInternalServerError, InternalError("Authentication not configured"))
			return
		}

		var req authRefreshRequest
		if !DecodeJSONOrReject(w, r, &req) {
			return
		}
		if strings.TrimSpace(req.RefreshToken) == "" {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "refresh_token is required"))
			return
		}

		oldHash := auth.HashRefreshToken(req.RefreshToken)
		sess, err := db.GetAuthSessionByRefreshHash(r.Context(), deps.DB, oldHash)
		if err != nil {
			log.Printf("[%s] refresh: lookup: %v", TraceID(r.Context()), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Token refresh failed"))
			return
		}
		if sess == nil {
			RespondError(w, r, http.StatusUnauthorized, Unauthorized(ErrInvalidToken, "Invalid or expired refresh token"))
			return
		}
		if sess.RevokedAt != nil {
			RespondError(w, r, http.StatusUnauthorized, Unauthorized(ErrInvalidToken, "Invalid or expired refresh token"))
			return
		}
		if time.Now().After(sess.ExpiresAt) {
			RespondError(w, r, http.StatusUnauthorized, Unauthorized(ErrInvalidToken, "Invalid or expired refresh token"))
			return
		}

		prof, err := db.GetProfileByUserID(r.Context(), deps.DB, sess.UserID)
		if err != nil {
			log.Printf("[%s] refresh: profile: %v", TraceID(r.Context()), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Token refresh failed"))
			return
		}
		email := ""
		if prof != nil && prof.Email != nil {
			email = *prof.Email
		}

		tx, owned, err := db.BeginOrContinue(r.Context(), deps.DB)
		if err != nil {
			log.Printf("[%s] refresh: begin: %v", TraceID(r.Context()), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Token refresh failed"))
			return
		}
		if owned {
			defer db.RollbackTx(r.Context(), tx)
		}

		now := time.Now().UTC()
		if err := db.RevokeAuthSession(r.Context(), tx, sess.ID, now); err != nil {
			log.Printf("[%s] refresh: revoke: %v", TraceID(r.Context()), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Token refresh failed"))
			return
		}

		newRefresh, err := auth.NewOpaqueRefreshToken()
		if err != nil {
			log.Printf("[%s] refresh: new token: %v", TraceID(r.Context()), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Token refresh failed"))
			return
		}
		newSID := "as_" + uuid.New().String()
		newHash := auth.HashRefreshToken(newRefresh)
		if err := db.CreateAuthSession(r.Context(), tx, newSID, sess.UserID, newHash, now.Add(refreshTokenTTL)); err != nil {
			log.Printf("[%s] refresh: insert session: %v", TraceID(r.Context()), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Token refresh failed"))
			return
		}

		access, exp, err := auth.IssueAccessToken([]byte(deps.JWTSigningSecret), sess.UserID, email, now)
		if err != nil {
			log.Printf("[%s] refresh: access jwt: %v", TraceID(r.Context()), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Token refresh failed"))
			return
		}

		if owned {
			if err := db.CommitTx(r.Context(), tx); err != nil {
				log.Printf("[%s] refresh: commit: %v", TraceID(r.Context()), err)
				RespondError(w, r, http.StatusInternalServerError, InternalError("Token refresh failed"))
				return
			}
		}

		RespondJSON(w, http.StatusOK, authTokenResponse{
			AccessToken:  access,
			RefreshToken: newRefresh,
			ExpiresAt:    exp,
		})
	}
}

func handleAuthLogout(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.DB == nil {
			RespondError(w, r, http.StatusServiceUnavailable, ServiceUnavailable("Database not available"))
			return
		}
		var req authLogoutRequest
		if !DecodeJSONOrReject(w, r, &req) {
			return
		}
		if strings.TrimSpace(req.RefreshToken) == "" {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "refresh_token is required"))
			return
		}
		hash := auth.HashRefreshToken(req.RefreshToken)
		if err := db.RevokeAuthSessionByRefreshHash(r.Context(), deps.DB, hash, time.Now().UTC()); err != nil {
			log.Printf("[%s] logout: %v", TraceID(r.Context()), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Logout failed"))
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
