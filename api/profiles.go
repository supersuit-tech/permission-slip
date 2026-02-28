package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"time"

	"github.com/supersuit-tech/permission-slip-web/db"
)

// profileResponse is the JSON shape returned by GET /profile.
type profileResponse struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Email     *string   `json:"email,omitempty"`
	Phone     *string   `json:"phone,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// updateProfileRaw uses json.RawMessage to detect which fields were actually
// provided in the request body, enabling true PATCH semantics (absent fields
// are left unchanged, explicit null clears the field).
type updateProfileRaw struct {
	Email json.RawMessage `json:"email"`
	Phone json.RawMessage `json:"phone"`
}

// RegisterProfileRoutes adds profile-related endpoints to the mux.
func RegisterProfileRoutes(mux *http.ServeMux, deps *Deps) {
	requireProfile := RequireProfile(deps)

	// Profile CRUD
	mux.Handle("GET /profile", requireProfile(handleGetProfile()))
	mux.Handle("PATCH /profile", requireProfile(handleUpdateProfile(deps)))

	// Notification preferences (sub-resource of profile)
	mux.Handle("GET /profile/notification-preferences", requireProfile(handleGetNotificationPreferences(deps)))
	mux.Handle("PUT /profile/notification-preferences", requireProfile(handleUpdateNotificationPreferences(deps)))
}

func handleGetProfile() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile := Profile(r.Context())

		RespondJSON(w, http.StatusOK, profileResponse{
			ID:        profile.ID,
			Username:  profile.Username,
			Email:     profile.Email,
			Phone:     profile.Phone,
			CreatedAt: profile.CreatedAt,
		})
	}
}

var (
	emailRegex = regexp.MustCompile(`(?i)^[^@\s]+@[^@\s]+\.[^@\s]+$`)
	phoneRegex = regexp.MustCompile(`^\+[1-9][0-9]{0,14}$`)
)

// parseOptionalString parses a json.RawMessage that may be a string, null,
// or absent. Returns (value, wasProvided, error):
//   - absent field   → (nil, false, nil)
//   - explicit null  → (nil, true, nil)
//   - empty string   → (nil, true, nil)   — treated as clearing the field
//   - valid string   → (&s, true, nil)
//   - wrong type     → (nil, false, error) — e.g. number, array, object
func parseOptionalString(raw json.RawMessage) (*string, bool, error) {
	if len(raw) == 0 {
		return nil, false, nil // field not present in JSON
	}
	if isRawJSONNull(raw) {
		return nil, true, nil // explicitly null → clear the field
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, false, fmt.Errorf("expected a string or null, got %s", raw)
	}
	if s == "" {
		return nil, true, nil // empty string → clear the field
	}
	return &s, true, nil
}

func handleUpdateProfile(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile := Profile(r.Context())

		var req updateProfileRaw
		if !DecodeJSONOrReject(w, r, &req) {
			return
		}

		// Start from current values and only override provided fields.
		email := profile.Email
		phone := profile.Phone

		if parsed, provided, err := parseOptionalString(req.Email); err != nil {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "email: "+err.Error()))
			return
		} else if provided {
			if parsed != nil && !emailRegex.MatchString(*parsed) {
				RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "Invalid email format"))
				return
			}
			email = parsed
		}

		if parsed, provided, err := parseOptionalString(req.Phone); err != nil {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "phone: "+err.Error()))
			return
		} else if provided {
			if parsed != nil && !phoneRegex.MatchString(*parsed) {
				RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "Invalid phone format — use E.164 (e.g. +15551234567)"))
				return
			}
			phone = parsed
		}

		if err := db.UpdateProfileContactFields(r.Context(), deps.DB, profile.ID, email, phone); err != nil {
			log.Printf("[%s] handleUpdateProfile: %v", TraceID(r.Context()), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to update profile"))
			return
		}

		// Re-fetch profile to return updated state.
		updated, err := db.GetProfileByUserID(r.Context(), deps.DB, profile.ID)
		if err != nil {
			log.Printf("[%s] handleUpdateProfile: re-fetch: %v", TraceID(r.Context()), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to update profile"))
			return
		}

		RespondJSON(w, http.StatusOK, profileResponse{
			ID:        updated.ID,
			Username:  updated.Username,
			Email:     updated.Email,
			Phone:     updated.Phone,
			CreatedAt: updated.CreatedAt,
		})
	}
}
