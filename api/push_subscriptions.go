package api

import (
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/supersuit-tech/permission-slip-web/db"
)

// Maximum lengths for push subscription fields.
const (
	maxPushEndpointLength = 2048
	maxPushP256dhLength   = 256 // P-256 public key is 65 bytes → 88 chars base64url
	maxPushAuthLength     = 64  // Auth secret is 16 bytes → 22 chars base64url
	maxExpoTokenLength    = 256 // Expo tokens are ~50 chars; generous limit
)

// --- Request / response types ---

type createPushSubscriptionRequest struct {
	// Type discriminates between web-push and expo subscriptions.
	// Defaults to "web-push" when omitted for backward compatibility.
	Type string `json:"type"`

	// Web Push fields (required when type is "web-push" or omitted).
	Endpoint string `json:"endpoint"`
	P256dh   string `json:"p256dh"`
	Auth     string `json:"auth"`

	// Expo field (required when type is "expo").
	ExpoToken string `json:"expo_token"`
}

type pushSubscriptionResponse struct {
	ID        int64     `json:"id"`
	Channel   string    `json:"channel"`
	Endpoint  *string   `json:"endpoint,omitempty"`
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

type vapidPublicKeyResponse struct {
	PublicKey string `json:"public_key"`
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
	mux.Handle("GET /config/vapid-key", requireProfile(handleGetVAPIDKey(deps)))
}

// --- Handlers ---

func handleListPushSubscriptions(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile := Profile(r.Context())

		// Optional channel filter: ?channel=web-push or ?channel=mobile-push
		channel := r.URL.Query().Get("channel")

		var data []pushSubscriptionResponse

		switch channel {
		case "":
			// Return both web-push and expo tokens.
			subs, err := db.ListWebPushSubscriptionsByUserID(r.Context(), deps.DB, profile.ID)
			if err != nil {
				log.Printf("[%s] ListPushSubscriptions (web-push): %v", TraceID(r.Context()), err)
				RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to list push subscriptions"))
				return
			}
			for _, s := range subs {
				data = append(data, toPushSubscriptionResponse(s))
			}
			tokens, err := db.ListExpoPushTokensByUserID(r.Context(), deps.DB, profile.ID)
			if err != nil {
				log.Printf("[%s] ListPushSubscriptions (mobile-push): %v", TraceID(r.Context()), err)
				RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to list push subscriptions"))
				return
			}
			for _, t := range tokens {
				data = append(data, expoTokenToResponse(t))
			}
		case db.PushChannelWebPush:
			subs, err := db.ListWebPushSubscriptionsByUserID(r.Context(), deps.DB, profile.ID)
			if err != nil {
				log.Printf("[%s] ListPushSubscriptions: %v", TraceID(r.Context()), err)
				RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to list push subscriptions"))
				return
			}
			for _, s := range subs {
				data = append(data, toPushSubscriptionResponse(s))
			}
		case db.PushChannelMobilePush:
			tokens, err := db.ListExpoPushTokensByUserID(r.Context(), deps.DB, profile.ID)
			if err != nil {
				log.Printf("[%s] ListPushSubscriptions: %v", TraceID(r.Context()), err)
				RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to list push subscriptions"))
				return
			}
			for _, t := range tokens {
				data = append(data, expoTokenToResponse(t))
			}
		default:
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "channel must be \"web-push\" or \"mobile-push\""))
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

		// Default to web-push for backward compatibility.
		if req.Type == "" {
			req.Type = "web-push"
		}

		switch req.Type {
		case "web-push":
			handleCreateWebPush(w, r, deps, profile.ID, req)
		case "expo", "mobile-push":
			handleCreateExpo(w, r, deps, profile.ID, req)
		default:
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "type must be \"web-push\", \"expo\", or \"mobile-push\""))
		}
	}
}

func handleCreateWebPush(w http.ResponseWriter, r *http.Request, deps *Deps, userID string, req createPushSubscriptionRequest) {
	if req.Endpoint == "" {
		RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "endpoint is required"))
		return
	}
	if !strings.HasPrefix(req.Endpoint, "https://") {
		RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "endpoint must be an HTTPS URL"))
		return
	}
	if len(req.Endpoint) > maxPushEndpointLength {
		RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "endpoint URL too long"))
		return
	}
	if req.P256dh == "" {
		RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "p256dh is required"))
		return
	}
	if len(req.P256dh) > maxPushP256dhLength {
		RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "p256dh value too long"))
		return
	}
	if req.Auth == "" {
		RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "auth is required"))
		return
	}
	if len(req.Auth) > maxPushAuthLength {
		RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "auth value too long"))
		return
	}

	sub, err := db.UpsertPushSubscription(r.Context(), deps.DB, userID, req.Endpoint, req.P256dh, req.Auth)
	if err != nil {
		log.Printf("[%s] CreatePushSubscription: %v", TraceID(r.Context()), err)
		RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to create push subscription"))
		return
	}

	RespondJSON(w, http.StatusCreated, toPushSubscriptionResponse(*sub))
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

		deleted, err := db.DeletePushSubscription(r.Context(), deps.DB, profile.ID, subID)
		if err != nil {
			log.Printf("[%s] DeletePushSubscription: %v", TraceID(r.Context()), err)
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

func handleGetVAPIDKey(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.VAPIDPublicKey == "" {
			RespondError(w, r, http.StatusServiceUnavailable, ServiceUnavailable("Web Push not configured"))
			return
		}

		RespondJSON(w, http.StatusOK, vapidPublicKeyResponse{
			PublicKey: deps.VAPIDPublicKey,
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

// toPushSubscriptionResponse converts a DB PushSubscription (web-push) to the API response.
func toPushSubscriptionResponse(s db.PushSubscription) pushSubscriptionResponse {
	return pushSubscriptionResponse{
		ID:        s.ID,
		Channel:   s.Channel,
		Endpoint:  s.Endpoint,
		ExpoToken: s.ExpoToken,
		CreatedAt: s.CreatedAt,
	}
}

// expoTokenToResponse converts a DB ExpoPushToken to the unified API response.
func expoTokenToResponse(t db.ExpoPushToken) pushSubscriptionResponse {
	return pushSubscriptionResponse{
		ID:        t.ID,
		Channel:   db.PushChannelMobilePush,
		ExpoToken: &t.Token,
		CreatedAt: t.CreatedAt,
	}
}
