package api

import "net/http"

// configResponse exposes server-level feature flags to the frontend.
// Add new fields here as more server config needs to be exposed
// (e.g., feature flags for upcoming features).
type configResponse struct {
	BillingEnabled       bool `json:"billing_enabled"`
	StripeTestMode       bool `json:"stripe_test_mode"`
	GoogleOAuthConfigured    bool `json:"google_oauth_configured"`
	MicrosoftOAuthConfigured bool `json:"microsoft_oauth_configured"`
}

// RegisterConfigRoutes adds server configuration endpoints to the mux.
func RegisterConfigRoutes(mux *http.ServeMux, deps *Deps) {
	requireSession := RequireSession(deps)
	mux.Handle("GET /v1/config", requireSession(handleGetConfig(deps)))
}

// handleGetConfig returns server-level feature flags that the frontend needs
// to adapt its UI (e.g., showing or hiding billing-related pages).
func handleGetConfig(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var googleConfigured, microsoftConfigured bool
		if deps.OAuthProviders != nil {
			if p, ok := deps.OAuthProviders.Get("google"); ok {
				googleConfigured = p.HasClientCredentials()
			}
			if p, ok := deps.OAuthProviders.Get("microsoft"); ok {
				microsoftConfigured = p.HasClientCredentials()
			}
		}
		RespondJSON(w, http.StatusOK, configResponse{
			BillingEnabled:           deps.BillingEnabled,
			StripeTestMode:           deps.StripeTestMode,
			GoogleOAuthConfigured:    googleConfigured,
			MicrosoftOAuthConfigured: microsoftConfigured,
		})
	}
}
