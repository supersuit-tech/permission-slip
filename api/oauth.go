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
//   - Callback derives user identity from the signed state token (no session required)
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

// subdomainPattern matches valid RFC 1123-style subdomains: lowercase
// alphanumeric and hyphens, not starting or ending with a hyphen, max 63 chars.
// Shared by all per-instance OAuth providers (Shopify, Zendesk, etc.) to
// prevent URL injection or SSRF when substituting into provider URL templates.
var subdomainPattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)

// shopSubdomainPattern uses the shared RFC 1123 label rules to validate the
// "shop" query parameter before substituting it into Shopify OAuth URLs.
var shopSubdomainPattern = subdomainPattern

// zendeskSubdomainPattern uses the shared RFC 1123 label rules to validate the
// "subdomain" query parameter before substituting it into Zendesk OAuth URLs.
var zendeskSubdomainPattern = subdomainPattern

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
	ID          string    `json:"id"`
	Provider    string    `json:"provider"`
	Scopes      []string  `json:"scopes"`
	Status      string    `json:"status"`
	ConnectedAt time.Time `json:"connected_at"`
	// Instance is the per-instance identifier for providers with subdomain-based
	// OAuth URLs (e.g. "mycompany.zendesk.com" for Zendesk, "mystore.myshopify.com"
	// for Shopify). Empty for providers with static OAuth endpoints.
	Instance string `json:"instance,omitempty"`
	// DisplayName is a human-readable identifier for this connection, typically
	// the authenticated user's email address. Empty when email is not available.
	DisplayName string `json:"display_name,omitempty"`
}

type oauthConnectionListResponse struct {
	Connections []oauthConnectionResponse `json:"connections"`
}

type oauthDisconnectResponse struct {
	Provider       string    `json:"provider"`
	DisconnectedAt time.Time `json:"disconnected_at"`
}

// --- Routes ---

func init() {
	RegisterRouteGroup(RegisterOAuthRoutes)
}

