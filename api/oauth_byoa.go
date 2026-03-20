// BYOA (Bring Your Own OAuth App) endpoints.
//
// These endpoints let self-hosted users register their own OAuth client
// credentials for any provider declared in a connector manifest or built-in
// config. This is useful when a provider has no platform-level credentials —
// the manifest declares endpoints and scopes but no client ID/secret (e.g.
// Salesforce). Users create an OAuth app in the provider's developer console,
// then enter the client ID and secret here.
//
// BYOA is DISABLED on the hosted platform (app.permissionslip.dev) where
// BillingEnabled is true. All providers ship with platform-level credentials
// on the hosted platform. On self-hosted deployments, users can either set
// provider credentials via environment variables or use BYOA.
//
// Provider resolution order: platform-level built-in → user-level BYOA config.
// BYOA credentials are merged into the existing provider config (preserving
// endpoints and scopes from built-in or manifest sources).
//
// Multi-tenancy note: BYOA credentials are stored per-user in the database,
// but the in-memory provider registry is global (shared across all users).
// When a user registers BYOA credentials, those credentials become the active
// provider config for ALL users' OAuth flows on this server instance. This is
// acceptable for single-tenant and self-hosted deployments. Multi-tenant
// deployments should use platform-level built-in credentials (env vars) instead,
// reserving BYOA for admin-level provider configuration.
//
// Client ID and secret are encrypted in Supabase Vault — never stored in plaintext.
package api

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/supersuit-tech/permission-slip-web/db"
	"github.com/supersuit-tech/permission-slip-web/oauth"
)

// oauthClientCredentialMaxLen is the maximum length for OAuth client ID and
// client secret values. OAuth providers typically use credentials well under
// this limit. The cap prevents storage abuse via the vault.
const oauthClientCredentialMaxLen = 2048

// --- Request/Response types ---

type createOAuthProviderConfigRequest struct {
	Provider     string `json:"provider" validate:"required"`
	ClientID     string `json:"client_id" validate:"required"`
	ClientSecret string `json:"client_secret" validate:"required"`
}

type updateOAuthProviderConfigRequest struct {
	ClientID     string `json:"client_id" validate:"required"`
	ClientSecret string `json:"client_secret" validate:"required"`
}

