package api

import (
	"log"
	"net/http"
	"sort"

	"github.com/supersuit-tech/permission-slip/db"
)

// validNotificationTypes is the set of notification_type values the DB allows.
var validNotificationTypes = map[string]bool{
	db.NotificationTypeStandingExecution: true,
}

var allNotificationTypes = func() []string {
	types := make([]string, 0, len(validNotificationTypes))
	for t := range validNotificationTypes {
		types = append(types, t)
	}
	sort.Strings(types)
	return types
}()

type notificationTypePreferenceResponse struct {
	NotificationType string `json:"notification_type"`
	Enabled          bool   `json:"enabled"`
}

type notificationTypePreferencesResponse struct {
	Preferences []notificationTypePreferenceResponse `json:"preferences"`
}

type updateNotificationTypePreferencesRequest struct {
	Preferences []notificationTypePreferenceUpdate `json:"preferences"`
}

type notificationTypePreferenceUpdate struct {
	NotificationType string `json:"notification_type"`
	Enabled          bool   `json:"enabled"`
}

func handleGetNotificationTypePreferences(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile := Profile(r.Context())

		prefs, err := db.GetNotificationTypePreferences(r.Context(), deps.DB, profile.ID)
		if err != nil {
			log.Printf("[%s] handleGetNotificationTypePreferences: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to load notification type preferences"))
			return
		}

		RespondJSON(w, http.StatusOK, notificationTypePreferencesResponse{
			Preferences: buildNotificationTypePreferencesResponse(prefs),
		})
	}
}

func handleUpdateNotificationTypePreferences(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile := Profile(r.Context())

		var req updateNotificationTypePreferencesRequest
		if !DecodeJSONOrReject(w, r, &req) {
			return
		}

		if len(req.Preferences) == 0 {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "At least one preference is required"))
			return
		}

		seen := make(map[string]bool)
		for _, p := range req.Preferences {
			if !validNotificationTypes[p.NotificationType] {
				RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "Invalid notification type: "+p.NotificationType))
				return
			}
			if seen[p.NotificationType] {
				RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "Duplicate notification type: "+p.NotificationType))
				return
			}
			seen[p.NotificationType] = true
		}

		for _, p := range req.Preferences {
			if err := db.UpsertNotificationTypePreference(r.Context(), deps.DB, profile.ID, p.NotificationType, p.Enabled); err != nil {
				log.Printf("[%s] handleUpdateNotificationTypePreferences: upsert %q: %v", TraceID(r.Context()), p.NotificationType, err)
				CaptureError(r.Context(), err)
				RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to update notification type preferences"))
				return
			}
		}

		prefs, err := db.GetNotificationTypePreferences(r.Context(), deps.DB, profile.ID)
		if err != nil {
			log.Printf("[%s] handleUpdateNotificationTypePreferences: re-fetch: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to load notification type preferences"))
			return
		}

		RespondJSON(w, http.StatusOK, notificationTypePreferencesResponse{
			Preferences: buildNotificationTypePreferencesResponse(prefs),
		})
	}
}

func buildNotificationTypePreferencesResponse(prefs []db.NotificationTypePreference) []notificationTypePreferenceResponse {
	prefMap := make(map[string]bool)
	for _, p := range prefs {
		prefMap[p.NotificationType] = p.Enabled
	}

	result := make([]notificationTypePreferenceResponse, 0, len(allNotificationTypes))
	for _, nt := range allNotificationTypes {
		enabled, exists := prefMap[nt]
		if !exists {
			enabled = true
		}
		result = append(result, notificationTypePreferenceResponse{
			NotificationType: nt,
			Enabled:          enabled,
		})
	}
	return result
}
