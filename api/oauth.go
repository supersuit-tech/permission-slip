package api

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/supersuit-tech/permission-slip-web/db"
	"github.com/supersuit-tech/permission-slip-web/oauth"
	"golang.org/x/oauth2"
)

// oauthStateTTL is the maximum lifetime of an OAuth CSRF state token.
const oauthStateTTL = 10 * time.Minute

// --- Response types ---

type oauthProviderResponse struct {
	ID             string   `json:"id"`
	Scopes         []string `json:"scopes"`
	Source         string   `json:"source"`
	HasCredentials bool     `json:"has_credentials"`
}

type oauthProviderListResponse struct {
	Providers []oauthProviderResponse `json:"providers"`
}

type oauthConnectionResponse struct {
	Provider    string    `json:"provider"`
	Scopes      []string  `json:"scopes"`
	Status      string    `json:"status"`
	ConnectedAt time.Time `json:"connected_at"`
}

type oauthConnectionListResponse struct {
	Connections []oauthConnectionResponse `json:"connections"`
}

type oauthDisconnectResponse struct {
	Provider       string    `json:"provider"`
	DisconnectedAt time.Time `json:"disconnected_at"`
}

// --- Routes ---

// RegisterOAuthRoutes adds OAuth-related endpoints to the mux.
func RegisterOAuthRoutes(mux *http.ServeMux, deps *Deps) {
	requireProfile := RequireProfile(deps)
	requireSession := RequireSession(deps)

	mux.Handle("GET /v1/oauth/providers", requireProfile(handleListOAuthProviders(deps)))
	mux.Handle("GET /v1/oauth/{provider}/authorize", requireProfile(handleOAuthAuthorize(deps)))
	mux.Handle("GET /v1/oauth/{provider}/callback", requireSession(handleOAuthCallback(deps)))
	mux.Handle("GET /v1/oauth/connections", requireProfile(handleListOAuthConnections(deps)))
	mux.Handle("DELETE /v1/oauth/connections/{provider}", requireProfile(handleDeleteOAuthConnection(deps)))
}

// --- CSRF state helpers ---

// oauthStateSecret returns the HMAC key to sign/verify OAuth state JWTs.
// Prefers OAuthStateSecret; falls back to SupabaseJWTSecret.
func oauthStateSecret(deps *Deps) string {
	if deps.OAuthStateSecret != "" {
		return deps.OAuthStateSecret
	}
	return deps.SupabaseJWTSecret
}

// createOAuthState produces a short-lived signed JWT encoding the user ID
// and OAuth provider. This prevents CSRF and state-fixation attacks on the
// callback endpoint.
func createOAuthState(deps *Deps, userID, provider string) (string, error) {
	secret := oauthStateSecret(deps)
	if secret == "" {
		return "", fmt.Errorf("no OAuth state secret configured")
	}
	now := time.Now()
	claims := jwt.MapClaims{
		"sub":      userID,
		"provider": provider,
		"iat":      now.Unix(),
		"exp":      now.Add(oauthStateTTL).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// verifyOAuthState validates the signed state JWT and returns the encoded
// user ID and provider. Returns an error if the signature is invalid, the
// token is expired, or the claims are missing.
func verifyOAuthState(deps *Deps, stateStr string) (userID, provider string, err error) {
	secret := oauthStateSecret(deps)
	if secret == "" {
		return "", "", fmt.Errorf("no OAuth state secret configured")
	}
	token, err := jwt.Parse(stateStr, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(secret), nil
	}, jwt.WithValidMethods([]string{"HS256"}), jwt.WithExpirationRequired())
	if err != nil {
		return "", "", fmt.Errorf("invalid state token: %w", err)
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return "", "", fmt.Errorf("invalid state claims")
	}
	sub, _ := claims["sub"].(string)
	prov, _ := claims["provider"].(string)
	if sub == "" || prov == "" {
		return "", "", fmt.Errorf("state token missing required claims")
	}
	return sub, prov, nil
}

// --- Helpers ---

// newOAuth2Config builds an oauth2.Config from a registry Provider and the
// callback URL derived from deps. Centralises config construction so the
// authorize and callback handlers stay in sync.
func newOAuth2Config(deps *Deps, provider oauth.Provider) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     provider.ClientID,
		ClientSecret: provider.ClientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  provider.AuthorizeURL,
			TokenURL: provider.TokenURL,
		},
		RedirectURL: oauthCallbackURL(deps, provider.ID),
		Scopes:      provider.Scopes,
	}
}

