// OAuth 2.0 authorization flow handlers.
//
// Flow overview:
//  1. Frontend redirects user to GET /v1/oauth/{provider}/authorize
//  2. Server generates a signed CSRF state token and redirects to the provider
//  3. User grants consent on the provider's consent screen
//  4. Provider redirects to GET /v1/oauth/{provider}/callback with auth code
//  5. Server exchanges the code for tokens, encrypts them in the vault,
//     and creates an oauth_connections row
//  6. Server redirects to /settings?oauth_status=success (or error)
//
// Security:
//   - CSRF protection via signed JWT state tokens (HS256, 10-min TTL)
//   - State encodes user ID + provider to prevent session fixation
//   - Callback verifies session user matches the state token's user
//   - Tokens stored server-side in Supabase Vault (AES-256-GCM)
//   - Provider path params validated against ProviderIDPattern
//   - No tokens or secrets are ever returned in API responses
package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/supersuit-tech/permission-slip-web/connectors"
	"github.com/supersuit-tech/permission-slip-web/db"
	"github.com/supersuit-tech/permission-slip-web/oauth"
	"golang.org/x/oauth2"
)

// oauthStateTTL is the maximum lifetime of an OAuth CSRF state token.
const oauthStateTTL = 10 * time.Minute

// shopSubdomainPattern matches valid Shopify store subdomains: lowercase
// alphanumeric and hyphens, not starting or ending with a hyphen, max 63 chars.
// Used to validate the "shop" query parameter before substituting it into URLs
// to prevent URL injection or SSRF.
var shopSubdomainPattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)

// isReservedOAuthParam returns true if the parameter name is a reserved
// OAuth 2.0 param that must not be overridden by AuthorizeParams.
// Uses the canonical list from connectors.ReservedAuthorizeParams.
func isReservedOAuthParam(name string) bool {
	return connectors.ReservedAuthorizeParams[name]
}

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

// createOAuthState produces a short-lived signed JWT encoding the user ID,
// OAuth provider, requested scopes, and optional shop subdomain. This prevents
// CSRF and state-fixation attacks on the callback endpoint and carries the
// final scope list so the callback can store exactly what was requested
// (including any extra scopes from the authorize query params).
//
// The shop parameter is used by providers with per-shop URLs (e.g. Shopify)
// so the callback can reconstruct the token endpoint. Pass "" for providers
// with static URLs.
func createOAuthState(deps *Deps, userID, provider string, scopes []string, shop string) (string, error) {
	secret := oauthStateSecret(deps)
	if secret == "" {
		return "", fmt.Errorf("no OAuth state secret configured")
	}
	now := time.Now()
	claims := jwt.MapClaims{
		"sub":      userID,
		"provider": provider,
		"scopes":   scopes,
		"iat":      now.Unix(),
		"exp":      now.Add(oauthStateTTL).Unix(),
	}
	if shop != "" {
		claims["shop"] = shop
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// oauthState holds the decoded claims from a verified OAuth state token.
type oauthState struct {
	UserID   string
	Provider string
	Scopes   []string
	// Shop is the Shopify store subdomain, used to reconstruct per-shop
	// OAuth URLs. Empty for providers with static endpoints.
	Shop string
}

// verifyOAuthState validates the signed state JWT and returns the encoded
// user ID, provider, requested scopes, and optional shop subdomain. Returns
// an error if the signature is invalid, the token is expired, or the claims
// are missing.
func verifyOAuthState(deps *Deps, stateStr string) (*oauthState, error) {
	secret := oauthStateSecret(deps)
	if secret == "" {
		return nil, fmt.Errorf("no OAuth state secret configured")
	}
	token, err := jwt.Parse(stateStr, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(secret), nil
	}, jwt.WithValidMethods([]string{"HS256"}), jwt.WithExpirationRequired())
	if err != nil {
		return nil, fmt.Errorf("invalid state token: %w", err)
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid state claims")
	}
	sub, _ := claims["sub"].(string)
	prov, _ := claims["provider"].(string)
	if sub == "" || prov == "" {
		return nil, fmt.Errorf("state token missing required claims")
	}
	// Parse scopes from the state token.
	var scopes []string
	if rawScopes, ok := claims["scopes"].([]any); ok {
		for _, s := range rawScopes {
			if str, ok := s.(string); ok {
				scopes = append(scopes, str)
			}
		}
	}
	shop, _ := claims["shop"].(string)
	return &oauthState{UserID: sub, Provider: prov, Scopes: scopes, Shop: shop}, nil
}

// --- Helpers ---

// newOAuth2Config builds an oauth2.Config from a registry Provider and the
// callback URL derived from deps. Centralises config construction so the
// authorize and callback handlers stay in sync.
//
// The Scopes slice is defensively copied so callers can append without
// mutating the provider's data (even though the registry already deep-copies).
func newOAuth2Config(deps *Deps, provider oauth.Provider) *oauth2.Config {
	scopes := make([]string, len(provider.Scopes))
	copy(scopes, provider.Scopes)
	return &oauth2.Config{
		ClientID:     provider.ClientID,
		ClientSecret: provider.ClientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  provider.AuthorizeURL,
			TokenURL: provider.TokenURL,
		},
		RedirectURL: oauthCallbackURL(deps, provider.ID),
		Scopes:      scopes,
	}
}