// RegisterOAuthRoutes adds OAuth-related endpoints to the mux.
func RegisterOAuthRoutes(mux *http.ServeMux, deps *Deps) {
	requireProfile := RequireProfile(deps)

	mux.Handle("GET /oauth/providers", requireProfile(handleListOAuthProviders(deps)))
	mux.Handle("GET /oauth/{provider}/authorize", AllowQueryParamToken(requireProfile(handleOAuthAuthorize(deps))))
	mux.Handle("GET /oauth/{provider}/callback", handleOAuthCallback(deps))
	mux.Handle("GET /oauth/connections", requireProfile(handleListOAuthConnections(deps)))
	mux.Handle("DELETE /oauth/connections/{connection_id}", requireProfile(handleDeleteOAuthConnection(deps)))
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
// OAuth provider, requested scopes, and optional per-instance identifier.
// This prevents CSRF and state-fixation attacks on the callback endpoint
// and carries the final scope list so the callback can store exactly what
// was requested (including any extra scopes from the authorize query params).
//
// The shop parameter carries the per-instance identifier for providers with
// dynamic OAuth URLs — the Shopify store subdomain (e.g. "mystore") or the
// Zendesk subdomain (e.g. "mycompany"). Pass "" for providers with static
// OAuth endpoints (e.g. Google, Slack).
//
// The replaceID parameter, if non-empty, is the ID of an existing OAuth
// connection to replace. When set, storeOAuthTokens deletes only that
// specific connection instead of all connections for the provider. When
// empty, a new connection is created alongside any existing ones.
//
// The returnTo parameter is the frontend path the user should be sent back
// to after the callback (e.g. "/agents/1"). Pass "" to redirect to "/".
func createOAuthState(deps *Deps, userID, provider string, scopes []string, shop, returnTo, replaceID string) (string, error) {
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
	if returnTo != "" {
		claims["return_to"] = returnTo
	}
	if replaceID != "" {
		claims["replace_id"] = replaceID
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// oauthState holds the decoded claims from a verified OAuth state token.
type oauthState struct {
	UserID   string
	Provider string
	Scopes   []string
	// Shop holds the per-instance identifier for providers with dynamic
	// OAuth URLs: the Shopify store subdomain (e.g. "mystore") or the
	// Zendesk subdomain (e.g. "mycompany"). Empty for providers with
	// static OAuth endpoints (e.g. Google, Slack). The JWT claim key is
	// "shop" for backward compatibility with in-flight tokens.
	Shop string
	// ReturnTo is the frontend path the user should be redirected to after
	// the OAuth callback completes (e.g. "/agents/1"). Empty falls back
	// to the root path "/".
	ReturnTo string
	// ReplaceID is the ID of an existing OAuth connection to replace. When
	// set, storeOAuthTokens deletes only that specific connection. When empty,
	// a new connection is created alongside any existing ones.
	ReplaceID string
}

// verifyOAuthState validates the signed state JWT and returns the encoded
// user ID, provider, requested scopes, and optional per-instance identifier
// (Shop field). Returns an error if the signature is invalid, the token is
// expired, or the required claims are missing.
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
	returnTo, _ := claims["return_to"].(string)
	replaceID, _ := claims["replace_id"].(string)
	return &oauthState{UserID: sub, Provider: prov, Scopes: scopes, Shop: shop, ReturnTo: returnTo, ReplaceID: replaceID}, nil
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

// isRelativePath returns true if s is a relative path that is safe to
// use as a redirect target (prevents open-redirect via protocol-relative
// URLs like "//evil.com" or absolute URLs like "https://evil.com").
func isRelativePath(s string) bool {
	return strings.HasPrefix(s, "/") && !strings.HasPrefix(s, "//")
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

// --- Per-subdomain URL helpers (Zendesk) ---

// providerNeedsSubdomain returns true if the provider's authorize or token URL
// contains a {subdomain} placeholder that must be resolved before use.
func providerNeedsSubdomain(p oauth.Provider) bool {
	return strings.Contains(p.AuthorizeURL, "{subdomain}") || strings.Contains(p.TokenURL, "{subdomain}")
}

// resolveSubdomainURLs replaces {subdomain} placeholders in the provider's
// authorize and token URLs with the given subdomain. The subdomain value must
// already be validated (see zendeskSubdomainPattern). Returns a copy of the
// provider with resolved URLs.
func resolveSubdomainURLs(p oauth.Provider, subdomain string) oauth.Provider {
	p.AuthorizeURL = strings.ReplaceAll(p.AuthorizeURL, "{subdomain}", subdomain)
	p.TokenURL = strings.ReplaceAll(p.TokenURL, "{subdomain}", subdomain)
	return p
}

// validateZendeskSubdomain checks that a subdomain value is a valid Zendesk
// subdomain. Accepts either a bare subdomain ("mycompany") or a full domain
// ("mycompany.zendesk.com") and returns the normalized bare subdomain.
func validateZendeskSubdomain(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimRight(raw, "/")
	raw = strings.ToLower(raw)

	// Accept full domain form.
	if strings.HasSuffix(raw, ".zendesk.com") {
		raw = strings.TrimSuffix(raw, ".zendesk.com")
	} else if strings.Contains(raw, ".") {
		return "", fmt.Errorf("subdomain must be a bare subdomain (e.g. \"mycompany\") or full domain (e.g. \"mycompany.zendesk.com\")")
	}

	if raw == "" {
		return "", fmt.Errorf("subdomain is required")
	}
	if !zendeskSubdomainPattern.MatchString(raw) {
		return "", fmt.Errorf("subdomain contains invalid characters")
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

		// Resolve per-instance URL templates (Shopify: {shop}, Zendesk: {subdomain}).
		// instanceID carries the per-instance identifier (subdomain or shop) so it
		// can be encoded in the CSRF state token for use during the callback.
		var instanceID string
		if providerNeedsShop(provider) {
			rawShop := r.URL.Query().Get("shop")
			var err error
			instanceID, err = validateShopSubdomain(rawShop)
			if err != nil {
				RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest,
					fmt.Sprintf("invalid shop parameter: %v", err)))
				return
			}
			provider = resolveShopURLs(provider, instanceID)
		} else if providerNeedsSubdomain(provider) {
			rawSubdomain := r.URL.Query().Get("subdomain")
			var err error
			instanceID, err = validateZendeskSubdomain(rawSubdomain)
			if err != nil {
				RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest,
					fmt.Sprintf("invalid subdomain parameter: %v", err)))
				return
			}
			provider = resolveSubdomainURLs(provider, instanceID)
		}

		cfg := newOAuth2Config(deps, provider)

		// Request additional scopes from query params if provided.
		if extraScopes := r.URL.Query()["scope"]; len(extraScopes) > 0 {
			cfg.Scopes = deduplicateScopes(append(cfg.Scopes, extraScopes...))
		}

		// Create the CSRF state token after computing the final scope list so
		// the callback can store exactly which scopes were requested.
		returnTo := r.URL.Query().Get("return_to")
		if returnTo != "" && !isRelativePath(returnTo) {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "invalid return_to parameter"))
			return
		}
		// Optional: replace an existing connection instead of creating alongside.
		replaceID := r.URL.Query().Get("replace")
		profile := Profile(r.Context())
		state, err := createOAuthState(deps, profile.ID, providerID, cfg.Scopes, instanceID, returnTo, replaceID)
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

		// Validate CSRF state first — even for error responses from the provider.
		// Since this endpoint has no session middleware, the signed state token is
		// the sole proof that the request originated from our authorize flow.
		// Validating it before anything else prevents unauthenticated callers from
		// triggering frontend redirects with arbitrary error text.
		stateStr := r.URL.Query().Get("state")
		if stateStr == "" {
			redirectToFrontend(w, r, deps, providerID, "error", "Missing state parameter", "", "")
			return
		}
		state, err := verifyOAuthState(deps, stateStr)
		if err != nil {
			log.Printf("[%s] OAuthCallback: invalid state: %v", TraceID(r.Context()), err)
			redirectToFrontend(w, r, deps, providerID, "error", "Invalid or expired state token", "", "")
			return
		}
		if state.Provider != providerID {
			log.Printf("[%s] OAuthCallback: provider mismatch: state=%s path=%s", TraceID(r.Context()), state.Provider, providerID)
			redirectToFrontend(w, r, deps, providerID, "error", "Provider mismatch", state.ReturnTo, "")
			return
		}

		// Check for error from provider (e.g. user denied consent).
		// This runs after state validation so only legitimate OAuth flows can
		// surface error messages to the frontend.
		if errCode := r.URL.Query().Get("error"); errCode != "" {
			errDesc := r.URL.Query().Get("error_description")
			if errDesc == "" {
				errDesc = errCode
			}
			redirectToFrontend(w, r, deps, providerID, "error", errDesc, state.ReturnTo, "")
			return
		}

		// The callback is a browser redirect from the OAuth provider, so there
		// is no Authorization header (no session middleware). The user identity
		// comes from the signed state token which was created during the
		// authorize step while the user was authenticated.
		userID := state.UserID

		// Look up profile for the user
		profile, err := db.GetProfileByUserID(r.Context(), deps.DB, userID)
		if err != nil || profile == nil {
			log.Printf("[%s] OAuthCallback: profile lookup failed: %v", TraceID(r.Context()), err)
			redirectToFrontend(w, r, deps, providerID, "error", "Profile not found", state.ReturnTo, "")
			return
		}

		code := r.URL.Query().Get("code")
		if code == "" {
			redirectToFrontend(w, r, deps, providerID, "error", "Missing authorization code", state.ReturnTo, "")
			return
		}

		provider, ok := deps.OAuthProviders.Get(providerID)
		if !ok {
			redirectToFrontend(w, r, deps, providerID, "error", "Provider not found", state.ReturnTo, "")
			return
		}

		// Resolve per-instance URL templates using the instance identifier from
		// the state token. Capture the placeholder flags before resolution so
		// we can determine the correct extra_data key below without re-checking
		// the (now-resolved) URLs.
		needsShop := providerNeedsShop(provider)
		needsSubdomain := providerNeedsSubdomain(provider)
		if state.Shop != "" {
			if needsShop {
				provider = resolveShopURLs(provider, state.Shop)
			} else if needsSubdomain {
				provider = resolveSubdomainURLs(provider, state.Shop)
			}
		}

		cfg := newOAuth2Config(deps, provider)

		// Exchange authorization code for tokens.
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()
		token, err := cfg.Exchange(ctx, code)
		if err != nil {
			log.Printf("[%s] OAuthCallback: token exchange failed: %v", TraceID(r.Context()), err)
			redirectToFrontend(w, r, deps, providerID, "error", "Token exchange failed", state.ReturnTo, "")
			return
		}

		// Store tokens in vault within a transaction. Use scopes from the
		// state token (set during authorize) so extra scopes are preserved.
		storedScopes := state.Scopes
		if len(storedScopes) == 0 {
			storedScopes = provider.Scopes // fallback for pre-existing state tokens
		}

		// Build extra data to persist alongside the tokens. For providers
		// with per-instance URLs, store an instance identifier so it's
		// available at execution time as a credential.
		var stateExtraData map[string]string
		if state.Shop != "" {
			if needsSubdomain {
				// Zendesk: store bare subdomain (e.g. "mycompany") so the
				// connector can build API URLs like mycompany.zendesk.com.
				stateExtraData = map[string]string{"subdomain": state.Shop}
			} else if needsShop {
				// Shopify: store the full shop domain (e.g. "mystore.myshopify.com").
				// NOTE: state.Shop is validated to be a bare subdomain during
				// the authorize step, so appending ".myshopify.com" is intentional.
				stateExtraData = map[string]string{"shop_domain": state.Shop + ".myshopify.com"}
			}
			// Other per-instance providers: no extra_data needed beyond the token.
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
				redirectToFrontend(w, r, deps, providerID, "error", "Could not retrieve account information — please try again", state.ReturnTo, "")
				return
			}
			if stateExtraData == nil {
				stateExtraData = make(map[string]string)
			}
			for k, v := range extra {
				stateExtraData[k] = v
			}
		}

		// Run best-effort email enrichers (fetches user email from provider
		// userinfo endpoints). Failures are logged but don't block the flow.
		if softExtra := runSoftEnrichers(ctx, providerID, token.AccessToken); softExtra != nil {
			if stateExtraData == nil {
				stateExtraData = make(map[string]string)
			}
			for k, v := range softExtra {
				stateExtraData[k] = v
			}
		}

		connID, err := storeOAuthTokens(r.Context(), deps, profile.ID, providerID, storedScopes, token, stateExtraData, state.ReplaceID)
		if err != nil {
			log.Printf("[%s] OAuthCallback: store tokens: %v", TraceID(r.Context()), err)
			redirectToFrontend(w, r, deps, providerID, "error", "Failed to store connection", state.ReturnTo, "")
			return
		}

		redirectToFrontend(w, r, deps, providerID, "success", "", state.ReturnTo, connID)
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
//
// replaceID, if non-empty, is the ID of an existing connection to replace.
// When set, only that specific connection is deleted. When empty, a new
// connection is created alongside any existing ones for the same provider.
func storeOAuthTokens(ctx context.Context, deps *Deps, userID, providerID string, scopes []string, token *oauth2.Token, stateExtra map[string]string, replaceID string) (string, error) {
	tx, owned, err := db.BeginOrContinue(ctx, deps.DB)
	if err != nil {
		return "", fmt.Errorf("begin tx: %w", err)
	}
	if owned {
		defer db.RollbackTx(ctx, tx) //nolint:errcheck // best-effort cleanup
	}

	// When replaceID is set, delete that specific connection (reconnect flow).
	// When empty, create a new connection alongside any existing ones.
	var existing *db.DeleteOAuthConnectionResult
	if replaceID != "" {
		// Validate that the connection being replaced belongs to the same
		// provider. Without this check a crafted URL could delete a
		// connection for a different provider (e.g. replace a Slack
		// connection while authorizing Google).
		oldConn, lookupErr := db.GetOAuthConnectionByID(ctx, tx, replaceID)
		if lookupErr != nil {
			return "", fmt.Errorf("lookup connection to replace: %w", lookupErr)
		}
		if oldConn != nil && oldConn.UserID == userID && oldConn.Provider == providerID {
			existing, err = db.DeleteOAuthConnectionByID(ctx, tx, userID, replaceID)
			if err != nil {
				var connErr *db.OAuthConnectionError
				if !errors.As(err, &connErr) || connErr.Code != db.OAuthConnectionErrNotFound {
					return "", fmt.Errorf("delete existing connection: %w", err)
				}
			}
		} else if oldConn != nil && oldConn.Provider != providerID {
			log.Printf("[%s] storeOAuthTokens: replaceID %q belongs to provider %q, not %q — ignoring", TraceID(ctx), replaceID, oldConn.Provider, providerID)
			// Ignore the replaceID — create a new connection instead.
		}
		// If oldConn is nil or belongs to another user, just ignore and create new.
	}

	// Track the old refresh token vault ID so we can reuse it if the new
	// token exchange doesn't include a refresh token (Google edge case where
	// prompt=consent + access_type=offline still omits the refresh token).
	var oldRefreshVaultID *string
	if existing != nil {
		oldRefreshVaultID = existing.RefreshTokenVaultID

		// Always clean up the old access token secret.
		if err := deps.Vault.DeleteSecret(ctx, tx, existing.AccessTokenVaultID); err != nil {
			return "", fmt.Errorf("delete orphaned access token secret: %w", err)
		}

		// Only delete the old refresh token secret if the new token includes
		// a replacement. Otherwise we reuse the existing vault entry.
		if existing.RefreshTokenVaultID != nil && token.RefreshToken != "" {
			if err := deps.Vault.DeleteSecret(ctx, tx, *existing.RefreshTokenVaultID); err != nil {
				return "", fmt.Errorf("delete orphaned refresh token secret: %w", err)
			}
			oldRefreshVaultID = nil // cleared — will create fresh below
		}
	}

	connID, err := generatePrefixedID("oconn_", 16)
	if err != nil {
		return "", fmt.Errorf("generate connection ID: %w", err)
	}

	accessVaultID, err := deps.Vault.CreateSecret(ctx, tx, connID+"_access", []byte(token.AccessToken))
	if err != nil {
		return "", fmt.Errorf("vault create access token: %w", err)
	}

	var refreshVaultID *string
	if token.RefreshToken != "" {
		id, err := deps.Vault.CreateSecret(ctx, tx, connID+"_refresh", []byte(token.RefreshToken))
		if err != nil {
			return "", fmt.Errorf("vault create refresh token: %w", err)
		}
		refreshVaultID = &id
	} else if oldRefreshVaultID != nil {
		// New token omitted the refresh token — reuse the old vault secret.
		log.Printf("[%s] WARNING: OAuth token exchange for provider %q did not return a refresh token; preserving existing refresh token from previous authorization", TraceID(ctx), providerID)
		refreshVaultID = oldRefreshVaultID
	} else if existing != nil {
		log.Printf("[%s] WARNING: OAuth token exchange for provider %q did not return a refresh token and no previous refresh token exists", TraceID(ctx), providerID)
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
		return "", fmt.Errorf("create connection: %w", err)
	}

	if owned {
		if err := db.CommitTx(ctx, tx); err != nil {
			return "", fmt.Errorf("commit: %w", err)
		}
	}
	return connID, nil
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

// docuSignHTTPClient is the shared HTTP client used by fetchDocuSignUserInfo.
// A package-level client is used to enable TCP connection reuse via the
// default transport's connection pool, reducing latency on the userinfo call.
var docuSignHTTPClient = &http.Client{Timeout: 10 * time.Second}

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

	resp, err := docuSignHTTPClient.Do(req)
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

// instanceFromExtraData extracts a human-readable instance identifier from an
// OAuth connection's extra_data JSON. For Zendesk connections it returns
// "{subdomain}.zendesk.com"; for Shopify it returns the raw shop_domain (which
// already includes ".myshopify.com"). Returns "" for providers with no
// per-instance data (e.g. Google, Slack).
func instanceFromExtraData(extraData json.RawMessage) string {
	if len(extraData) == 0 {
		return ""
	}
	var extra map[string]string
	if err := json.Unmarshal(extraData, &extra); err != nil {
		return ""
	}
	if subdomain, ok := extra["subdomain"]; ok && subdomain != "" {
		return subdomain + ".zendesk.com"
	}
	if shopDomain, ok := extra["shop_domain"]; ok && shopDomain != "" {
		return shopDomain
	}
	return ""
}

// displayNameFromExtraData extracts a human-readable display name from an
// OAuth connection's extra_data JSON. Prefers "display_name" (e.g. GitHub
// login or Microsoft displayName), falls back to "email". Returns "" if
// neither is present.
func displayNameFromExtraData(extraData json.RawMessage) string {
	if len(extraData) == 0 {
		return ""
	}
	var extra map[string]string
	if err := json.Unmarshal(extraData, &extra); err != nil {
		return ""
	}
	if dn := extra["display_name"]; dn != "" {
		return dn
	}
	return extra["email"]
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
				ID:          c.ID,
				Provider:    c.Provider,
				Scopes:      c.Scopes,
				Status:      c.Status,
				ConnectedAt: c.CreatedAt,
				Instance:    instanceFromExtraData(c.ExtraData),
				DisplayName: displayNameFromExtraData(c.ExtraData),
			}
		}

		RespondJSON(w, http.StatusOK, oauthConnectionListResponse{Connections: data})
	}
}

