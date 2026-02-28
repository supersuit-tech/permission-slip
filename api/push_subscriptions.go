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
)

// --- Request / response types ---

type createPushSubscriptionRequest struct {
	Endpoint string `json:"endpoint"`
	P256dh   string `json:"p256dh"`
	Auth     string `json:"auth"`
}

type pushSubscriptionResponse struct {
	ID        int64     `json:"id"`
	Endpoint  string    `json:"endpoint"`
	CreatedAt time.Time `json:"created_at"`
}

type pushSubscriptionListResponse struct {
	Subscriptions []pushSubscriptionResponse `json:"subscriptions"`
}

type deletePushSubscriptionResponse struct {
	ID        int64     `json:"id"`
	DeletedAt time.Time `json:"deleted_at"`
}

type vapidPublicKeyResponse struct {
	PublicKey string `json:"public_key"`
}

// --- Routes ---

// RegisterPushSubscriptionRoutes adds push subscription endpoints to the mux.
func RegisterPushSubscriptionRoutes(mux *http.ServeMux, deps *Deps) {
	requireProfile := RequireProfile(deps)
	mux.Handle("GET /push-subscriptions", requireProfile(handleListPushSubscriptions(deps)))
	mux.Handle("POST /push-subscriptions", requireProfile(handleCreatePushSubscription(deps)))
	mux.Handle("DELETE /push-subscriptions/{subscription_id}", requireProfile(handleDeletePushSubscription(deps)))
	mux.Handle("GET /config/vapid-key", requireProfile(handleGetVAPIDKey(deps)))
}

// --- Handlers ---

func handleListPushSubscriptions(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile := Profile(r.Context())

		subs, err := db.ListPushSubscriptionsByUserID(r.Context(), deps.DB, profile.ID)
		if err != nil {
			log.Printf("[%s] ListPushSubscriptions: %v", TraceID(r.Context()), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to list push subscriptions"))
			return
		}

		data := make([]pushSubscriptionResponse, len(subs))
		for i, s := range subs {
			data[i] = pushSubscriptionResponse{
				ID:        s.ID,
				Endpoint:  s.Endpoint,
				CreatedAt: s.CreatedAt,
			}
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

		sub, err := db.UpsertPushSubscription(r.Context(), deps.DB, profile.ID, req.Endpoint, req.P256dh, req.Auth)
		if err != nil {
			log.Printf("[%s] CreatePushSubscription: %v", TraceID(r.Context()), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to create push subscription"))
			return
		}

		RespondJSON(w, http.StatusCreated, pushSubscriptionResponse{
			ID:        sub.ID,
			Endpoint:  sub.Endpoint,
			CreatedAt: sub.CreatedAt,
		})
	}
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
