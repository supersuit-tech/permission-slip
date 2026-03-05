// BYOA (Bring Your Own OAuth App) endpoints.
//
// These endpoints let users register their own OAuth client credentials for
// any provider declared in a connector manifest. This is essential for:
//   - Providers that don't have platform-level credentials (e.g. Salesforce)
//   - Self-hosted deployments where users bring their own Google/Microsoft apps
//
// Provider resolution order: platform-level built-in → user-level BYOA config.
// BYOA credentials are merged into the existing provider config (preserving
// endpoints and scopes from built-in or manifest sources).
//
// Client ID and secret are encrypted in Supabase Vault — never stored in plaintext.
package api

import (
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/supersuit-tech/permission-slip-web/db"
	"github.com/supersuit-tech/permission-slip-web/oauth"
)

// --- Request/Response types ---

type createOAuthProviderConfigRequest struct {
	Provider     string `json:"provider" validate:"required"`
	ClientID     string `json:"client_id" validate:"required"`
	ClientSecret string `json:"client_secret" validate:"required"`
}

type oauthProviderConfigResponse struct {
	Provider  string    `json:"provider"`
	CreatedAt time.Time `json:"created_at"`
}

type oauthProviderConfigListResponse struct {
	Configs []oauthProviderConfigResponse `json:"configs"`
}

type oauthProviderConfigDeleteResponse struct {
	Provider  string    `json:"provider"`
	DeletedAt time.Time `json:"deleted_at"`
}

// --- Routes ---

// RegisterOAuthBYOARoutes adds BYOA provider management endpoints to the mux.
func RegisterOAuthBYOARoutes(mux *http.ServeMux, deps *Deps) {
	requireProfile := RequireProfile(deps)

	mux.Handle("POST /v1/oauth/provider-configs", requireProfile(handleCreateOAuthProviderConfig(deps)))
	mux.Handle("GET /v1/oauth/provider-configs", requireProfile(handleListOAuthProviderConfigs(deps)))
	mux.Handle("DELETE /v1/oauth/provider-configs/{provider}", requireProfile(handleDeleteOAuthProviderConfig(deps)))
}

// --- Handlers ---