type oauthProviderConfigResponse struct {
	Provider  string    `json:"provider"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type oauthProviderConfigListResponse struct {
	Configs []oauthProviderConfigResponse `json:"configs"`
}

type oauthProviderConfigDeleteResponse struct {
	Provider  string    `json:"provider"`
	DeletedAt time.Time `json:"deleted_at"`
}

// --- Routes ---

func init() {
	RegisterRouteGroup(RegisterOAuthBYOARoutes)
}

// RegisterOAuthBYOARoutes adds BYOA provider management endpoints to the mux.
// BYOA is only available on self-hosted deployments (BillingEnabled == false).
// On the hosted platform (app.permissionslip.dev), all providers ship with
// platform-level credentials so BYOA is unnecessary.
func RegisterOAuthBYOARoutes(mux *http.ServeMux, deps *Deps) {
	if deps.BillingEnabled {
		// Hosted deployment — BYOA is disabled. Register handlers that
		// return 404 so the endpoints don't leak into the hosted API.
		byoaDisabled := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			RespondError(w, r, http.StatusNotFound, NotFound(ErrConfigurationDisabled, "BYOA is not available on hosted deployments"))
		})
		mux.Handle("POST /oauth/provider-configs", byoaDisabled)
		mux.Handle("GET /oauth/provider-configs", byoaDisabled)
		mux.Handle("PUT /oauth/provider-configs/{provider}", byoaDisabled)
		mux.Handle("DELETE /oauth/provider-configs/{provider}", byoaDisabled)
		return
	}

	requireProfile := RequireProfile(deps)

	mux.Handle("POST /oauth/provider-configs", requireProfile(handleCreateOAuthProviderConfig(deps)))
	mux.Handle("GET /oauth/provider-configs", requireProfile(handleListOAuthProviderConfigs(deps)))
	mux.Handle("PUT /oauth/provider-configs/{provider}", requireProfile(handleUpdateOAuthProviderConfig(deps)))
	mux.Handle("DELETE /oauth/provider-configs/{provider}", requireProfile(handleDeleteOAuthProviderConfig(deps)))
}

// --- Helpers ---

// validateClientCredentialLengths checks that client_id and client_secret do
// not exceed the maximum allowed length. Returns true if valid.
func validateClientCredentialLengths(w http.ResponseWriter, r *http.Request, clientID, clientSecret string) bool {
	if len(clientID) > oauthClientCredentialMaxLen {
		RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "client_id exceeds maximum length"))
		return false
	}
	if len(clientSecret) > oauthClientCredentialMaxLen {
		RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "client_secret exceeds maximum length"))
		return false
	}
	return true
}

// storeClientCredentials encrypts client_id and client_secret in the vault
// and returns the vault secret IDs. Used by both create and update handlers.
//
// If the client_secret vault write fails after client_id succeeded, the
// client_id secret is cleaned up to prevent orphaned vault entries.
func storeClientCredentials(ctx context.Context, deps *Deps, tx db.DBTX, namePrefix, clientID, clientSecret string) (clientIDVaultID, clientSecretVaultID string, err error) {
	clientIDVaultID, err = deps.Vault.CreateSecret(ctx, tx, namePrefix+"_client_id", []byte(clientID))
	if err != nil {
		return "", "", err
	}
	clientSecretVaultID, err = deps.Vault.CreateSecret(ctx, tx, namePrefix+"_client_secret", []byte(clientSecret))
	if err != nil {
		// Clean up the client_id secret to avoid orphaned vault entries.
		_ = deps.Vault.DeleteSecret(ctx, tx, clientIDVaultID)
		return "", "", err
	}
	return clientIDVaultID, clientSecretVaultID, nil
}

// updateOAuthRegistry merges BYOA credentials into the in-memory provider
// registry. Non-fatal — DB is the source of truth.
func updateOAuthRegistry(ctx context.Context, deps *Deps, providerID, clientID, clientSecret string) {
	if deps.OAuthProviders == nil {
		return
	}
	if err := deps.OAuthProviders.Register(oauth.Provider{
		ID:           providerID,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Source:       oauth.SourceBYOA,
	}); err != nil {
		log.Printf("BYOA registry update for %q: %v", providerID, err)
		CaptureError(ctx, err)
	}
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
		if !validateClientCredentialLengths(w, r, req.ClientID, req.ClientSecret) {
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
			CaptureError(r.Context(), err)
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
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to save OAuth provider config"))
			return
		}

		clientIDVaultID, clientSecretVaultID, err := storeClientCredentials(r.Context(), deps, tx, configID, req.ClientID, req.ClientSecret)
		if err != nil {
			log.Printf("[%s] CreateOAuthProviderConfig: vault store: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
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
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to save OAuth provider config"))
			return
		}

		if owned {
			if err := db.CommitTx(r.Context(), tx); err != nil {
				log.Printf("[%s] CreateOAuthProviderConfig: commit: %v", TraceID(r.Context()), err)
				CaptureError(r.Context(), err)
				RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to save OAuth provider config"))
				return
			}
		}

		// Update the in-memory registry so subsequent OAuth flows pick up the
		// new BYOA credentials immediately. Non-fatal — DB is the source of truth.
		updateOAuthRegistry(r.Context(), deps, req.Provider, req.ClientID, req.ClientSecret)

		RespondJSON(w, http.StatusCreated, oauthProviderConfigResponse{
			Provider:  config.Provider,
			CreatedAt: config.CreatedAt,
			UpdatedAt: config.UpdatedAt,
		})
	}
}

// handleUpdateOAuthProviderConfig replaces the client credentials for an
// existing BYOA provider config. This supports credential rotation without
// requiring delete + create.
func handleUpdateOAuthProviderConfig(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.Vault == nil || deps.DB == nil {
			RespondError(w, r, http.StatusServiceUnavailable, ServiceUnavailable("OAuth not available"))
			return
		}

		providerID := r.PathValue("provider")
		if !validProviderID(providerID) {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "invalid provider ID"))
			return
		}

		var req updateOAuthProviderConfigRequest
		if !DecodeJSONOrReject(w, r, &req) {
			return
		}
		if !ValidateRequest(w, r, &req) {
			return
		}
		if !validateClientCredentialLengths(w, r, req.ClientID, req.ClientSecret) {
			return
		}

		profile := Profile(r.Context())

		tx, owned, err := db.BeginOrContinue(r.Context(), deps.DB)
		if err != nil {
			log.Printf("[%s] UpdateOAuthProviderConfig: begin tx: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to update OAuth provider config"))
			return
		}
		if owned {
			defer db.RollbackTx(r.Context(), tx) //nolint:errcheck // best-effort cleanup
		}

		// Look up existing config.
		existing, err := db.GetOAuthProviderConfig(r.Context(), tx, profile.ID, providerID)
		if err != nil {
			log.Printf("[%s] UpdateOAuthProviderConfig: lookup: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to update OAuth provider config"))
			return
		}
		if existing == nil {
			RespondError(w, r, http.StatusNotFound, NotFound(ErrOAuthProviderConfigNotFound,
				"OAuth provider config not found. Create one first with POST."))
			return
		}

		// Delete old vault secrets (best-effort).
		_ = deps.Vault.DeleteSecret(r.Context(), tx, existing.ClientIDVaultID)
		_ = deps.Vault.DeleteSecret(r.Context(), tx, existing.ClientSecretVaultID)

		// Store new secrets.
		newClientIDVaultID, newClientSecretVaultID, err := storeClientCredentials(r.Context(), deps, tx, existing.ID, req.ClientID, req.ClientSecret)
		if err != nil {
			log.Printf("[%s] UpdateOAuthProviderConfig: vault store: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to update OAuth provider config"))
			return
		}

		// Update the DB row with new vault IDs.
		config, err := db.UpdateOAuthProviderConfig(r.Context(), tx, profile.ID, providerID, newClientIDVaultID, newClientSecretVaultID)
		if err != nil {
			log.Printf("[%s] UpdateOAuthProviderConfig: db update: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to update OAuth provider config"))
			return
		}

		if owned {
			if err := db.CommitTx(r.Context(), tx); err != nil {
				log.Printf("[%s] UpdateOAuthProviderConfig: commit: %v", TraceID(r.Context()), err)
				CaptureError(r.Context(), err)
				RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to update OAuth provider config"))
				return
			}
		}

		// Update the in-memory registry. Non-fatal — DB is the source of truth.
		updateOAuthRegistry(r.Context(), deps, providerID, req.ClientID, req.ClientSecret)

		RespondJSON(w, http.StatusOK, oauthProviderConfigResponse{
			Provider:  config.Provider,
			CreatedAt: config.CreatedAt,
			UpdatedAt: config.UpdatedAt,
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
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to list OAuth provider configs"))
			return
		}

		data := make([]oauthProviderConfigResponse, len(configs))
		for i, c := range configs {
			data[i] = oauthProviderConfigResponse{
				Provider:  c.Provider,
				CreatedAt: c.CreatedAt,
				UpdatedAt: c.UpdatedAt,
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
			CaptureError(r.Context(), err)
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
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to delete OAuth provider config"))
			return
		}

		// Delete vault secrets (best-effort — idempotent).
		if err := deps.Vault.DeleteSecret(r.Context(), tx, result.ClientIDVaultID); err != nil {
			log.Printf("[%s] DeleteOAuthProviderConfig: vault delete client_id: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
		}
		if err := deps.Vault.DeleteSecret(r.Context(), tx, result.ClientSecretVaultID); err != nil {
			log.Printf("[%s] DeleteOAuthProviderConfig: vault delete client_secret: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
		}

		if owned {
			if err := db.CommitTx(r.Context(), tx); err != nil {
				log.Printf("[%s] DeleteOAuthProviderConfig: commit: %v", TraceID(r.Context()), err)
				CaptureError(r.Context(), err)
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
