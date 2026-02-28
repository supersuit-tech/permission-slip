package api

import (
	"log"
	"net/http"
	"sort"

	"github.com/supersuit-tech/permission-slip-web/db"
)

// notificationPreferenceResponse is a single channel preference.
type notificationPreferenceResponse struct {
	Channel string `json:"channel"`
	Enabled bool   `json:"enabled"`
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

		RespondJSON(w, http.StatusOK, notificationPreferencesResponse{
			Preferences: buildPreferencesResponse(prefs),
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
			Preferences: buildPreferencesResponse(prefs),
		})
	}
}

// buildPreferencesResponse converts DB preferences into the API response,
// defaulting missing channels to enabled.
func buildPreferencesResponse(prefs []db.NotificationPreference) []notificationPreferenceResponse {
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
		result = append(result, notificationPreferenceResponse{
			Channel: ch,
			Enabled: enabled,
		})
	}
	return result
}