// handleDeleteOAuthConnection disconnects a specific OAuth connection by its ID.
func handleDeleteOAuthConnection(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.Vault == nil || deps.DB == nil {
			RespondError(w, r, http.StatusServiceUnavailable, ServiceUnavailable("OAuth not available"))
			return
		}

		profile := Profile(r.Context())
		connectionID := r.PathValue("connection_id")
		if connectionID == "" {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "missing connection ID"))
			return
		}

		// Look up the connection first so we can return the provider name.
		conn, err := db.GetOAuthConnectionByID(r.Context(), deps.DB, connectionID)
		if err != nil {
			log.Printf("[%s] DeleteOAuthConnection: lookup: %v", TraceID(r.Context()), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to disconnect OAuth connection"))
			return
		}
		if conn == nil || conn.UserID != profile.ID {
			RespondError(w, r, http.StatusNotFound, NotFound(ErrOAuthConnectionNotFound, "OAuth connection not found"))
			return
		}

		tx, owned, err := db.BeginOrContinue(r.Context(), deps.DB)
		if err != nil {
			log.Printf("[%s] DeleteOAuthConnection: begin tx: %v", TraceID(r.Context()), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to disconnect OAuth connection"))
			return
		}
		if owned {
			defer db.RollbackTx(r.Context(), tx) //nolint:errcheck // best-effort cleanup
		}

		result, err := db.DeleteOAuthConnectionByID(r.Context(), tx, profile.ID, connectionID)
		if err != nil {
			var connErr *db.OAuthConnectionError
			if errors.As(err, &connErr) && connErr.Code == db.OAuthConnectionErrNotFound {
				RespondError(w, r, http.StatusNotFound, NotFound(ErrOAuthConnectionNotFound, "OAuth connection not found"))
				return
			}
			log.Printf("[%s] DeleteOAuthConnection: %v", TraceID(r.Context()), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to disconnect OAuth connection"))
			return
		}

		// Delete vault secrets within the transaction. A failure here aborts
		// the PostgreSQL transaction, so we must return immediately on error
		// (matching the pattern in handleDeleteCredential).
		if err := deps.Vault.DeleteSecret(r.Context(), tx, result.AccessTokenVaultID); err != nil {
			log.Printf("[%s] DeleteOAuthConnection: vault delete access: %v", TraceID(r.Context()), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to disconnect OAuth connection"))
			return
		}
		if result.RefreshTokenVaultID != nil {
			if err := deps.Vault.DeleteSecret(r.Context(), tx, *result.RefreshTokenVaultID); err != nil {
				log.Printf("[%s] DeleteOAuthConnection: vault delete refresh: %v", TraceID(r.Context()), err)
				RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to disconnect OAuth connection"))
				return
			}
		}

		if owned {
			if err := db.CommitTx(r.Context(), tx); err != nil {
				log.Printf("[%s] DeleteOAuthConnection: commit: %v", TraceID(r.Context()), err)
				RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to disconnect OAuth connection"))
				return
			}
		}

		RespondJSON(w, http.StatusOK, oauthDisconnectResponse{
			Provider:       conn.Provider,
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

// redirectToFrontend redirects the user's browser back to the page they
// started the OAuth flow from (returnTo), or to "/" if no return path was
// provided. Status and error info are passed as query parameters so the
// frontend can display an appropriate toast.
func redirectToFrontend(w http.ResponseWriter, r *http.Request, deps *Deps, provider, status, errMsg, returnTo, connectionID string) {
	path := "/"
	if returnTo != "" && isRelativePath(returnTo) {
		path = returnTo
	}
	parsedURL, err := url.Parse(oauthBaseURL(deps) + path)
	if err != nil {
		parsedURL, _ = url.Parse(oauthBaseURL(deps) + "/")
	}
	q := parsedURL.Query()
	q.Set("oauth_provider", provider)
	q.Set("oauth_status", status)
	if errMsg != "" {
		q.Set("oauth_error", errMsg)
	}
	if connectionID != "" {
		q.Set("oauth_connection_id", connectionID)
	}
	parsedURL.RawQuery = q.Encode()
	http.Redirect(w, r, parsedURL.String(), http.StatusTemporaryRedirect)
}
