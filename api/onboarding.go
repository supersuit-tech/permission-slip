package api

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/supersuit-tech/permission-slip-web/db"
	"github.com/supersuit-tech/permission-slip-web/shared"
)

type onboardingRequest struct {
	Username       string `json:"username" validate:"required"`
	MarketingOptIn bool   `json:"marketing_opt_in"` // defaults to false when omitted
}

type onboardingResponse struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	CreatedAt time.Time `json:"created_at"`
}

// RegisterOnboardingRoutes adds onboarding endpoints to the mux.
// RequireSession is used (not RequireProfile) because the user has no profile yet.
func init() {
	RegisterRouteGroup(RegisterOnboardingRoutes)
}

func RegisterOnboardingRoutes(mux *http.ServeMux, deps *Deps) {
	requireSession := RequireSession(deps)
	mux.Handle("POST /onboarding", requireSession(handleOnboarding(deps)))
}

func handleOnboarding(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := UserID(r.Context())

		if deps.DB == nil {
			RespondError(w, r, http.StatusServiceUnavailable, ServiceUnavailable("Database not available"))
			return
		}

		// If user already has a profile, return it (idempotent).
		existing, err := db.GetProfileByUserID(r.Context(), deps.DB, userID)
		if err != nil {
			log.Printf("[%s] Onboarding: profile lookup: %v", TraceID(r.Context()), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to check existing profile"))
			return
		}
		if existing != nil {
			RespondJSON(w, http.StatusOK, toOnboardingResponse(existing))
			return
		}

		var req onboardingRequest
		if !DecodeJSONOrReject(w, r, &req) {
			return
		}

		// Trim before validation so the required tag rejects whitespace-only strings.
		req.Username = strings.TrimSpace(req.Username)

		if !ValidateRequest(w, r, &req) {
			return
		}

		username := req.Username
		if err := validateUsername(username); err != nil {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, err.Error()))
			return
		}

		profile, err := db.CreateProfile(r.Context(), deps.DB, userID, username, req.MarketingOptIn)
		if err != nil {
			var onboardErr *db.OnboardingError
			if errors.As(err, &onboardErr) {
				switch onboardErr.Code {
				case db.OnboardingErrUsernameTaken:
					// Before rejecting, check if the taken username belongs to
					// the same user under a different Supabase identity (new UUID,
					// same email). If so, re-link the existing profile.
					if email := UserEmail(r.Context()); email != "" {
						owner, lookupErr := db.FindProfileByUsername(r.Context(), deps.DB, username)
						if lookupErr == nil && owner != nil {
							oldOwner, emailErr := db.FindProfileByAuthEmail(r.Context(), deps.DB, email)
							if emailErr == nil && oldOwner != nil && oldOwner.ID == owner.ID {
								if rlErr := db.RelinkProfile(r.Context(), deps.DB, owner.ID, userID); rlErr != nil {
									log.Printf("[%s] Onboarding: re-link on username conflict: %v", TraceID(r.Context()), rlErr)
								} else {
									log.Printf("[%s] Onboarding: re-linked profile %s→%s via username conflict", TraceID(r.Context()), owner.ID, userID)
									owner.ID = userID
									RespondJSON(w, http.StatusOK, toOnboardingResponse(owner))
									return
								}
							}
						}
					}
					RespondError(w, r, http.StatusConflict, Conflict(ErrConstraintViolation, "Username is already taken"))
					return
				case db.OnboardingErrProfileExists:
					// Concurrent request already created the profile — re-fetch and return it.
					existing, fetchErr := db.GetProfileByUserID(r.Context(), deps.DB, userID)
					if fetchErr != nil || existing == nil {
						log.Printf("[%s] Onboarding: re-fetch after concurrent create: %v", TraceID(r.Context()), fetchErr)
						RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to create profile"))
						return
					}
					RespondJSON(w, http.StatusOK, toOnboardingResponse(existing))
					return
				}
			}
			log.Printf("[%s] Onboarding: create profile: %v", TraceID(r.Context()), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to create profile"))
			return
		}

		// Create a subscription for the new user. Plan selection is
		// centralized in db.DefaultPlanID to avoid duplicating the
		// billing-enabled logic.
		if _, err := db.CreateSubscription(r.Context(), deps.DB, profile.ID, db.DefaultPlanID(deps.BillingEnabled)); err != nil {
			log.Printf("[%s] Onboarding: create subscription: %v", TraceID(r.Context()), err)
			// Non-fatal: the profile was created successfully. Log the error
			// but still return the profile so the user isn't blocked.
		}

		RespondJSON(w, http.StatusCreated, toOnboardingResponse(profile))
	}
}

func toOnboardingResponse(p *db.Profile) onboardingResponse {
	return onboardingResponse{
		ID:        p.ID,
		Username:  p.Username,
		CreatedAt: p.CreatedAt,
	}
}

// isASCIILetterOrDigit returns true for a-z, A-Z, 0-9.
// We intentionally restrict to ASCII to prevent Unicode homoglyph attacks
// (e.g. Cyrillic 'а' impersonating Latin 'a').
func isASCIILetterOrDigit(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9')
}

// validateUsername enforces basic username rules:
// - 3–32 ASCII characters
// - ASCII letters, digits, underscores, hyphens only
// - must start with an ASCII letter or digit
func validateUsername(username string) error {
	n := len(username) // byte length == char length for ASCII-only strings
	if n < shared.UsernameMinLength {
		return fmt.Errorf("username must be at least %d characters", shared.UsernameMinLength)
	}
	if n > shared.UsernameMaxLength {
		return fmt.Errorf("username must be at most %d characters", shared.UsernameMaxLength)
	}
	for i := 0; i < n; i++ {
		b := username[i]
		if b > 127 {
			return errors.New("username may only contain ASCII letters, digits, underscores, and hyphens")
		}
		if i == 0 && !isASCIILetterOrDigit(b) {
			return errors.New("username must start with a letter or digit")
		}
		if !isASCIILetterOrDigit(b) && b != '_' && b != '-' {
			return errors.New("username may only contain ASCII letters, digits, underscores, and hyphens")
		}
	}
	return nil
}