// deduplicateScopes returns a new slice with duplicate scope strings removed,
// preserving original order.
func deduplicateScopes(scopes []string) []string {
	seen := make(map[string]struct{}, len(scopes))
	out := make([]string, 0, len(scopes))
	for _, s := range scopes {
		if _, ok := seen[s]; !ok {
			seen[s] = struct{}{}
			out = append(out, s)
		}
	}
	return out
}

// --- Handlers ---

// handleListOAuthProviders returns all registered OAuth providers with their
// configuration status. This lets the frontend discover which providers are
// available, which are ready to use, and which need BYOA setup.
func handleListOAuthProviders(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.OAuthProviders == nil {
			RespondJSON(w, http.StatusOK, oauthProviderListResponse{Providers: []oauthProviderResponse{}})
			return
		}

		providers := deps.OAuthProviders.List()
		data := make([]oauthProviderResponse, len(providers))
		for i, p := range providers {
			data[i] = oauthProviderResponse{
				ID:             p.ID,
				Scopes:         p.Scopes,
				Source:         string(p.Source),
				HasCredentials: p.HasClientCredentials(),
			}
		}
		RespondJSON(w, http.StatusOK, oauthProviderListResponse{Providers: data})
	}
}

// handleOAuthAuthorize generates the authorization URL for a given provider
// and redirects the user's browser to the provider's consent screen.
func handleOAuthAuthorize(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.OAuthProviders == nil {
			RespondError(w, r, http.StatusServiceUnavailable, ServiceUnavailable("OAuth not available"))
			return
		}

		providerID := r.PathValue("provider")
		if providerID == "" {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "provider is required"))
			return
		}

		provider, ok := deps.OAuthProviders.Get(providerID)
		if !ok {
			RespondError(w, r, http.StatusNotFound, NotFound(ErrOAuthProviderNotFound, "OAuth provider not found"))
			return
		}
		if !provider.HasClientCredentials() {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrOAuthProviderUnconfigured,
				"OAuth provider is not configured. Supply client credentials via BYOA settings."))
			return
		}

		profile := Profile(r.Context())
		state, err := createOAuthState(deps, profile.ID, providerID)
		if err != nil {
			log.Printf("[%s] OAuthAuthorize: create state: %v", TraceID(r.Context()), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to initiate OAuth flow"))
			return
		}

		cfg := newOAuth2Config(deps, provider)

		// Request additional scopes from query params if provided.
		if extraScopes := r.URL.Query()["scope"]; len(extraScopes) > 0 {
			cfg.Scopes = deduplicateScopes(append(cfg.Scopes, extraScopes...))
		}

		authURL := cfg.AuthCodeURL(state, oauth2.AccessTypeOffline)
		http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
	}
}

// handleOAuthCallback exchanges the authorization code for tokens, stores them
// in the vault, and creates an oauth_connections row.
func handleOAuthCallback(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.OAuthProviders == nil || deps.Vault == nil || deps.DB == nil {
			RespondError(w, r, http.StatusServiceUnavailable, ServiceUnavailable("OAuth not available"))
			return
		}

		providerID := r.PathValue("provider")

		// Check for error from provider (e.g. user denied consent)
		if errCode := r.URL.Query().Get("error"); errCode != "" {
			errDesc := r.URL.Query().Get("error_description")
			if errDesc == "" {
				errDesc = errCode
			}
			redirectToFrontend(w, r, deps, providerID, "error", errDesc)
			return
		}

		// Validate CSRF state
		stateStr := r.URL.Query().Get("state")
		if stateStr == "" {
			redirectToFrontend(w, r, deps, providerID, "error", "Missing state parameter")
			return
		}
		stateUserID, stateProvider, err := verifyOAuthState(deps, stateStr)
		if err != nil {
			log.Printf("[%s] OAuthCallback: invalid state: %v", TraceID(r.Context()), err)
			redirectToFrontend(w, r, deps, providerID, "error", "Invalid or expired state token")
			return
		}
		if stateProvider != providerID {
			log.Printf("[%s] OAuthCallback: provider mismatch: state=%s path=%s", TraceID(r.Context()), stateProvider, providerID)
			redirectToFrontend(w, r, deps, providerID, "error", "Provider mismatch")
			return
		}

		// Verify the session user matches the state user
		sessionUserID := UserID(r.Context())
		if sessionUserID != stateUserID {
			log.Printf("[%s] OAuthCallback: user mismatch: state=%s session=%s", TraceID(r.Context()), stateUserID, sessionUserID)
			redirectToFrontend(w, r, deps, providerID, "error", "Session mismatch")
			return
		}

		// Look up profile for the user
		profile, err := db.GetProfileByUserID(r.Context(), deps.DB, sessionUserID)
		if err != nil || profile == nil {
			log.Printf("[%s] OAuthCallback: profile lookup failed: %v", TraceID(r.Context()), err)
			redirectToFrontend(w, r, deps, providerID, "error", "Profile not found")
			return
		}

		code := r.URL.Query().Get("code")
		if code == "" {
			redirectToFrontend(w, r, deps, providerID, "error", "Missing authorization code")
			return
		}

		provider, ok := deps.OAuthProviders.Get(providerID)
		if !ok {
			redirectToFrontend(w, r, deps, providerID, "error", "Provider not found")
			return
		}

		cfg := newOAuth2Config(deps, provider)

		// Exchange authorization code for tokens.
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()
		token, err := cfg.Exchange(ctx, code)
		if err != nil {
			log.Printf("[%s] OAuthCallback: token exchange failed: %v", TraceID(r.Context()), err)
			redirectToFrontend(w, r, deps, providerID, "error", "Token exchange failed")
			return
		}

		// Store tokens in vault within a transaction.
		if err := storeOAuthTokens(r.Context(), deps, profile.ID, providerID, provider.Scopes, token); err != nil {
			log.Printf("[%s] OAuthCallback: store tokens: %v", TraceID(r.Context()), err)
			redirectToFrontend(w, r, deps, providerID, "error", "Failed to store connection")
			return
		}

		redirectToFrontend(w, r, deps, providerID, "success", "")
	}
}

