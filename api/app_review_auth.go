package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func init() {
	RegisterRouteGroup(registerAppReviewAuthRoutes)
}

func registerAppReviewAuthRoutes(mux *http.ServeMux, deps *Deps) {
	mux.HandleFunc("POST /v1/auth/app-review-login", handleAppReviewLogin(deps))
}

// handleAppReviewLogin authenticates the App Store review account using a
// pre-shared OTP code. This bypasses the normal Supabase OTP flow because
// Apple's review team cannot receive real OTP emails.
//
// The endpoint validates the email and OTP against the APP_REVIEW_EMAIL and
// APP_REVIEW_OTP environment variables. If they match, it uses the Supabase
// Admin API to generate a magic link for the review user, then exchanges the
// link token for a real session.
//
// Security considerations:
//   - Only enabled when both APP_REVIEW_EMAIL and APP_REVIEW_OTP are set
//   - The OTP is a static secret — treat it like a password in env config
//   - Rate limited by the standard API rate limiter (applied upstream)
//   - Returns generic errors to avoid leaking whether the feature is enabled
func handleAppReviewLogin(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cfg := loadAppReviewConfig()
		if !cfg.enabled {
			RespondError(w, r, http.StatusUnauthorized, Unauthorized(ErrInvalidToken, "Invalid credentials"))
			return
		}

		var body struct {
			Email string `json:"email"`
			OTP   string `json:"otp"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "Invalid request body"))
			return
		}

		if !strings.EqualFold(body.Email, cfg.email) || body.OTP != cfg.otp {
			RespondError(w, r, http.StatusUnauthorized, Unauthorized(ErrInvalidToken, "Invalid credentials"))
			return
		}

		if deps.SupabaseURL == "" || deps.SupabaseServiceRoleKey == "" {
			log.Printf("[%s] app-review-login: SUPABASE_URL or SUPABASE_SERVICE_ROLE_KEY not set", TraceID(r.Context()))
			RespondError(w, r, http.StatusInternalServerError, InternalError("Authentication service not configured"))
			return
		}

		// Step 1: Generate a magic link via the Supabase Admin API.
		// This creates the user if they don't exist and returns a link
		// containing a token_hash we can exchange for a session.
		linkReq := map[string]string{
			"type":  "magiclink",
			"email": body.Email,
		}
		linkBody, _ := json.Marshal(linkReq)

		generateURL := fmt.Sprintf("%s/auth/v1/admin/generate_link", deps.SupabaseURL)
		httpReq, err := http.NewRequestWithContext(r.Context(), http.MethodPost, generateURL, bytes.NewReader(linkBody))
		if err != nil {
			log.Printf("[%s] app-review-login: create generate_link request: %v", TraceID(r.Context()), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to authenticate"))
			return
		}
		httpReq.Header.Set("Authorization", "Bearer "+deps.SupabaseServiceRoleKey)
		httpReq.Header.Set("apikey", deps.SupabaseServiceRoleKey)
		httpReq.Header.Set("Content-Type", "application/json")

		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(httpReq)
		if err != nil {
			log.Printf("[%s] app-review-login: generate_link request failed: %v", TraceID(r.Context()), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to authenticate"))
			return
		}
		defer resp.Body.Close()

		respBody, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusOK {
			log.Printf("[%s] app-review-login: generate_link returned %d: %s", TraceID(r.Context()), resp.StatusCode, string(respBody))
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to authenticate"))
			return
		}

		var linkResp struct {
			ActionLink string `json:"action_link"`
		}
		if err := json.Unmarshal(respBody, &linkResp); err != nil || linkResp.ActionLink == "" {
			log.Printf("[%s] app-review-login: failed to parse generate_link response", TraceID(r.Context()))
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to authenticate"))
			return
		}

		// Step 2: Extract the token_hash from the action_link URL.
		parsed, err := url.Parse(linkResp.ActionLink)
		if err != nil {
			log.Printf("[%s] app-review-login: failed to parse action_link: %v", TraceID(r.Context()), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to authenticate"))
			return
		}

		// The token might be in the query params or the URL fragment.
		// Supabase puts it in query: ?token_hash=...&type=magiclink
		tokenHash := parsed.Query().Get("token_hash")
		tokenType := parsed.Query().Get("type")
		if tokenHash == "" {
			// Some Supabase versions put the token directly
			tokenHash = parsed.Query().Get("token")
			tokenType = "magiclink"
		}
		if tokenHash == "" {
			log.Printf("[%s] app-review-login: no token_hash in action_link: %s", TraceID(r.Context()), linkResp.ActionLink)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to authenticate"))
			return
		}
		if tokenType == "" {
			tokenType = "magiclink"
		}

		// Step 3: Exchange the token for a session via the verify endpoint.
		verifyReq := map[string]string{
			"token_hash": tokenHash,
			"type":       tokenType,
		}
		verifyBody, _ := json.Marshal(verifyReq)

		verifyURL := fmt.Sprintf("%s/auth/v1/verify", deps.SupabaseURL)
		httpReq2, err := http.NewRequestWithContext(r.Context(), http.MethodPost, verifyURL, bytes.NewReader(verifyBody))
		if err != nil {
			log.Printf("[%s] app-review-login: create verify request: %v", TraceID(r.Context()), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to authenticate"))
			return
		}
		httpReq2.Header.Set("apikey", deps.SupabaseServiceRoleKey)
		httpReq2.Header.Set("Content-Type", "application/json")

		resp2, err := client.Do(httpReq2)
		if err != nil {
			log.Printf("[%s] app-review-login: verify request failed: %v", TraceID(r.Context()), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to authenticate"))
			return
		}
		defer resp2.Body.Close()

		verifyRespBody, _ := io.ReadAll(resp2.Body)
		if resp2.StatusCode != http.StatusOK {
			log.Printf("[%s] app-review-login: verify returned %d: %s", TraceID(r.Context()), resp2.StatusCode, string(verifyRespBody))
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to authenticate"))
			return
		}

		// Step 4: Return the session tokens to the client.
		// The verify response contains access_token, refresh_token, user, etc.
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(verifyRespBody)
	}
}
