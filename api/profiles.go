package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"time"

	"github.com/supersuit-tech/permission-slip/db"
)

// profileResponse is the JSON shape returned by GET /profile.
type profileResponse struct {
	ID             string    `json:"id"`
	Username       string    `json:"username"`
	Email          *string   `json:"email,omitempty"`
	Phone          *string   `json:"phone,omitempty"`
	MarketingOptIn bool      `json:"marketing_opt_in"`
	CreatedAt      time.Time `json:"created_at"`
}

func toProfileResponse(p *db.Profile) profileResponse {
	return profileResponse{
		ID:             p.ID,
		Username:       p.Username,
		Email:          p.Email,
		Phone:          p.Phone,
		MarketingOptIn: p.MarketingOptIn,
		CreatedAt:      p.CreatedAt,
	}
}

// updateProfileRaw uses json.RawMessage to detect which fields were actually
// provided in the request body, enabling true PATCH semantics (absent fields
// are left unchanged, explicit null clears the field).
type updateProfileRaw struct {
	Email          json.RawMessage `json:"email"`
	Phone          json.RawMessage `json:"phone"`
	MarketingOptIn json.RawMessage `json:"marketing_opt_in"`
}

func init() {
	RegisterRouteGroup(RegisterProfileRoutes)
}

// RegisterProfileRoutes adds profile-related endpoints to the mux.
func RegisterProfileRoutes(mux *http.ServeMux, deps *Deps) {
	requireProfile := RequireProfile(deps)

	// Profile CRUD
	mux.Handle("GET /profile", requireProfile(handleGetProfile()))
	mux.Handle("PATCH /profile", requireProfile(handleUpdateProfile(deps)))
	mux.Handle("DELETE /profile", requireProfile(handleDeleteAccount(deps)))

	// Notification preferences (sub-resource of profile)
	mux.Handle("GET /profile/notification-preferences", requireProfile(handleGetNotificationPreferences(deps)))
	mux.Handle("PUT /profile/notification-preferences", requireProfile(handleUpdateNotificationPreferences(deps)))

	// Data retention policy
	mux.Handle("GET /profile/data-retention", requireProfile(handleGetDataRetention(deps)))
}

func handleGetProfile() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile := Profile(r.Context())

		RespondJSON(w, http.StatusOK, toProfileResponse(profile))
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

// parseOptionalBool parses a json.RawMessage that may be a bool or absent.
// Returns (value, wasProvided, error):
//   - absent field   → (nil, false, nil)
//   - true/false     → (&b, true, nil)
//   - null           → (nil, false, error) — boolean fields are non-nullable
//   - wrong type     → (nil, false, error)
func parseOptionalBool(raw json.RawMessage) (*bool, bool, error) {
	if len(raw) == 0 {
		return nil, false, nil // field not present in JSON
	}
	if isRawJSONNull(raw) {
		return nil, false, fmt.Errorf("expected a boolean, got null")
	}
	var b bool
	if err := json.Unmarshal(raw, &b); err != nil {
		return nil, false, fmt.Errorf("expected a boolean, got %s", raw)
	}
	return &b, true, nil
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
		var marketingOptIn *bool // nil means "not provided" → leave unchanged

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

		if parsed, provided, err := parseOptionalBool(req.MarketingOptIn); err != nil {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "marketing_opt_in: "+err.Error()))
			return
		} else if provided {
			marketingOptIn = parsed
		}

		if err := db.UpdateProfileFields(r.Context(), deps.DB, profile.ID, email, phone, marketingOptIn); err != nil {
			log.Printf("[%s] handleUpdateProfile: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to update profile"))
			return
		}

		// Re-fetch profile to return updated state.
		updated, err := db.GetProfileByUserID(r.Context(), deps.DB, profile.ID)
		if err != nil {
			log.Printf("[%s] handleUpdateProfile: re-fetch: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to update profile"))
			return
		}

		RespondJSON(w, http.StatusOK, toProfileResponse(updated))
	}
}

// deleteAccountRequest is the JSON body for DELETE /profile.
// The confirmation field acts as a safety check to prevent accidental deletion.
type deleteAccountRequest struct {
	Confirmation string `json:"confirmation"`
}

// deleteAccountResponse is the JSON shape returned by DELETE /profile.
type deleteAccountResponse struct {
	Deleted bool   `json:"deleted"`
	Message string `json:"message"`
}