// storeOAuthTokens persists the OAuth tokens in the vault and creates or
// updates the oauth_connections row. If a connection already exists for this
// user+provider, it is replaced (re-authorization flow).
func storeOAuthTokens(ctx context.Context, deps *Deps, userID, providerID string, scopes []string, token *oauth2.Token) error {
	tx, owned, err := db.BeginOrContinue(ctx, deps.DB)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	if owned {
		defer db.RollbackTx(ctx, tx) //nolint:errcheck // best-effort cleanup
	}

	// Delete existing connection if present (re-auth flow).
	existing, err := db.DeleteOAuthConnection(ctx, tx, userID, providerID)
	if err != nil {
		var connErr *db.OAuthConnectionError
		if !errors.As(err, &connErr) || connErr.Code != db.OAuthConnectionErrNotFound {
			return fmt.Errorf("delete existing connection: %w", err)
		}
		// Not found is fine — first connection.
	}
	if existing != nil {
		// Clean up old vault secrets.
		_ = deps.Vault.DeleteSecret(ctx, tx, existing.AccessTokenVaultID)
		if existing.RefreshTokenVaultID != nil {
			_ = deps.Vault.DeleteSecret(ctx, tx, *existing.RefreshTokenVaultID)
		}
	}

	connID, err := generatePrefixedID("oconn_", 16)
	if err != nil {
		return fmt.Errorf("generate connection ID: %w", err)
	}

	accessVaultID, err := deps.Vault.CreateSecret(ctx, tx, connID+"_access", []byte(token.AccessToken))
	if err != nil {
		return fmt.Errorf("vault create access token: %w", err)
	}

	var refreshVaultID *string
	if token.RefreshToken != "" {
		id, err := deps.Vault.CreateSecret(ctx, tx, connID+"_refresh", []byte(token.RefreshToken))
		if err != nil {
			return fmt.Errorf("vault create refresh token: %w", err)
		}
		refreshVaultID = &id
	}

	var tokenExpiry *time.Time
	if !token.Expiry.IsZero() {
		tokenExpiry = &token.Expiry
	}

	_, err = db.CreateOAuthConnection(ctx, tx, db.CreateOAuthConnectionParams{
		ID:                  connID,
		UserID:              userID,
		Provider:            providerID,
		AccessTokenVaultID:  accessVaultID,
		RefreshTokenVaultID: refreshVaultID,
		Scopes:              scopes,
		TokenExpiry:         tokenExpiry,
	})
	if err != nil {
		return fmt.Errorf("create connection: %w", err)
	}

	if owned {
		if err := db.CommitTx(ctx, tx); err != nil {
			return fmt.Errorf("commit: %w", err)
		}
	}
	return nil
}

