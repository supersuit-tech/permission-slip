package api

import "net/http"

// configResponse exposes server-level feature flags to the frontend.
// Add new fields here as more server config needs to be exposed
// (e.g., feature flags for upcoming features).
type configResponse struct {
	GoogleOAuthConfigured    bool `json:"google_oauth_configured"`
	MicrosoftOAuthConfigured bool `json:"microsoft_oauth_configured"`
}

func init() {
	RegisterRouteGroup(RegisterConfigRoutes)
}

// RegisterConfigRoutes adds server configuration endpoints to the mux.
func RegisterConfigRoutes(mux *http.ServeMux, deps *Deps) {
	requireSession := RequireSession(deps)
	mux.Handle("GET /config", requireSession(handleGetConfig(deps)))
}

// handleGetConfig returns server-level feature flags that the frontend needs
// to adapt its UI.
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
			GoogleOAuthConfigured:    googleConfigured,
			MicrosoftOAuthConfigured: microsoftConfigured,
		})
	}
}
