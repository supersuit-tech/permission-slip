package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sort"

	"github.com/supersuit-tech/permission-slip-web/db"
)

// notificationPreferenceResponse is a single channel preference.
// The Available field indicates whether the user's plan supports this channel;
// when false, the frontend should show an upgrade prompt instead of a toggle.
type notificationPreferenceResponse struct {
	Channel   string `json:"channel"`
	Enabled   bool   `json:"enabled"`
	Available bool   `json:"available"`
}

// notificationPreferencesResponse wraps the list.
type notificationPreferencesResponse struct {
	Preferences []notificationPreferenceResponse `json:"preferences"`
}

// updateNotificationPreferencesRequest is the JSON body for PUT /profile/notification-preferences.
type updateNotificationPreferencesRequest struct {
	Preferences []notificationPreferenceUpdate `json:"preferences"`
}

type notificationPreferenceUpdate struct {
	Channel string `json:"channel"`
	Enabled bool   `json:"enabled"`
}

// validChannels is the set of notification channels the DB schema allows.
// This is the single source of truth — allChannels is derived from it.
var validChannels = map[string]bool{
	"email":    true,
	"web-push": true,
	"sms":      true,
}

// allChannels is derived from validChannels to avoid duplication.
var allChannels = func() []string {
	channels := make([]string, 0, len(validChannels))
	for ch := range validChannels {
		channels = append(channels, ch)
	}
	sort.Strings(channels) // deterministic order: email, sms, web-push
	return channels
}()

func handleGetNotificationPreferences(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile := Profile(r.Context())

		prefs, err := db.GetNotificationPreferences(r.Context(), deps.DB, profile.ID)
		if err != nil {
			log.Printf("[%s] handleGetNotificationPreferences: %v", TraceID(r.Context()), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to load notification preferences"))
			return
		}

		smsAvailable := isUserOnPaidPlan(r.Context(), deps.DB, profile.ID)

		RespondJSON(w, http.StatusOK, notificationPreferencesResponse{
			Preferences: buildPreferencesResponse(prefs, smsAvailable),
		})
	}
}

func handleUpdateNotificationPreferences(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile := Profile(r.Context())

		var req updateNotificationPreferencesRequest
		if !DecodeJSONOrReject(w, r, &req) {
			return
		}

		if len(req.Preferences) == 0 {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "At least one preference is required"))
			return
		}

		// Validate and reject duplicates.
		seen := make(map[string]bool)
		for _, p := range req.Preferences {
			if !validChannels[p.Channel] {
				RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "Invalid notification channel: "+p.Channel))
				return
			}
			if seen[p.Channel] {
				RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "Duplicate notification channel: "+p.Channel))
				return
			}
			seen[p.Channel] = true
		}

		// Gate SMS behind paid tier: reject enabling SMS on the free plan.
		if seen["sms"] {
			for _, p := range req.Preferences {
				if p.Channel == "sms" && p.Enabled {
					if aborted := checkSMSPlanAccess(r.Context(), w, r, deps.DB, profile.ID); aborted {
						return
					}
					break
				}
			}
		}

		for _, p := range req.Preferences {
			if err := db.UpsertNotificationPreference(r.Context(), deps.DB, profile.ID, p.Channel, p.Enabled); err != nil {
				log.Printf("[%s] handleUpdateNotificationPreferences: upsert %q: %v", TraceID(r.Context()), p.Channel, err)
				RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to update notification preferences"))
				return
			}
		}

		// Re-fetch and return updated preferences.
		prefs, err := db.GetNotificationPreferences(r.Context(), deps.DB, profile.ID)
		if err != nil {
			log.Printf("[%s] handleUpdateNotificationPreferences: re-fetch: %v", TraceID(r.Context()), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to update notification preferences"))
			return
		}

		smsAvailable := isUserOnPaidPlan(r.Context(), deps.DB, profile.ID)

		RespondJSON(w, http.StatusOK, notificationPreferencesResponse{
			Preferences: buildPreferencesResponse(prefs, smsAvailable),
		})
	}
}

// checkSMSPlanAccess verifies that the user's plan allows SMS notifications.
// Returns true if the request should be aborted (free tier or error).
func checkSMSPlanAccess(ctx context.Context, w http.ResponseWriter, r *http.Request, d db.DBTX, userID string) bool {
	sub, err := db.GetSubscriptionByUserID(ctx, d, userID)
	if err != nil {
		log.Printf("[%s] checkSMSPlanAccess: get subscription: %v", TraceID(r.Context()), err)
		CaptureError(r.Context(), fmt.Errorf("check SMS plan access: %w", err))
		RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to check plan"))
		return true
	}
	if sub == nil || sub.PlanID == db.PlanFree {
		resp := Forbidden(ErrSMSRequiresPaidPlan, "SMS notifications require a paid plan. Upgrade to Pay As You Go to enable SMS.")
		resp.Error.Details = map[string]any{
			"current_plan": db.PlanFree,
			"required_plan": db.PlanPayAsYouGo,
		}
		RespondError(w, r, http.StatusForbidden, resp)
		return true
	}
	return false
}

// isUserOnPaidPlan returns true if the user has an active subscription on a
// non-free plan. Used to determine channel availability in preference responses.
func isUserOnPaidPlan(ctx context.Context, d db.DBTX, userID string) bool {
	sub, err := db.GetSubscriptionByUserID(ctx, d, userID)
	if err != nil || sub == nil {
		return false
	}
	return sub.PlanID != db.PlanFree
}

// paidOnlyChannels lists channels that require a paid plan.
var paidOnlyChannels = map[string]bool{
	"sms": true,
}

// buildPreferencesResponse converts DB preferences into the API response,
// defaulting missing channels to enabled. The smsAvailable flag controls
// whether plan-gated channels (SMS) are marked as available.
func buildPreferencesResponse(prefs []db.NotificationPreference, smsAvailable bool) []notificationPreferenceResponse {
	channelMap := make(map[string]bool)
	for _, p := range prefs {
		channelMap[p.Channel] = p.Enabled
	}

	result := make([]notificationPreferenceResponse, 0, len(allChannels))
	for _, ch := range allChannels {
		enabled, exists := channelMap[ch]
		if !exists {
			enabled = true // missing rows default to enabled
		}
		available := true
		if paidOnlyChannels[ch] {
			available = smsAvailable
		}
		result = append(result, notificationPreferenceResponse{
			Channel:   ch,
			Enabled:   enabled,
			Available: available,
		})
	}
	return result
}