// handleCreateOAuthProviderConfig stores user-provided OAuth client credentials
// for a provider. The credentials are encrypted in the vault and the provider
// registry is updated so subsequent OAuth flows use the BYOA credentials.
func handleCreateOAuthProviderConfig(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.Vault == nil || deps.DB == nil {
			RespondError(w, r, http.StatusServiceUnavailable, ServiceUnavailable("OAuth not available"))
			return
		}

		var req createOAuthProviderConfigRequest
		if !DecodeJSONOrReject(w, r, &req) {
			return
		}
		if !ValidateRequest(w, r, &req) {
			return
		}

		if !validProviderID(req.Provider) {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "invalid provider ID"))
			return
		}

		// Verify the provider exists in the registry (must be declared by a
		// built-in config or a connector manifest before BYOA can supply creds).
		if deps.OAuthProviders != nil {
			if _, ok := deps.OAuthProviders.Get(req.Provider); !ok {
				RespondError(w, r, http.StatusNotFound, NotFound(ErrOAuthProviderNotFound,
					"OAuth provider not found. Provider must be declared by a built-in config or connector manifest before configuring BYOA credentials."))
				return
			}
		}

		profile := Profile(r.Context())

		tx, owned, err := db.BeginOrContinue(r.Context(), deps.DB)
		if err != nil {
			log.Printf("[%s] CreateOAuthProviderConfig: begin tx: %v", TraceID(r.Context()), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to save OAuth provider config"))
			return
		}
		if owned {
			defer db.RollbackTx(r.Context(), tx) //nolint:errcheck // best-effort cleanup
		}

		// Store client ID and secret in the vault.
		configID, err := generatePrefixedID("opc_", 16)
		if err != nil {
			log.Printf("[%s] CreateOAuthProviderConfig: generate ID: %v", TraceID(r.Context()), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to save OAuth provider config"))
			return
		}

		clientIDVaultID, err := deps.Vault.CreateSecret(r.Context(), tx, configID+"_client_id", []byte(req.ClientID))
		if err != nil {
			log.Printf("[%s] CreateOAuthProviderConfig: vault create client_id: %v", TraceID(r.Context()), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to save OAuth provider config"))
			return
		}

		clientSecretVaultID, err := deps.Vault.CreateSecret(r.Context(), tx, configID+"_client_secret", []byte(req.ClientSecret))
		if err != nil {
			log.Printf("[%s] CreateOAuthProviderConfig: vault create client_secret: %v", TraceID(r.Context()), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to save OAuth provider config"))
			return
		}

		config, err := db.CreateOAuthProviderConfig(r.Context(), tx, db.CreateOAuthProviderConfigParams{
			ID:                  configID,
			UserID:              profile.ID,
			Provider:            req.Provider,
			ClientIDVaultID:     clientIDVaultID,
			ClientSecretVaultID: clientSecretVaultID,
		})
		if err != nil {
			var configErr *db.OAuthProviderConfigError
			if errors.As(err, &configErr) && configErr.Code == db.OAuthProviderConfigErrDuplicate {
				RespondError(w, r, http.StatusConflict, Conflict(ErrOAuthProviderConfigExists,
					"OAuth provider config already exists for this provider. Delete the existing config first."))
				return
			}
			log.Printf("[%s] CreateOAuthProviderConfig: db create: %v", TraceID(r.Context()), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to save OAuth provider config"))
			return
		}

		if owned {
			if err := db.CommitTx(r.Context(), tx); err != nil {
				log.Printf("[%s] CreateOAuthProviderConfig: commit: %v", TraceID(r.Context()), err)
				RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to save OAuth provider config"))
				return
			}
		}

		// Update the in-memory registry so subsequent OAuth flows pick up the
		// new BYOA credentials immediately.
		if deps.OAuthProviders != nil {
			if err := deps.OAuthProviders.Register(oauth.Provider{
				ID:           req.Provider,
				ClientID:     req.ClientID,
				ClientSecret: req.ClientSecret,
				Source:       oauth.SourceBYOA,
			}); err != nil {
				log.Printf("[%s] CreateOAuthProviderConfig: registry update: %v", TraceID(r.Context()), err)
				// Non-fatal — DB is the source of truth. Registry will be
				// consistent after a restart.
			}
		}

		RespondJSON(w, http.StatusCreated, oauthProviderConfigResponse{
			Provider:  config.Provider,
			CreatedAt: config.CreatedAt,
		})
	}
}

// handleListOAuthProviderConfigs returns all BYOA provider configs for the
// authenticated user. Client secrets are never included in the response.
func handleListOAuthProviderConfigs(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.DB == nil {
			RespondError(w, r, http.StatusServiceUnavailable, ServiceUnavailable("Database not available"))
			return
		}

		profile := Profile(r.Context())
		configs, err := db.ListOAuthProviderConfigsByUser(r.Context(), deps.DB, profile.ID)
		if err != nil {
			log.Printf("[%s] ListOAuthProviderConfigs: %v", TraceID(r.Context()), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to list OAuth provider configs"))
			return
		}

		data := make([]oauthProviderConfigResponse, len(configs))
		for i, c := range configs {
			data[i] = oauthProviderConfigResponse{
				Provider:  c.Provider,
				CreatedAt: c.CreatedAt,
			}
		}

		RespondJSON(w, http.StatusOK, oauthProviderConfigListResponse{Configs: data})
	}
}

// handleDeleteOAuthProviderConfig removes a user's BYOA provider config and
// its encrypted vault secrets. After deletion, the provider reverts to its
// built-in or manifest configuration (if any).
func handleDeleteOAuthProviderConfig(deps *Deps) http.HandlerFunc {
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
			log.Printf("[%s] DeleteOAuthProviderConfig: begin tx: %v", TraceID(r.Context()), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to delete OAuth provider config"))
			return
		}
		if owned {
			defer db.RollbackTx(r.Context(), tx) //nolint:errcheck // best-effort cleanup
		}

		result, err := db.DeleteOAuthProviderConfig(r.Context(), tx, profile.ID, providerID)
		if err != nil {
			var configErr *db.OAuthProviderConfigError
			if errors.As(err, &configErr) && configErr.Code == db.OAuthProviderConfigErrNotFound {
				RespondError(w, r, http.StatusNotFound, NotFound(ErrOAuthProviderConfigNotFound, "OAuth provider config not found"))
				return
			}
			log.Printf("[%s] DeleteOAuthProviderConfig: %v", TraceID(r.Context()), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to delete OAuth provider config"))
			return
		}

		// Delete vault secrets (best-effort — idempotent).
		if err := deps.Vault.DeleteSecret(r.Context(), tx, result.ClientIDVaultID); err != nil {
			log.Printf("[%s] DeleteOAuthProviderConfig: vault delete client_id: %v", TraceID(r.Context()), err)
		}
		if err := deps.Vault.DeleteSecret(r.Context(), tx, result.ClientSecretVaultID); err != nil {
			log.Printf("[%s] DeleteOAuthProviderConfig: vault delete client_secret: %v", TraceID(r.Context()), err)
		}

		if owned {
			if err := db.CommitTx(r.Context(), tx); err != nil {
				log.Printf("[%s] DeleteOAuthProviderConfig: commit: %v", TraceID(r.Context()), err)
				RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to delete OAuth provider config"))
				return
			}
		}

		// Revert the provider in the registry to its non-BYOA configuration.
		// The simplest approach is to remove the BYOA source and re-register
		// the base provider if it exists. For now, we remove the BYOA override
		// by re-registering the base provider from built-in/manifest sources.
		// This is safe because the registry's priority system means the base
		// provider will be restored correctly.
		revertProviderAfterBYOADelete(deps, providerID)

		RespondJSON(w, http.StatusOK, oauthProviderConfigDeleteResponse{
			Provider:  providerID,
			DeletedAt: time.Now().UTC(),
		})
	}
}

// revertProviderAfterBYOADelete removes the BYOA source from a provider in
// the in-memory registry. Since the registry doesn't track layered sources,
// we remove the provider and re-register from built-in defaults if applicable.
func revertProviderAfterBYOADelete(deps *Deps, providerID string) {
	if deps.OAuthProviders == nil {
		return
	}

	// Get the current provider to check if it's BYOA.
	current, ok := deps.OAuthProviders.Get(providerID)
	if !ok || current.Source != oauth.SourceBYOA {
		return // Not BYOA or not found — nothing to revert.
	}

	// Remove the BYOA-enhanced provider.
	_ = deps.OAuthProviders.Remove(providerID)

	// Re-register from built-in config if available.
	builtIn := oauth.BuiltInProviders()
	for _, bp := range builtIn {
		if bp.ID == providerID {
			_ = deps.OAuthProviders.Register(bp)
			return
		}
	}

	// If not built-in, re-register the provider without credentials
	// (preserving the endpoint/scope config from the former BYOA entry).
	_ = deps.OAuthProviders.Register(oauth.Provider{
		ID:           current.ID,
		AuthorizeURL: current.AuthorizeURL,
		TokenURL:     current.TokenURL,
		Scopes:       current.Scopes,
		Source:       oauth.SourceManifest,
	})
}
