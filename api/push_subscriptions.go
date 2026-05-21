package api

import (
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/supersuit-tech/permission-slip/db"
)

// Maximum lengths for push subscription fields.
const (
	maxExpoTokenLength = 256 // Expo tokens are ~50 chars; generous limit
)

// --- Request / response types ---

type createPushSubscriptionRequest struct {
	// Type must be "expo" or "mobile-push".
	Type string `json:"type"`

	// ExpoToken is required for all mobile push subscriptions.
	ExpoToken string `json:"expo_token"`
}

type pushSubscriptionResponse struct {
	ID        int64     `json:"id"`
	Channel   string    `json:"channel"`
	ExpoToken *string   `json:"expo_token,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type pushSubscriptionListResponse struct {
	Subscriptions []pushSubscriptionResponse `json:"subscriptions"`
}

type deletePushSubscriptionResponse struct {
	ID        int64     `json:"id"`
	DeletedAt time.Time `json:"deleted_at"`
}

type unregisterExpoPushTokenRequest struct {
	ExpoToken string `json:"expo_token"`
}

type unregisterExpoPushTokenResponse struct {
	UnregisteredAt time.Time `json:"unregistered_at"`
}

// --- Routes ---

func init() {
	RegisterRouteGroup(RegisterPushSubscriptionRoutes)
}

// RegisterPushSubscriptionRoutes adds push subscription endpoints to the mux.
func RegisterPushSubscriptionRoutes(mux *http.ServeMux, deps *Deps) {
	requireProfile := RequireProfile(deps)
	mux.Handle("GET /push-subscriptions", requireProfile(handleListPushSubscriptions(deps)))
	mux.Handle("POST /push-subscriptions", requireProfile(handleCreatePushSubscription(deps)))
	mux.Handle("DELETE /push-subscriptions/{subscription_id}", requireProfile(handleDeletePushSubscription(deps)))
	mux.Handle("POST /push-subscriptions/unregister", requireProfile(handleUnregisterExpoPushToken(deps)))
}

// --- Handlers ---

func handleListPushSubscriptions(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile := Profile(r.Context())

		// Optional channel filter: ?channel=mobile-push
		channel := r.URL.Query().Get("channel")

		var data []pushSubscriptionResponse

		switch channel {
		case "", db.PushChannelMobilePush:
			tokens, err := db.ListExpoPushTokensByUserID(r.Context(), deps.DB, profile.ID)
			if err != nil {
				log.Printf("[%s] ListPushSubscriptions (mobile-push): %v", TraceID(r.Context()), err)
				CaptureError(r.Context(), err)
				RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to list push subscriptions"))
				return
			}
			for _, t := range tokens {
				data = append(data, expoTokenToResponse(t))
			}
		default:
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "channel must be \"mobile-push\""))
			return
		}

		if data == nil {
			data = []pushSubscriptionResponse{}
		}
		RespondJSON(w, http.StatusOK, pushSubscriptionListResponse{Subscriptions: data})
	}
}

func handleCreatePushSubscription(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile := Profile(r.Context())

		var req createPushSubscriptionRequest
		if !DecodeJSONOrReject(w, r, &req) {
			return
		}

		switch req.Type {
		case "expo", "mobile-push", "":
			handleCreateExpo(w, r, deps, profile.ID, req)
		default:
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "type must be \"expo\" or \"mobile-push\""))
		}
	}
}

func handleCreateExpo(w http.ResponseWriter, r *http.Request, deps *Deps, userID string, req createPushSubscriptionRequest) {
	if req.ExpoToken == "" {
		RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "expo_token is required"))
		return
	}
	if !isValidExpoToken(req.ExpoToken) {
		RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest,
			"expo_token must be a valid Expo push token (e.g. ExponentPushToken[...] or ExpoPushToken[...])"))
		return
	}
	if len(req.ExpoToken) > maxExpoTokenLength {
		RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "expo_token too long"))
		return
	}

	tok, err := db.UpsertExpoPushToken(r.Context(), deps.DB, userID, req.ExpoToken)
	if err != nil {
		log.Printf("[%s] CreateExpoPushToken: %v", TraceID(r.Context()), err)
		CaptureError(r.Context(), err)
		RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to create push subscription"))
		return
	}

	RespondJSON(w, http.StatusCreated, expoTokenToResponse(*tok))
}

func handleDeletePushSubscription(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile := Profile(r.Context())
		idStr := r.PathValue("subscription_id")

		subID, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil || subID <= 0 {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "Invalid subscription_id"))
			return
		}

		deleted, err := db.DeleteExpoPushToken(r.Context(), deps.DB, profile.ID, subID)
		if err != nil {
			log.Printf("[%s] DeletePushSubscription: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to delete push subscription"))
			return
		}

		if !deleted {
			RespondError(w, r, http.StatusNotFound, NotFound(ErrInvalidRequest, "Push subscription not found"))
			return
		}

		RespondJSON(w, http.StatusOK, deletePushSubscriptionResponse{
			ID:        subID,
			DeletedAt: time.Now().UTC(),
		})
	}
}

// handleUnregisterExpoPushToken removes an Expo push token by value.
// This is more convenient for mobile clients than DELETE by ID, since the
// device knows its own token but may not have cached the subscription ID.
func handleUnregisterExpoPushToken(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile := Profile(r.Context())

		var req unregisterExpoPushTokenRequest
		if !DecodeJSONOrReject(w, r, &req) {
			return
		}

		if req.ExpoToken == "" {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "expo_token is required"))
			return
		}

		deleted, err := db.DeleteExpoPushTokenForUser(r.Context(), deps.DB, profile.ID, req.ExpoToken)
		if err != nil {
			log.Printf("[%s] UnregisterExpoPushToken: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to unregister push token"))
			return
		}

		if !deleted {
			RespondError(w, r, http.StatusNotFound, NotFound(ErrInvalidRequest, "Expo push token not found"))
			return
		}

		RespondJSON(w, http.StatusOK, unregisterExpoPushTokenResponse{
			UnregisteredAt: time.Now().UTC(),
		})
	}
}

// isValidExpoToken checks whether a token matches the Expo push token format:
// ExponentPushToken[...] or ExpoPushToken[...]
func isValidExpoToken(token string) bool {
	if strings.HasPrefix(token, "ExponentPushToken[") && strings.HasSuffix(token, "]") {
		return len(token) > len("ExponentPushToken[]") // must have content inside brackets
	}
	if strings.HasPrefix(token, "ExpoPushToken[") && strings.HasSuffix(token, "]") {
		return len(token) > len("ExpoPushToken[]")
	}
	return false
}

// expoTokenToResponse converts a DB ExpoPushToken to the API response.
func expoTokenToResponse(t db.ExpoPushToken) pushSubscriptionResponse {
	return pushSubscriptionResponse{
		ID:        t.ID,
		Channel:   db.PushChannelMobilePush,
		ExpoToken: &t.Token,
		CreatedAt: t.CreatedAt,
	}
}
