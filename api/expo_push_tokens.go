package api

import (
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/supersuit-tech/permission-slip-web/db"
)

// Maximum length for an Expo push token string.
const maxExpoPushTokenLength = 256

// --- Request / response types ---

type createExpoPushTokenRequest struct {
	Token string `json:"token"`
}

type expoPushTokenResponse struct {
	ID        int64     `json:"id"`
	Token     string    `json:"token"`
	CreatedAt time.Time `json:"created_at"`
}

type expoPushTokenListResponse struct {
	Tokens []expoPushTokenResponse `json:"tokens"`
}

type deleteExpoPushTokenResponse struct {
	ID        int64     `json:"id"`
	DeletedAt time.Time `json:"deleted_at"`
}

// --- Routes ---

// RegisterExpoPushTokenRoutes adds Expo push token endpoints to the mux.
func init() {
	RegisterRouteGroup(RegisterExpoPushTokenRoutes)
}

func RegisterExpoPushTokenRoutes(mux *http.ServeMux, deps *Deps) {
	requireProfile := RequireProfile(deps)
	mux.Handle("GET /expo-push-tokens", requireProfile(handleListExpoPushTokens(deps)))
	mux.Handle("POST /expo-push-tokens", requireProfile(handleCreateExpoPushToken(deps)))
	mux.Handle("DELETE /expo-push-tokens/{token_id}", requireProfile(handleDeleteExpoPushToken(deps)))
}

// --- Handlers ---

func handleListExpoPushTokens(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile := Profile(r.Context())

		tokens, err := db.ListExpoPushTokensByUserID(r.Context(), deps.DB, profile.ID)
		if err != nil {
			log.Printf("[%s] ListExpoPushTokens: %v", TraceID(r.Context()), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to list Expo push tokens"))
			return
		}

		data := make([]expoPushTokenResponse, len(tokens))
		for i, t := range tokens {
			data[i] = expoPushTokenResponse{
				ID:        t.ID,
				Token:     t.Token,
				CreatedAt: t.CreatedAt,
			}
		}

		RespondJSON(w, http.StatusOK, expoPushTokenListResponse{Tokens: data})
	}
}

func handleCreateExpoPushToken(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile := Profile(r.Context())

		var req createExpoPushTokenRequest
		if !DecodeJSONOrReject(w, r, &req) {
			return
		}

		req.Token = strings.TrimSpace(req.Token)
		if req.Token == "" {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "token is required"))
			return
		}
		if len(req.Token) > maxExpoPushTokenLength {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "token too long"))
			return
		}
		if !isValidExpoPushToken(req.Token) {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "token must be a valid Expo push token (ExponentPushToken[...] or ExpoPushToken[...])"))
			return
		}

		tok, err := db.UpsertExpoPushToken(r.Context(), deps.DB, profile.ID, req.Token)
		if err != nil {
			log.Printf("[%s] CreateExpoPushToken: %v", TraceID(r.Context()), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to register Expo push token"))
			return
		}

		RespondJSON(w, http.StatusCreated, expoPushTokenResponse{
			ID:        tok.ID,
			Token:     tok.Token,
			CreatedAt: tok.CreatedAt,
		})
	}
}

func handleDeleteExpoPushToken(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile := Profile(r.Context())
		idStr := r.PathValue("token_id")

		tokenID, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil || tokenID <= 0 {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "Invalid token_id"))
			return
		}

		deleted, err := db.DeleteExpoPushToken(r.Context(), deps.DB, profile.ID, tokenID)
		if err != nil {
			log.Printf("[%s] DeleteExpoPushToken: %v", TraceID(r.Context()), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to delete Expo push token"))
			return
		}

		if !deleted {
			RespondError(w, r, http.StatusNotFound, NotFound(ErrInvalidRequest, "Expo push token not found"))
			return
		}

		RespondJSON(w, http.StatusOK, deleteExpoPushTokenResponse{
			ID:        tokenID,
			DeletedAt: time.Now().UTC(),
		})
	}
}

// isValidExpoPushToken checks that the token has a valid Expo push token
// format: either ExponentPushToken[...] or ExpoPushToken[...].
func isValidExpoPushToken(token string) bool {
	return (strings.HasPrefix(token, "ExponentPushToken[") || strings.HasPrefix(token, "ExpoPushToken[")) &&
		strings.HasSuffix(token, "]")
}