func handleDeleteAccount(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile := Profile(r.Context())

		var req deleteAccountRequest
		if !DecodeJSONOrReject(w, r, &req) {
			return
		}

		if req.Confirmation != "DELETE" {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest,
				`To confirm account deletion, set "confirmation" to "DELETE"`))
			return
		}

		// Run deletion inside a transaction so vault cleanup and profile
		// deletion are atomic.
		tx, owned, err := db.BeginOrContinue(r.Context(), deps.DB)
		if err != nil {
			log.Printf("[%s] handleDeleteAccount: begin tx: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to delete account"))
			return
		}
		if owned {
			defer db.RollbackTx(r.Context(), tx)
		}

		// Build vault delete function if vault is available.
		var vaultDeleteFn func(ctx context.Context, d db.DBTX, secretID string) error
		if deps.Vault != nil {
			vaultDeleteFn = deps.Vault.DeleteSecret
		}

		if err := db.DeleteAccount(r.Context(), tx, profile.ID, vaultDeleteFn); err != nil {
			log.Printf("[%s] handleDeleteAccount: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to delete account"))
			return
		}

		if owned {
			if err := db.CommitTx(r.Context(), tx); err != nil {
				log.Printf("[%s] handleDeleteAccount: commit: %v", TraceID(r.Context()), err)
				CaptureError(r.Context(), err)
				RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to delete account"))
				return
			}
		}

		// Delete the Supabase Auth user. This is best-effort after the
		// profile has been deleted — even if this fails, the user can no
		// longer access any application data.
		if err := deleteSupabaseAuthUser(r.Context(), deps, profile.ID); err != nil {
			log.Printf("[%s] handleDeleteAccount: supabase auth cleanup (non-fatal): %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
		}

		RespondJSON(w, http.StatusOK, deleteAccountResponse{
			Deleted: true,
			Message: "Account and all associated data have been permanently deleted.",
		})
	}
}

// dataRetentionResponse is the JSON shape returned by GET /profile/data-retention.
type dataRetentionResponse struct {
	PlanID                 string     `json:"plan_id"`
	PlanName               string     `json:"plan_name"`
	AuditRetentionDays     int        `json:"audit_retention_days"`
	EffectiveRetentionDays int        `json:"effective_retention_days"`
	GracePeriodEndsAt      *time.Time `json:"grace_period_ends_at,omitempty"`
}

func handleGetDataRetention(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile := Profile(r.Context())

		sp, err := db.GetSubscriptionWithPlan(r.Context(), deps.DB, profile.ID)
		if err != nil {
			log.Printf("[%s] handleGetDataRetention: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to load data retention policy"))
			return
		}

		// Fallback for users without a subscription (shouldn't happen, but be safe).
		// Keep these defaults in sync with the free plan in plans seed data.
		if sp == nil {
			RespondJSON(w, http.StatusOK, dataRetentionResponse{
				PlanID:                 db.PlanFree,
				PlanName:               "Free",
				AuditRetentionDays:     7,
				EffectiveRetentionDays: 7,
			})
			return
		}

		RespondJSON(w, http.StatusOK, dataRetentionResponse{
			PlanID:                 sp.Plan.ID,
			PlanName:               sp.Plan.Name,
			AuditRetentionDays:     sp.Plan.AuditRetentionDays,
			EffectiveRetentionDays: sp.EffectiveRetentionDays(),
			GracePeriodEndsAt:      sp.GracePeriodEndsAt(),
		})
	}
}

// uuidRegex validates that a string is a well-formed UUID v4 (lowercase hex).
// Used to prevent path traversal when constructing Supabase Admin API URLs.
var uuidRegex = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// deleteSupabaseAuthUser calls the Supabase Admin API to delete the auth user.
// Requires SupabaseURL and SupabaseServiceRoleKey to be configured. If either
// is missing, it logs a warning and returns nil (no-op).
func deleteSupabaseAuthUser(ctx context.Context, deps *Deps, userID string) error {
	if deps.SupabaseURL == "" || deps.SupabaseServiceRoleKey == "" {
		log.Printf("[%s] Warning: SUPABASE_URL or SUPABASE_SERVICE_ROLE_KEY not set; skipping auth user deletion", TraceID(ctx))
		return nil
	}

	if !uuidRegex.MatchString(userID) {
		return fmt.Errorf("invalid user ID format: %q", userID)
	}

	url := fmt.Sprintf("%s/auth/v1/admin/users/%s", deps.SupabaseURL, userID)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+deps.SupabaseServiceRoleKey)
	req.Header.Set("apikey", deps.SupabaseServiceRoleKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("supabase admin API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("supabase admin API returned %d for user %s", resp.StatusCode, userID)
	}
	return nil
}