// validProviderID checks that a provider path parameter matches the expected
// format (lowercase alphanumeric, hyphens, underscores). Returns false for
// empty or malformed IDs. This prevents attacker-controlled path segments
// from appearing in redirect URLs or log messages without validation.
func validProviderID(id string) bool {
	return id != "" && oauth.ProviderIDPattern.MatchString(id)
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

// --- Per-shop URL helpers ---

// providerNeedsShop returns true if the provider's authorize or token URL
// contains a {shop} placeholder that must be resolved before use.
func providerNeedsShop(p oauth.Provider) bool {
	return strings.Contains(p.AuthorizeURL, "{shop}") || strings.Contains(p.TokenURL, "{shop}")
}

// resolveShopURLs replaces {shop} placeholders in the provider's authorize and
// token URLs with the given shop subdomain. The shop value must already be
// validated (see shopSubdomainPattern). Returns a copy of the provider with
// resolved URLs.
func resolveShopURLs(p oauth.Provider, shop string) oauth.Provider {
	p.AuthorizeURL = strings.ReplaceAll(p.AuthorizeURL, "{shop}", shop)
	p.TokenURL = strings.ReplaceAll(p.TokenURL, "{shop}", shop)
	return p
}

// validateShopSubdomain checks that a shop value is a valid Shopify subdomain.
// Accepts either a bare subdomain ("mystore") or a full domain
// ("mystore.myshopify.com") and returns the normalized subdomain.
func validateShopSubdomain(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimRight(raw, "/")
	raw = strings.ToLower(raw)

	// Accept full domain form.
	if strings.HasSuffix(raw, ".myshopify.com") {
		raw = strings.TrimSuffix(raw, ".myshopify.com")
	} else if strings.Contains(raw, ".") {
		return "", fmt.Errorf("shop must be a subdomain (e.g. \"mystore\") or full domain (e.g. \"mystore.myshopify.com\")")
	}

	if raw == "" {
		return "", fmt.Errorf("shop is required")
	}
	if !shopSubdomainPattern.MatchString(raw) {
		return "", fmt.Errorf("shop contains invalid characters")
	}
	return raw, nil
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
//
// For providers with per-shop URLs (e.g. Shopify), the "shop" query parameter
// is required and specifies the store subdomain. It is validated, used to
// resolve URL templates, and included in the state token so the callback can
// reconstruct the token endpoint.
func handleOAuthAuthorize(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.OAuthProviders == nil {
			RespondError(w, r, http.StatusServiceUnavailable, ServiceUnavailable("OAuth not available"))
			return
		}

		providerID := r.PathValue("provider")
		if !validProviderID(providerID) {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "invalid provider ID"))
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

		// Resolve per-shop URL templates (e.g. Shopify).
		var shop string
		if providerNeedsShop(provider) {
			rawShop := r.URL.Query().Get("shop")
			var err error
			shop, err = validateShopSubdomain(rawShop)
			if err != nil {
				RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest,
					fmt.Sprintf("invalid shop parameter: %v", err)))
				return
			}
			provider = resolveShopURLs(provider, shop)
		}

		cfg := newOAuth2Config(deps, provider)

		// Request additional scopes from query params if provided.
		if extraScopes := r.URL.Query()["scope"]; len(extraScopes) > 0 {
			cfg.Scopes = deduplicateScopes(append(cfg.Scopes, extraScopes...))
		}

		// Create the CSRF state token after computing the final scope list so
		// the callback can store exactly which scopes were requested.
		profile := Profile(r.Context())
		state, err := createOAuthState(deps, profile.ID, providerID, cfg.Scopes, shop)
		if err != nil {
			log.Printf("[%s] OAuthAuthorize: create state: %v", TraceID(r.Context()), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to initiate OAuth flow"))
			return
		}

		// Build auth URL with standard params + any provider-specific params
		// (e.g. Atlassian's audience=api.atlassian.com for 3LO, Slack's
		// comma-separated scope override).
		authOpts := []oauth2.AuthCodeOption{oauth2.AccessTypeOffline}
		for k, v := range provider.AuthorizeParams {
			if isReservedOAuthParam(k) {
				log.Printf("[%s] OAuthAuthorize: skipping reserved param %q from provider %q AuthorizeParams", TraceID(r.Context()), k, providerID)
				continue
			}
			authOpts = append(authOpts, oauth2.SetAuthURLParam(k, v))
		}
		authURL := cfg.AuthCodeURL(state, authOpts...)
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
		if !validProviderID(providerID) {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "invalid provider ID"))
			return
		}

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
		state, err := verifyOAuthState(deps, stateStr)
		if err != nil {
			log.Printf("[%s] OAuthCallback: invalid state: %v", TraceID(r.Context()), err)
			redirectToFrontend(w, r, deps, providerID, "error", "Invalid or expired state token")
			return
		}
		if state.Provider != providerID {
			log.Printf("[%s] OAuthCallback: provider mismatch: state=%s path=%s", TraceID(r.Context()), state.Provider, providerID)
			redirectToFrontend(w, r, deps, providerID, "error", "Provider mismatch")
			return
		}

		// Verify the session user matches the state user
		sessionUserID := UserID(r.Context())
		if sessionUserID != state.UserID {
			log.Printf("[%s] OAuthCallback: user mismatch: state=%s session=%s", TraceID(r.Context()), state.UserID, sessionUserID)
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

		// Resolve per-shop URL templates using the shop from the state token.
		if state.Shop != "" && providerNeedsShop(provider) {
			provider = resolveShopURLs(provider, state.Shop)
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

		// Store tokens in vault within a transaction. Use scopes from the
		// state token (set during authorize) so extra scopes are preserved.
		storedScopes := state.Scopes
		if len(storedScopes) == 0 {
			storedScopes = provider.Scopes // fallback for pre-existing state tokens
		}

		// Build extra data to persist alongside the tokens. For providers
		// with per-shop URLs (e.g. Shopify), store the shop_domain so it's
		// available at execution time as a credential.
		var stateExtraData map[string]string
		// NOTE: state.Shop is validated and normalized during the authorize step
		// to be a bare subdomain (e.g. "mystore"), so appending ".myshopify.com"
		// here is intentional and produces the full domain.
		if state.Shop != "" {
			stateExtraData = map[string]string{"shop_domain": state.Shop + ".myshopify.com"}
		}

		// Run the provider's post-exchange enricher (if any). Enrichers fetch
		// supplemental data not included in the token response (e.g. account IDs,
		// base URLs) and return it for storage in extra_data so connectors can
		// access it at execution time. A returned error is treated as a hard
		// failure: storing a connection without required credentials would leave
		// the user with a "Connected" status that produces confusing errors on use.
		if enricher, ok := postOAuthEnrichers[providerID]; ok {
			extra, err := enricher(ctx, token.AccessToken)
			if err != nil {
				log.Printf("[%s] OAuthCallback: post-OAuth enrichment for %q failed: %v", TraceID(r.Context()), providerID, err)
				redirectToFrontend(w, r, deps, providerID, "error", "Could not retrieve account information — please try again")
				return
			}
			if stateExtraData == nil {
				stateExtraData = make(map[string]string)
			}
			for k, v := range extra {
				stateExtraData[k] = v
			}
		}

		if err := storeOAuthTokens(r.Context(), deps, profile.ID, providerID, storedScopes, token, stateExtraData); err != nil {
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
//
// stateExtra contains additional key-value pairs to persist in extra_data
// alongside any fields extracted from the token response. This is used for
// data carried through the state token (e.g. shop_domain for Shopify).
// It may be nil when no state-derived extra data is needed.
func storeOAuthTokens(ctx context.Context, deps *Deps, userID, providerID string, scopes []string, token *oauth2.Token, stateExtra map[string]string) error {
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

	// Extract provider-specific extra data from the token response.
	// For example, Salesforce includes instance_url which connectors
	// need to construct API base URLs. State-derived extra data (e.g.
	// Shopify's shop_domain) is merged in after token extraction.
	extraData := extractTokenExtraData(token, stateExtra)

	_, err = db.CreateOAuthConnection(ctx, tx, db.CreateOAuthConnectionParams{
		ID:                  connID,
		UserID:              userID,
		Provider:            providerID,
		AccessTokenVaultID:  accessVaultID,
		RefreshTokenVaultID: refreshVaultID,
		Scopes:              scopes,
		TokenExpiry:         tokenExpiry,
		ExtraData:           extraData,
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

// tokenExtraKeys lists the extra fields from OAuth token responses that should
// be persisted in extra_data. These are provider-specific fields that connectors
// need at execution time (e.g. Salesforce's instance_url).
var tokenExtraKeys = []string{"instance_url"}

// extractTokenExtraData pulls known extra fields from an OAuth token response
// and merges in state-derived extra data, then marshals as JSON for storage.
// Returns nil if no relevant extra data is present (keeps the column NULL for
// most providers).
//
// Values are validated before storage: URLs must be well-formed HTTPS to prevent
// storing attacker-controlled values that could be used for SSRF at execution time.
// State-derived data (e.g. shop_domain) has already been validated during the
// authorize step, so it is merged directly.
func extractTokenExtraData(token *oauth2.Token, stateExtra map[string]string) json.RawMessage {
	extra := make(map[string]string)
	for _, key := range tokenExtraKeys {
		if val := token.Extra(key); val != nil {
			if s, ok := val.(string); ok && s != "" {
				// Validate URL-type extra fields to prevent storing
				// malicious values that could enable SSRF.
				if isURLExtraKey(key) {
					u, err := url.Parse(s)
					if err != nil || u.Scheme != "https" || u.Host == "" {
						log.Printf("oauth: ignoring invalid %s value in token extra data: %q", key, s)
						continue
					}
				}
				extra[key] = s
			}
		}
	}
	// Merge state-derived extra data (e.g. shop_domain from Shopify OAuth flow).
	for k, v := range stateExtra {
		extra[k] = v
	}
	if len(extra) == 0 {
		return nil
	}
	data, err := json.Marshal(extra)
	if err != nil {
		return nil
	}
	return data
}

// isURLExtraKey returns true if the given extra data key is expected to contain
// a URL value that should be validated before storage.
// Only keys in tokenExtraKeys are ever passed to this function — stateExtraData
// values (e.g. shop_domain, DocuSign's base_url) are validated at the call site
// before being added to stateExtraData and must NOT be added here.
func isURLExtraKey(key string) bool {
	return key == "instance_url"
}

// postOAuthEnricher fetches supplemental data for a provider after a successful
// token exchange. It receives a context (already deadline-constrained) and the
// fresh access token, and returns key-value pairs to merge into extra_data.
// Returning an error aborts the OAuth callback so the connection is not stored.
//
// Register enrichers in postOAuthEnrichers below. Any provider that requires
// account IDs, base URLs, or other data not included in the token response
// should have an enricher rather than inline logic in the callback handler.
type postOAuthEnricher func(ctx context.Context, accessToken string) (map[string]string, error)

// postOAuthEnrichers maps provider IDs to their post-exchange enrichers.
// Add a new entry here when a provider needs supplemental data fetched after
// the token exchange (see postOAuthEnricher above for the contract).
var postOAuthEnrichers = map[string]postOAuthEnricher{
	"docusign": func(ctx context.Context, accessToken string) (map[string]string, error) {
		return fetchDocuSignUserInfo(ctx, accessToken, docuSignUserInfoURL)
	},
}

// docuSignUserInfoURL is the endpoint used to retrieve the authenticated user's
// account information after a successful OAuth token exchange.
const docuSignUserInfoURL = "https://account.docusign.com/oauth/userinfo"

// docuSignUserInfo is the subset of the DocuSign userinfo response we care about.
type docuSignUserInfo struct {
	Accounts []struct {
		AccountID string `json:"account_id"`
		IsDefault bool   `json:"is_default"`
		BaseURI   string `json:"base_uri"`
	} `json:"accounts"`
}

// fetchDocuSignUserInfo calls DocuSign's userinfo endpoint with the given access
// token and returns extra credentials to store alongside the OAuth connection.
// The returned map contains account_id and base_url for the user's default account
// (falling back to the first account if none is marked as default).
// Both values are required by the DocuSign connector at execution time.
//
// userInfoURL is normally docuSignUserInfoURL; tests pass a local httptest server
// URL to avoid real network calls.
func fetchDocuSignUserInfo(ctx context.Context, accessToken, userInfoURL string) (map[string]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, userInfoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create userinfo request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("userinfo request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("userinfo returned HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return nil, fmt.Errorf("read userinfo response: %w", err)
	}

	var info docuSignUserInfo
	if err := json.Unmarshal(body, &info); err != nil {
		return nil, fmt.Errorf("parse userinfo response: %w", err)
	}
	if len(info.Accounts) == 0 {
		return nil, fmt.Errorf("no accounts in DocuSign userinfo response")
	}

	// Prefer the default account; fall back to the first account.
	account := info.Accounts[0]
	for _, a := range info.Accounts {
		if a.IsDefault {
			account = a
			break
		}
	}

	if account.AccountID == "" {
		return nil, fmt.Errorf("account_id missing in DocuSign userinfo response")
	}
	if account.BaseURI == "" {
		return nil, fmt.Errorf("base_uri missing in DocuSign userinfo response")
	}

	// Validate that base_uri is a DocuSign HTTPS URL before storing it
	// to prevent SSRF via a compromised or malicious userinfo response.
	parsed, err := url.Parse(account.BaseURI)
	if err != nil || parsed.Scheme != "https" || !strings.HasSuffix(strings.ToLower(parsed.Hostname()), ".docusign.net") {
		return nil, fmt.Errorf("base_uri %q is not a valid DocuSign HTTPS URL", account.BaseURI)
	}

	return map[string]string{
		"account_id": account.AccountID,
		// Append the REST API path so the connector's base_url credential
		// resolves to the correct versioned endpoint.
		"base_url": strings.TrimRight(account.BaseURI, "/") + "/restapi/v2.1",
	}, nil
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
		if !validProviderID(providerID) {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "invalid provider ID"))
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

// oauthBaseURL returns the public base URL for OAuth-related redirects.
// Prefers OAuthRedirectBaseURL (for deployments where the public URL differs
// from the internal BaseURL); falls back to BaseURL.
func oauthBaseURL(deps *Deps) string {
	if deps.OAuthRedirectBaseURL != "" {
		return deps.OAuthRedirectBaseURL
	}
	return deps.BaseURL
}

// oauthCallbackURL constructs the OAuth callback URL for a given provider.
func oauthCallbackURL(deps *Deps, providerID string) string {
	return oauthBaseURL(deps) + "/api/v1/oauth/" + url.PathEscape(providerID) + "/callback"
}

// redirectToFrontend redirects the user's browser to the frontend settings
// page with a status and optional error message as query parameters. The
// oauth_tab param lets the frontend auto-navigate to the connections section.
func redirectToFrontend(w http.ResponseWriter, r *http.Request, deps *Deps, provider, status, errMsg string) {
	u := oauthBaseURL(deps) + "/settings"
	params := url.Values{}
	params.Set("oauth_provider", provider)
	params.Set("oauth_status", status)
	params.Set("oauth_tab", "connections")
	if errMsg != "" {
		params.Set("oauth_error", errMsg)
	}
	http.Redirect(w, r, u+"?"+params.Encode(), http.StatusTemporaryRedirect)
}