// handleListOAuthConnections returns all OAuth connections for the authenticated user.
func handleListOAuthConnections(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.DB == nil {
			RespondError(w, r, http.StatusServiceUnavailable, ServiceUnavailable("Database not available"))
			return
		}

		profile := Profile(r.Context())
		conns, err := db.ListOAuthConnectionsByUser(r.Context(), deps.DB, profile.ID)
		if err != nil {
			log.Printf("[%s] ListOAuthConnections: %v", TraceID(r.Context()), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to list OAuth connections"))
			return
		}

		data := make([]oauthConnectionResponse, len(conns))
		for i, c := range conns {
			data[i] = oauthConnectionResponse{
				Provider:    c.Provider,
				Scopes:      c.Scopes,
				Status:      c.Status,
				ConnectedAt: c.CreatedAt,
			}
		}

		RespondJSON(w, http.StatusOK, oauthConnectionListResponse{Connections: data})
	}
}

// handleDeleteOAuthConnection disconnects an OAuth provider for the authenticated user.
func handleDeleteOAuthConnection(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.Vault == nil || deps.DB == nil {
			RespondError(w, r, http.StatusServiceUnavailable, ServiceUnavailable("OAuth not available"))
			return
		}

		profile := Profile(r.Context())
		providerID := r.PathValue("provider")
		if providerID == "" {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "provider is required"))
			return
		}

		tx, owned, err := db.BeginOrContinue(r.Context(), deps.DB)
		if err != nil {
			log.Printf("[%s] DeleteOAuthConnection: begin tx: %v", TraceID(r.Context()), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to disconnect OAuth provider"))
			return
		}
		if owned {
			defer db.RollbackTx(r.Context(), tx) //nolint:errcheck // best-effort cleanup
		}

		result, err := db.DeleteOAuthConnection(r.Context(), tx, profile.ID, providerID)
		if err != nil {
			var connErr *db.OAuthConnectionError
			if errors.As(err, &connErr) && connErr.Code == db.OAuthConnectionErrNotFound {
				RespondError(w, r, http.StatusNotFound, NotFound(ErrOAuthConnectionNotFound, "OAuth connection not found"))
				return
			}
			log.Printf("[%s] DeleteOAuthConnection: %v", TraceID(r.Context()), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to disconnect OAuth provider"))
			return
		}

		// Delete vault secrets (best-effort — idempotent).
		if err := deps.Vault.DeleteSecret(r.Context(), tx, result.AccessTokenVaultID); err != nil {
			log.Printf("[%s] DeleteOAuthConnection: vault delete access: %v", TraceID(r.Context()), err)
		}
		if result.RefreshTokenVaultID != nil {
			if err := deps.Vault.DeleteSecret(r.Context(), tx, *result.RefreshTokenVaultID); err != nil {
				log.Printf("[%s] DeleteOAuthConnection: vault delete refresh: %v", TraceID(r.Context()), err)
			}
		}

		if owned {
			if err := db.CommitTx(r.Context(), tx); err != nil {
				log.Printf("[%s] DeleteOAuthConnection: commit: %v", TraceID(r.Context()), err)
				RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to disconnect OAuth provider"))
				return
			}
		}

		RespondJSON(w, http.StatusOK, oauthDisconnectResponse{
			Provider:       providerID,
			DisconnectedAt: time.Now().UTC(),
		})
	}
}

// --- URL helpers ---

// oauthCallbackURL constructs the OAuth callback URL for a given provider.
// Uses OAuthRedirectBaseURL if set, otherwise falls back to BaseURL.
func oauthCallbackURL(deps *Deps, providerID string) string {
	base := deps.OAuthRedirectBaseURL
	if base == "" {
		base = deps.BaseURL
	}
	return base + "/api/v1/oauth/" + url.PathEscape(providerID) + "/callback"
}

// redirectToFrontend redirects the user's browser to the frontend settings
// page with a status and optional error message as query parameters. The
// oauth_tab param lets the frontend auto-navigate to the connections section.
func redirectToFrontend(w http.ResponseWriter, r *http.Request, deps *Deps, provider, status, errMsg string) {
	base := deps.OAuthRedirectBaseURL
	if base == "" {
		base = deps.BaseURL
	}
	u := base + "/settings"
	params := url.Values{}
	params.Set("oauth_provider", provider)
	params.Set("oauth_status", status)
	params.Set("oauth_tab", "connections")
	if errMsg != "" {
		params.Set("oauth_error", errMsg)
	}
	http.Redirect(w, r, u+"?"+params.Encode(), http.StatusTemporaryRedirect)
}
