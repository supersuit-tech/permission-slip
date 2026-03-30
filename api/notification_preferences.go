package api

import (
	"log"
	"net/http"
	"sort"

	"github.com/supersuit-tech/permission-slip-web/db"
)

// notificationPreferenceResponse is a single channel preference.
// The Available field indicates whether the channel can be used;
// when false, the frontend shows a "coming soon" badge instead of a toggle.
// Note: SMS is excluded from the response entirely (not just marked unavailable)
// when the server has no SMS sender configured.
// Web push is excluded from the response entirely so it is not configurable in the product UI.
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

func handleGetNotificationPreferences(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile := Profile(r.Context())

		prefs, err := db.GetNotificationPreferences(r.Context(), deps.DB, profile.ID)
		if err != nil {
			log.Printf("[%s] handleGetNotificationPreferences: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to load notification preferences"))
			return
		}

		RespondJSON(w, http.StatusOK, notificationPreferencesResponse{
			Preferences: buildPreferencesResponse(prefs, deps.SMSEnabled),
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

		// Web push preferences are not exposed in the API (no in-product UI). Reject any
		// update so stale clients cannot write preference rows.
		for _, p := range req.Preferences {
			if p.Channel == "web-push" {
				RespondError(w, r, http.StatusForbidden, Forbidden(ErrChannelNotConfigured, "Web push notification preferences are not available."))
				return
			}
		}

		// Gate SMS: reject any SMS preference change when the server has no SMS sender configured
		// or the operator has explicitly hidden SMS (SMS_NOTIFICATIONS_HIDDEN=true).
		// This prevents stale clients from writing sms=false rows that would override the
		// "missing rows default to enabled" behaviour when SMS is later configured.
		for _, p := range req.Preferences {
			if p.Channel == "sms" && !deps.SMSEnabled {
				RespondError(w, r, http.StatusForbidden, Forbidden(ErrChannelNotConfigured, "SMS notifications are not configured on this server."))
				return
			}
		}

		for _, p := range req.Preferences {
			if err := db.UpsertNotificationPreference(r.Context(), deps.DB, profile.ID, p.Channel, p.Enabled); err != nil {
				log.Printf("[%s] handleUpdateNotificationPreferences: upsert %q: %v", TraceID(r.Context()), p.Channel, err)
				CaptureError(r.Context(), err)
				RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to update notification preferences"))
				return
			}
		}

		// Re-fetch and return updated preferences.
		prefs, err := db.GetNotificationPreferences(r.Context(), deps.DB, profile.ID)
		if err != nil {
			log.Printf("[%s] handleUpdateNotificationPreferences: re-fetch: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to update notification preferences"))
			return
		}

		RespondJSON(w, http.StatusOK, notificationPreferencesResponse{
			Preferences: buildPreferencesResponse(prefs, deps.SMSEnabled),
		})
	}
}

// buildPreferencesResponse converts DB preferences into the API response,
// defaulting missing channels to enabled. SMS is excluded entirely
// when smsEnabled is false (server has no SMS sender or operator hid it).
func buildPreferencesResponse(prefs []db.NotificationPreference, smsEnabled bool) []notificationPreferenceResponse {
	channelMap := make(map[string]bool)
	for _, p := range prefs {
		channelMap[p.Channel] = p.Enabled
	}

	result := make([]notificationPreferenceResponse, 0, len(allChannels))
	for _, ch := range allChannels {
		// Skip SMS entirely when the server doesn't have it configured.
		if ch == "sms" && !smsEnabled {
			continue
		}
		// Web push is not user-configurable in the product; omit from the list.
		if ch == "web-push" {
			continue
		}
		enabled, exists := channelMap[ch]
		if !exists {
			enabled = true // missing rows default to enabled
		}
		available := true
		result = append(result, notificationPreferenceResponse{
			Channel:   ch,
			Enabled:   enabled,
			Available: available,
		})
	}
	return result
}
