package api

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/supersuit-tech/permission-slip-web/db"
)

const (
	// defaultInviteTTL is the default invite lifetime (15 minutes).
	defaultInviteTTL = 15 * time.Minute

	// inviteRateLimit is the maximum number of invites a user can create per window.
	inviteRateLimit = 10

	// inviteRateWindow is the sliding window for invite rate limiting.
	inviteRateWindow = 1 * time.Hour
)

type createInviteRequest struct {
	ExpiresIn *int `json:"expires_in" validate:"omitempty,gte=60,lte=86400"` // TTL in seconds (optional, 1min–24h)
}

type createInviteResponse struct {
	ID         string    `json:"id"`
	InviteCode string    `json:"invite_code"`
	InviteURL  string    `json:"invite_url,omitempty"`
	Status     string    `json:"status"`
	ExpiresAt  time.Time `json:"expires_at"`
	CreatedAt  time.Time `json:"created_at"`
}

// RegisterRegistrationInviteRoutes adds registration-invite endpoints to the mux.
func RegisterRegistrationInviteRoutes(mux *http.ServeMux, deps *Deps) {
	requireProfile := RequireProfile(deps)
	mux.Handle("POST /registration-invites", requireProfile(handleCreateRegistrationInvite(deps)))
}

func handleCreateRegistrationInvite(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := Profile(r.Context()).ID

		// Rate-limit check runs before body decoding to reject abusive
		// callers early, before spending resources on parsing or crypto.
		// Skipped in development mode to avoid slowing down local testing.
		if !deps.DevMode {
			recentCount, err := db.CountRecentInvitesByUser(r.Context(), deps.DB, userID, inviteRateWindow)
			if err != nil {
				log.Printf("[%s] CreateRegistrationInvite: rate limit check: %v", TraceID(r.Context()), err)
				RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to create registration invite"))
				return
			}
			if recentCount >= inviteRateLimit {
				// retry_after is 0: with a sliding window we can't cheaply
				// compute the exact time until a slot opens.
				// NOTE: user-facing message says "per hour" — update if
				// inviteRateWindow changes from 1h.
				RespondError(w, r, http.StatusTooManyRequests, TooManyRequests(
					fmt.Sprintf("You can create at most %d invites per hour", inviteRateLimit),
					0,
				))
				return
			}
		}

		var req createInviteRequest
		if !DecodeJSONOrReject(w, r, &req) {
			return
		}
		if !ValidateRequest(w, r, &req) {
			return
		}

		ttl := defaultInviteTTL
		if req.ExpiresIn != nil {
			ttl = time.Duration(*req.ExpiresIn) * time.Second
		}

		const maxRetries = 3
		var invite *db.RegistrationInvite
		var inviteCode string
		var rateLimited bool

		for attempt := 0; attempt < maxRetries; attempt++ {
			var err error
			inviteCode, err = generateInviteCode()
			if err != nil {
				log.Printf("[%s] CreateRegistrationInvite: failed to generate invite code: %v", TraceID(r.Context()), err)
				RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to generate invite code"))
				return
			}

			inviteID, err := generatePrefixedID("ri_", 16)
			if err != nil {
				log.Printf("[%s] CreateRegistrationInvite: failed to generate invite ID: %v", TraceID(r.Context()), err)
				RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to generate invite"))
				return
			}

			codeHash := hashCodeHex(inviteCode, deps.InviteHMACKey)
			ttlSeconds := int(ttl.Seconds())

			if deps.DevMode {
				invite, err = db.CreateRegistrationInvite(r.Context(), deps.DB, inviteID, userID, codeHash, ttlSeconds)
			} else {
				invite, err = db.CreateRegistrationInviteIfUnderLimit(
					r.Context(), deps.DB, inviteID, userID, codeHash, ttlSeconds,
					int(inviteRateWindow.Seconds()), inviteRateLimit,
				)
			}
			if err != nil {
				var pgErr *pgconn.PgError
				if errors.As(err, &pgErr) && pgErr.Code == db.PgCodeUniqueViolation {
					log.Printf("[%s] CreateRegistrationInvite: unique violation on attempt %d, retrying", TraceID(r.Context()), attempt+1)
					continue
				}
				log.Printf("[%s] CreateRegistrationInvite: %v", TraceID(r.Context()), err)
				RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to create registration invite"))
				return
			}
			if invite == nil {
				// Atomic rate limit check rejected the insert.
				rateLimited = true
			}
			break
		}
		if rateLimited {
			RespondError(w, r, http.StatusTooManyRequests, TooManyRequests(
				fmt.Sprintf("You can create at most %d invites per hour", inviteRateLimit),
				0,
			))
			return
		}
		if invite == nil {
			log.Printf("[%s] CreateRegistrationInvite: exhausted retries due to unique violations", TraceID(r.Context()))
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to create registration invite"))
			return
		}

		resp := createInviteResponse{
			ID:         invite.ID,
			InviteCode: inviteCode,
			Status:     invite.Status,
			ExpiresAt:  invite.ExpiresAt,
			CreatedAt:  invite.CreatedAt,
		}

		if deps.BaseURL != "" {
			resp.InviteURL = buildInviteURL(deps.BaseURL, inviteCode)
		}

		RespondJSON(w, http.StatusCreated, resp)
	}
}
