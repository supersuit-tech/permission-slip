package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

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
	"email":       true,
	"web-push":    true,
	"sms":         true,
	"mobile-push": true,
}

// allChannels is derived from validChannels to avoid duplication.
var allChannels = func() []string {
	channels := make([]string, 0, len(validChannels))
	for ch := range validChannels {
		channels = append(channels, ch)
	}
	sort.Strings(channels) // deterministic order: email, mobile-push, sms, web-push
	return channels
}()

// paidOnlyChannels lists channels that require a paid plan.
var paidOnlyChannels = map[string]bool{
	"sms": true,
}

// betaDisabledChannels lists channels that are disabled during the beta period.
// These channels are always unavailable regardless of plan.
var betaDisabledChannels = map[string]bool{
	"sms": true,
}

func handleGetNotificationPreferences(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile := Profile(r.Context())

		prefs, err := db.GetNotificationPreferences(r.Context(), deps.DB, profile.ID)
		if err != nil {
			log.Printf("[%s] handleGetNotificationPreferences: %v", TraceID(r.Context()), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to load notification preferences"))
			return
		}

		planID := userPlanID(r.Context(), deps.DB, profile.ID)

		RespondJSON(w, http.StatusOK, notificationPreferencesResponse{
			Preferences: buildPreferencesResponse(prefs, planID),
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

		// Gate beta-disabled channels: reject enabling any channel disabled during beta.
		for _, p := range req.Preferences {
			if p.Enabled && betaDisabledChannels[p.Channel] {
				label := cases.Title(language.English).String(strings.NewReplacer("-", " ").Replace(p.Channel))
				msg := fmt.Sprintf("%s notifications are not yet available. They will be enabled after the beta period.", label)
				RespondError(w, r, http.StatusForbidden, Forbidden(ErrChannelUnavailableBeta, msg))
				return
			}
		}

		// Look up plan for the response (SMS availability in response payload).
		planID := userPlanID(r.Context(), deps.DB, profile.ID)

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

		RespondJSON(w, http.StatusOK, notificationPreferencesResponse{
			Preferences: buildPreferencesResponse(prefs, planID),
		})
	}
}

// userPlanID returns the user's current plan ID, defaulting to "free" if the
// subscription is missing or the lookup fails. Errors are logged but not
// surfaced — this is a read-only helper for response enrichment.
func userPlanID(ctx context.Context, d db.DBTX, userID string) string {
	sub, err := db.GetSubscriptionByUserID(ctx, d, userID)
	if err != nil {
		log.Printf("userPlanID: get subscription for %s: %v", userID, err)
		return db.PlanFree
	}
	if sub == nil {
		return db.PlanFree
	}
	return sub.PlanID
}

// isPaidPlan returns true if the plan ID represents a paid (non-free) plan.
func isPaidPlan(planID string) bool {
	return planID != db.PlanFree
}

// buildPreferencesResponse converts DB preferences into the API response,
// defaulting missing channels to enabled. The planID controls whether
// plan-gated channels (SMS) are marked as available.
func buildPreferencesResponse(prefs []db.NotificationPreference, planID string) []notificationPreferenceResponse {
	channelMap := make(map[string]bool)
	for _, p := range prefs {
		channelMap[p.Channel] = p.Enabled
	}

	paid := isPaidPlan(planID)
	result := make([]notificationPreferenceResponse, 0, len(allChannels))
	for _, ch := range allChannels {
		enabled, exists := channelMap[ch]
		if !exists {
			if betaDisabledChannels[ch] {
				enabled = false // beta-disabled channels default to off
			} else {
				enabled = true // missing rows default to enabled
			}
		}
		available := true
		if betaDisabledChannels[ch] {
			available = false // always unavailable during beta
		} else if paidOnlyChannels[ch] {
			available = paid
		}
		result = append(result, notificationPreferenceResponse{
			Channel:   ch,
			Enabled:   enabled,
			Available: available,
		})
	}
	return result
}
