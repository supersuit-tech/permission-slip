package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/supersuit-tech/permission-slip-web/connectors"
	"github.com/supersuit-tech/permission-slip-web/db"
	"github.com/supersuit-tech/permission-slip-web/oauth"
)

// paymentParams holds optional payment-related fields from the execute request.
// When nil, no payment method processing is performed.
type paymentParams struct {
	PaymentMethodID string
	AmountCents     *int
}

// executeConnectorAction looks up the action in the connector registry,
// fetches and decrypts the user's credentials, and executes the action.
// Returns (nil, nil) if no connector is registered for the action type
// (graceful degradation during the transition period).
//
// When the action declares requires_payment_method, the caller must provide
// a paymentParams with a valid payment_method_id and amount_cents. The function
// validates the payment method and enforces spending limits before execution,
// then records the transaction only after the connector succeeds. This ensures
// failed executions don't count against the user's spending limits.
func executeConnectorAction(ctx context.Context, deps *Deps, agentID int64, userID, actionType string, parameters json.RawMessage, pp *paymentParams) (*connectors.ActionResult, error) {
	if deps.Connectors == nil {
		return nil, nil
	}

	action, ok := deps.Connectors.GetAction(actionType)
	if !ok {
		return nil, nil
	}

	// Look up all required credentials for this action's connector.
	// A connector may support multiple auth methods (e.g. both oauth2 and api_key).
	// We try OAuth first; if the user has no OAuth connection, fall back to static credentials.
	reqCreds, err := db.GetRequiredCredentialsByActionType(ctx, deps.DB, actionType)
	if err != nil {
		return nil, fmt.Errorf("look up required credentials: %w", err)
	}

	connectorID := strings.SplitN(actionType, ".", 2)[0]

	var creds connectors.Credentials
	creds, err = resolveCredentialsWithFallback(ctx, deps, agentID, userID, actionType, connectorID, reqCreds)
	if err != nil {
		return nil, err
	}

	// Validate credentials before executing the action.
	if conn, ok := deps.Connectors.Get(connectorID); ok {
		if err := conn.ValidateCredentials(ctx, creds); err != nil {
			return nil, err
		}
	}

	// ── Payment method validation ─────────────────────────────────
	// Validate the payment method and enforce spending limits before execution.
	// The transaction is recorded after execution succeeds (see below).
	var paymentInfo *connectors.PaymentInfo
	var resolvedPM *resolvedPaymentMethod
	requiresPayment, err := db.GetActionRequiresPaymentMethod(ctx, deps.DB, actionType)
	if err != nil {
		return nil, fmt.Errorf("check requires_payment_method: %w", err)
	}

	if requiresPayment {
		resolvedPM, err = validatePaymentMethod(ctx, deps, userID, pp)
		if err != nil {
			return nil, err
		}
		paymentInfo = &connectors.PaymentInfo{
			StripePaymentMethodID: resolvedPM.stripePaymentMethodID,
			Brand:                 resolvedPM.brand,
			Last4:                 resolvedPM.last4,
			AmountCents:           resolvedPM.amount,
		}
	}

	result, err := action.Execute(ctx, connectors.ActionRequest{
		ActionType:  actionType,
		Parameters:  parameters,
		Credentials: creds,
		Payment:     paymentInfo,
	})
	if err != nil {
		return nil, err
	}

	// ── Record payment transaction (only on success) ──────────────
	// The transaction is recorded after the connector succeeds to avoid
	// counting failed executions against the user's spending limits.
	if resolvedPM != nil {
		_, txErr := db.CreatePaymentMethodTransaction(ctx, deps.DB, &db.PaymentMethodTransaction{
			PaymentMethodID: resolvedPM.paymentMethodID,
			UserID:          userID,
			ConnectorID:     connectorID,
			ActionType:      actionType,
			AmountCents:     resolvedPM.amount,
			Description:     fmt.Sprintf("Action execution: %s", actionType),
		})
		if txErr != nil {
			// The connector already executed successfully. Log the failure but
			// don't fail the request — the payment transaction is for limit
			// tracking, not for billing. A missing record means slightly lax
			// monthly limit enforcement, which is safer than discarding a
			// successful execution result.
			log.Printf("[payment] failed to record transaction for %s (pm=%s amount=%d): %v",
				actionType, resolvedPM.paymentMethodID, resolvedPM.amount, txErr)
		}
	}

	return result, nil
}

// resolvedPaymentMethod holds the validated payment method state between
// validation (pre-execution) and transaction recording (post-execution).
type resolvedPaymentMethod struct {
	paymentMethodID      string
	stripePaymentMethodID string
	brand                string
	last4                string
	amount               int
}

// validatePaymentMethod validates the payment method, enforces ownership and
// spending limits, and returns the resolved payment info. Does NOT record the
// transaction — that happens after successful connector execution.
//
// Note: there is a small TOCTOU window between the monthly spend check here
// and the transaction recording after execution. A concurrent request could
// pass the limit check in this window, resulting in a slight overspend. This
// is an acceptable trade-off: the window is small (duration of connector
// execution), the consequence is minor (slightly exceeding a soft limit), and
// the alternative (locking the payment method row for the entire execution)
// would create unacceptable contention for connectors with slow external calls.
func validatePaymentMethod(ctx context.Context, deps *Deps, userID string, pp *paymentParams) (*resolvedPaymentMethod, error) {
	if pp == nil || pp.PaymentMethodID == "" {
		return nil, &connectors.PaymentError{
			Code:    connectors.PaymentErrMissing,
			Message: "this action requires a payment_method_id",
		}
	}
	if pp.AmountCents == nil {
		return nil, &connectors.PaymentError{
			Code:    connectors.PaymentErrAmountRequired,
			Message: "amount_cents is required for actions that require a payment method",
		}
	}
	amount := *pp.AmountCents
	if amount < 0 {
		return nil, &connectors.PaymentError{
			Code:    connectors.PaymentErrInvalidAmount,
			Message: "amount_cents must be non-negative",
		}
	}

	// Look up the payment method, scoped to the user.
	pm, err := db.GetPaymentMethodByID(ctx, deps.DB, userID, pp.PaymentMethodID)
	if err != nil {
		return nil, fmt.Errorf("look up payment method: %w", err)
	}
	if pm == nil {
		return nil, &connectors.PaymentError{
			Code:    connectors.PaymentErrNotFound,
			Message: "payment method not found or does not belong to this user",
		}
	}

	// Enforce per-transaction limit.
	if pm.PerTransactionLimit != nil && amount > *pm.PerTransactionLimit {
		return nil, &connectors.PaymentError{
			Code:    connectors.PaymentErrPerTxLimit,
			Message: fmt.Sprintf("amount %d cents exceeds per-transaction limit of %d cents", amount, *pm.PerTransactionLimit),
			Details: map[string]any{
				"requested_amount_cents":      amount,
				"per_transaction_limit_cents": *pm.PerTransactionLimit,
			},
		}
	}

	// Enforce monthly limit.
	if pm.MonthlyLimit != nil {
		monthlySpend, err := db.GetMonthlySpend(ctx, deps.DB, pm.ID)
		if err != nil {
			return nil, fmt.Errorf("check monthly spend: %w", err)
		}
		remaining := *pm.MonthlyLimit - monthlySpend
		if amount > remaining {
			return nil, &connectors.PaymentError{
				Code:    connectors.PaymentErrMonthlyLimit,
				Message: fmt.Sprintf("amount %d cents would exceed monthly limit of %d cents (current spend: %d cents)", amount, *pm.MonthlyLimit, monthlySpend),
				Details: map[string]any{
					"requested_amount_cents": amount,
					"monthly_limit_cents":    *pm.MonthlyLimit,
					"current_spend_cents":    monthlySpend,
					"remaining_cents":        remaining,
				},
			}
		}
	}

	return &resolvedPaymentMethod{
		paymentMethodID:       pm.ID,
		stripePaymentMethodID: pm.StripePaymentMethodID,
		brand:                 pm.Brand,
		last4:                 pm.Last4,
		amount:                amount,
	}, nil
}

// resolveCredentialsWithFallback tries to resolve credentials in priority order.
// For connectors supporting multiple auth methods (e.g. OAuth + API key), it
// tries OAuth first, then falls back to static credentials if the user hasn't
// connected their OAuth account.
func resolveCredentialsWithFallback(ctx context.Context, deps *Deps, agentID int64, userID, actionType, connectorID string, reqCreds []db.RequiredCredential) (connectors.Credentials, error) {
	// If there's a built-in OAuth provider for this connector but the manifest
	// only declares static credentials (e.g. Shopify), synthesize an oauth2
	// entry so OAuth is tried first with static as fallback.
	hasExplicitOAuth := false
	for _, rc := range reqCreds {
		if rc.AuthType == "oauth2" {
			hasExplicitOAuth = true
			break
		}
	}
	if !hasExplicitOAuth && deps.OAuthProviders != nil {
		if _, providerExists := deps.OAuthProviders.Get(connectorID); providerExists {
			reqCreds = append([]db.RequiredCredential{{
				AuthType:      "oauth2",
				OAuthProvider: &connectorID,
			}}, reqCreds...)
		}
	}

	if len(reqCreds) == 0 {
		return connectors.NewCredentials(nil), nil
	}

	// Check for an agent-specific credential binding. If one exists, use it
	// directly instead of falling through to user-global resolution.
	if agentID > 0 {
		binding, err := db.GetAgentConnectorCredential(ctx, deps.DB, agentID, connectorID)
		if err != nil {
			log.Printf("resolve agent connector credential: %v", err)
			// Fall through to global resolution on lookup error.
		} else if binding != nil {
			if binding.OAuthConnectionID != nil {
				conn, oauthErr := db.GetOAuthConnectionByID(ctx, deps.DB, *binding.OAuthConnectionID)
				if oauthErr != nil {
					return connectors.Credentials{}, fmt.Errorf("look up bound OAuth connection: %w", oauthErr)
				}
				if conn == nil {
					return connectors.Credentials{}, &connectors.ValidationError{
						Message: "bound OAuth connection no longer exists — reassign credentials for this connector",
					}
				}
				// Use the bound connection directly (not a provider re-query)
				// so the specific oauth_connection_id in the binding is honoured.
				return resolveOAuthCredentialsFromConnection(ctx, deps, conn)
			}
			if binding.CredentialID != nil {
				creds, credErr := resolveStaticCredentialByID(ctx, deps, *binding.CredentialID)
				if credErr != nil {
					return connectors.Credentials{}, credErr
				}
				return creds, nil
			}
		}
	}

	// Single credential type: use the existing path directly.
	if len(reqCreds) == 1 {
		rc := reqCreds[0]
		if rc.AuthType == "oauth2" {
			return resolveOAuthCredentials(ctx, deps, userID, &rc)
		}
		return resolveStaticCredentials(ctx, deps, userID, actionType)
	}

	// Multiple credential types: try OAuth first, fall back to static.
	for _, rc := range reqCreds {
		if rc.AuthType == "oauth2" {
			creds, err := resolveOAuthCredentials(ctx, deps, userID, &rc)
			if err == nil {
				return creds, nil
			}
			var oauthErr *connectors.OAuthRefreshError
			if errors.As(err, &oauthErr) {
				// For implicit OAuth (system-injected, not declared in the
				// connector manifest), any OAuth failure should fall back to
				// static credentials — the user's primary auth was always static.
				if !hasExplicitOAuth {
					continue
				}
				// For explicit OAuth, only fall back when the user has no
				// connection at all. If the connection exists but needs
				// re-auth or refresh failed, fail immediately so the user
				// knows to reconnect.
				if oauthErr.NotConnected {
					continue
				}
			}
			return connectors.Credentials{}, err
		}
	}

	// No OAuth connection available — fall back to static credentials.
	for _, rc := range reqCreds {
		if rc.AuthType != "oauth2" {
			creds, err := resolveStaticCredentialsByService(ctx, deps, userID, rc.Service)
			if err == nil {
				return creds, nil
			}
			// If the specific static credential isn't found either, continue
			// to try others or fall through to the final error.
			var valErr *connectors.ValidationError
			if errors.As(err, &valErr) {
				continue
			}
			return connectors.Credentials{}, err
		}
	}

	// Build a helpful message listing which credential options are available.
	var methods []string
	for _, rc := range reqCreds {
		if rc.AuthType == "oauth2" {
			provider := "OAuth"
			if rc.OAuthProvider != nil {
				provider = *rc.OAuthProvider + " OAuth"
			}
			methods = append(methods, provider+" (Settings → Connected Accounts)")
		} else {
			methods = append(methods, rc.Service+" "+rc.AuthType)
		}
	}
	return connectors.Credentials{}, &connectors.ValidationError{
		Message: fmt.Sprintf("no credentials available — configure one of: %s", strings.Join(methods, ", ")),
	}
}

// resolveStaticCredentialsByService fetches and decrypts static credentials for
// a specific service. Used when falling back from OAuth to API key.
func resolveStaticCredentialsByService(ctx context.Context, deps *Deps, userID, service string) (connectors.Credentials, error) {
	return decryptServiceCredentials(ctx, deps, userID, service)
}

// resolveStaticCredentials fetches and decrypts static credentials (api_key, basic, custom)
// for the given action type.
func resolveStaticCredentials(ctx context.Context, deps *Deps, userID, actionType string) (connectors.Credentials, error) {
	var zero connectors.Credentials
	services, err := db.GetRequiredServicesByActionType(ctx, deps.DB, actionType)
	if err != nil {
		return zero, fmt.Errorf("look up required services: %w", err)
	}
	credMap := make(map[string]string, len(services))
	for _, service := range services {
		creds, err := decryptServiceCredentials(ctx, deps, userID, service)
		if err != nil {
			return connectors.Credentials{}, err
		}
		// Merge into the combined map (multi-service connectors need all credentials).
		for k, v := range creds.ToMap() {
			credMap[k] = v
		}
	}
	return connectors.NewCredentials(credMap), nil
}

// decryptServiceCredentials fetches and decrypts credentials for a single
// service from the vault. Shared by resolveStaticCredentials (multi-service)
// and resolveStaticCredentialsByService (single-service fallback).
func decryptServiceCredentials(ctx context.Context, deps *Deps, userID, service string) (connectors.Credentials, error) {
	var zero connectors.Credentials
	if deps.Vault == nil {
		return zero, fmt.Errorf("credential vault is not configured but connector requires service %q", service)
	}
	decrypted, err := db.GetDecryptedCredentials(ctx, deps.DB, deps.Vault.ReadSecret, userID, service, nil)
	if err != nil {
		var credErr *db.CredentialError
		if errors.As(err, &credErr) && credErr.Code == db.CredentialErrNotFound {
			return zero, &connectors.ValidationError{
				Message: fmt.Sprintf("no credentials stored for service %q", service),
			}
		}
		return zero, fmt.Errorf("decrypt credentials for service %q: %w", service, err)
	}
	credMap := make(map[string]string, len(decrypted))
	for k, v := range decrypted {
		switch vv := v.(type) {
		case string:
			credMap[k] = vv
		default:
			b, jsonErr := json.Marshal(v)
			if jsonErr != nil {
				return zero, fmt.Errorf("marshal credential %q for service %q: %w", k, service, jsonErr)
			}
			credMap[k] = string(b)
		}
	}
	return connectors.NewCredentials(credMap), nil
}

// resolveStaticCredentialByID fetches and decrypts a specific credential by its
// ID (from the agent_connector_credentials binding). This bypasses the
// service-based lookup and goes straight to the vault.
func resolveStaticCredentialByID(ctx context.Context, deps *Deps, credentialID string) (connectors.Credentials, error) {
	var zero connectors.Credentials
	if deps.Vault == nil {
		return zero, fmt.Errorf("credential vault is not configured")
	}
	vaultSecretID, err := db.GetVaultSecretIDByCredentialID(ctx, deps.DB, credentialID)
	if err != nil {
		var credErr *db.CredentialError
		if errors.As(err, &credErr) && credErr.Code == db.CredentialErrNotFound {
			return zero, &connectors.ValidationError{
				Message: "bound credential no longer exists — reassign credentials for this connector",
			}
		}
		return zero, fmt.Errorf("look up bound credential: %w", err)
	}
	raw, err := deps.Vault.ReadSecret(ctx, deps.DB, vaultSecretID)
	if err != nil {
		return zero, fmt.Errorf("decrypt bound credential: %w", err)
	}
	var parsed map[string]any
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return zero, fmt.Errorf("unmarshal bound credential: %w", err)
	}
	credMap := make(map[string]string, len(parsed))
	for k, v := range parsed {
		switch vv := v.(type) {
		case string:
			credMap[k] = vv
		default:
			b, jsonErr := json.Marshal(v)
			if jsonErr != nil {
				return zero, fmt.Errorf("marshal credential field %q: %w", k, jsonErr)
			}
			credMap[k] = string(b)
		}
	}
	return connectors.NewCredentials(credMap), nil
}

// resolveOAuthCredentials looks up the user's OAuth connection for the required
// provider, refreshes the token if it's expired or near-expiry, and returns
// credentials containing the valid access token.
func resolveOAuthCredentials(ctx context.Context, deps *Deps, userID string, reqCred *db.RequiredCredential) (connectors.Credentials, error) {
	var zero connectors.Credentials
	if deps.Vault == nil {
		return zero, fmt.Errorf("credential vault is not configured but connector requires OAuth credentials")
	}
	if deps.OAuthProviders == nil {
		return zero, fmt.Errorf("OAuth provider registry is not configured")
	}

	providerID := ""
	if reqCred.OAuthProvider != nil {
		providerID = *reqCred.OAuthProvider
	}
	if providerID == "" {
		return zero, &connectors.ValidationError{
			Message: "connector requires OAuth but no oauth_provider is configured",
		}
	}

	// Look up the user's OAuth connection for this provider.
	conn, err := db.GetOAuthConnectionByProvider(ctx, deps.DB, userID, providerID)
	if err != nil {
		return zero, fmt.Errorf("look up OAuth connection for provider %q: %w", providerID, err)
	}
	if conn == nil {
		return zero, &connectors.OAuthRefreshError{
			Provider:     providerID,
			Message:      fmt.Sprintf("no OAuth connection for provider %q — user must connect via Settings", providerID),
			NotConnected: true,
		}
	}

	// Check connection status (allowlist: only "active" connections may be used).
	// Using != active instead of checking individual non-active statuses ensures
	// that any new status added in the future is denied by default.
	if conn.Status != db.OAuthStatusActive {
		return zero, &connectors.OAuthRefreshError{
			Provider: providerID,
			Message:  fmt.Sprintf("OAuth connection for %q has status %q — user must re-authorize", providerID, conn.Status),
		}
	}

	// Check if the token needs refreshing (expired or within pre-emptive buffer).
	if conn.TokenExpiry != nil && time.Now().After(conn.TokenExpiry.Add(-oauth.TokenExpiryBuffer)) {
		if err := refreshOAuthConnection(ctx, deps, conn, providerID); err != nil {
			return zero, err
		}
	}

	// Read the (possibly refreshed) access token from the vault.
	accessTokenBytes, err := deps.Vault.ReadSecret(ctx, deps.DB, conn.AccessTokenVaultID)
	if err != nil {
		return zero, fmt.Errorf("read access token from vault: %w", err)
	}

	creds := map[string]string{}

	// Merge provider-specific extra data (e.g. instance_url) into credentials
	// so connectors can access them alongside the access token.
	if len(conn.ExtraData) > 0 {
		var extra map[string]string
		if err := json.Unmarshal(conn.ExtraData, &extra); err == nil {
			for k, v := range extra {
				creds[k] = v
			}
		}
	}

	// Set access_token AFTER merging extra_data so that a tampered extra_data
	// field cannot overwrite the vault-sourced access token.
	creds["access_token"] = string(accessTokenBytes)

	return connectors.NewCredentials(creds), nil
}

// resolveOAuthCredentialsFromConnection resolves credentials from a specific
// OAuth connection (already fetched by ID). This is used by agent-specific
// credential bindings to honour the exact bound connection rather than
// re-querying by provider+user, which could return a different connection.
func resolveOAuthCredentialsFromConnection(ctx context.Context, deps *Deps, conn *db.OAuthConnection) (connectors.Credentials, error) {
	var zero connectors.Credentials
	if deps.Vault == nil {
		return zero, fmt.Errorf("credential vault is not configured but connector requires OAuth credentials")
	}
	if deps.OAuthProviders == nil {
		return zero, fmt.Errorf("OAuth provider registry is not configured")
	}

	if conn.Status != db.OAuthStatusActive {
		return zero, &connectors.OAuthRefreshError{
			Provider: conn.Provider,
			Message:  fmt.Sprintf("bound OAuth connection has status %q — user must re-authorize", conn.Status),
		}
	}

	// Refresh the token if expired or near-expiry.
	if conn.TokenExpiry != nil && time.Now().After(conn.TokenExpiry.Add(-oauth.TokenExpiryBuffer)) {
		if err := refreshOAuthConnection(ctx, deps, conn, conn.Provider); err != nil {
			return zero, err
		}
	}

	// Read the (possibly refreshed) access token from the vault.
	accessTokenBytes, err := deps.Vault.ReadSecret(ctx, deps.DB, conn.AccessTokenVaultID)
	if err != nil {
		return zero, fmt.Errorf("read access token from vault: %w", err)
	}

	creds := map[string]string{}
	if len(conn.ExtraData) > 0 {
		var extra map[string]string
		if err := json.Unmarshal(conn.ExtraData, &extra); err == nil {
			for k, v := range extra {
				creds[k] = v
			}
		}
	}
	creds["access_token"] = string(accessTokenBytes)

	return connectors.NewCredentials(creds), nil
}

// refreshOAuthConnection performs a token refresh for the given OAuth connection.
// On success, it updates the vault secrets and DB row, and sets
// conn.AccessTokenVaultID to the new vault ID so the caller reads the fresh token.
// On failure, it marks the connection as needs_reauth and returns an OAuthRefreshError.
//
// Note: if the background refresh job refreshes the same connection concurrently,
// both paths produce valid tokens. The last writer wins in the DB, and the other's
// vault secrets become orphaned. This is safe (no data loss or auth failure) but
// may leave unused vault entries until the connection is deleted or re-authorized.
func refreshOAuthConnection(ctx context.Context, deps *Deps, conn *db.OAuthConnection, providerID string) error {
	provider, ok := deps.OAuthProviders.Get(providerID)
	if !ok {
		return &connectors.OAuthRefreshError{
			Provider: providerID,
			Message:  fmt.Sprintf("OAuth provider %q is not registered", providerID),
		}
	}

	if conn.RefreshTokenVaultID == nil {
		// No refresh token — can't refresh. Mark as needs_reauth.
		if statusErr := db.UpdateOAuthConnectionStatus(ctx, deps.DB, conn.ID, conn.UserID, db.OAuthStatusNeedsReauth); statusErr != nil {
			log.Printf("failed to update OAuth connection status to needs_reauth: %v", statusErr)
		}
		return &connectors.OAuthRefreshError{
			Provider: providerID,
			Message:  "token expired and no refresh token available — user must re-authorize",
		}
	}

	// Read the refresh token from the vault.
	refreshTokenBytes, err := deps.Vault.ReadSecret(ctx, deps.DB, *conn.RefreshTokenVaultID)
	if err != nil {
		return fmt.Errorf("read refresh token from vault: %w", err)
	}

	// Perform the refresh with a dedicated timeout so the OAuth provider call
	// gets a fair chance even if the incoming request context has little time left.
	// Use WithoutCancel to preserve context values (trace IDs, Sentry spans) while
	// detaching from the parent's cancellation signal.
	refreshCtx, refreshCancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer refreshCancel()
	result, err := oauth.RefreshTokens(refreshCtx, provider, string(refreshTokenBytes))
	if err != nil {
		// Refresh failed (token revoked, expired, etc.). Mark as needs_reauth.
		// Log the full error server-side for debugging; return a sanitized message to the caller.
		log.Printf("oauth refresh failed for provider %q connection %s: %v", providerID, conn.ID, err)
		if statusErr := db.UpdateOAuthConnectionStatus(ctx, deps.DB, conn.ID, conn.UserID, db.OAuthStatusNeedsReauth); statusErr != nil {
			log.Printf("failed to update OAuth connection status to needs_reauth: %v", statusErr)
		}
		return &connectors.OAuthRefreshError{
			Provider: providerID,
			Message:  "token refresh failed — user must re-authorize",
		}
	}

	// Store the new tokens in the vault and update the DB row.
	tx, owned, txErr := db.BeginOrContinue(ctx, deps.DB)
	if txErr != nil {
		return fmt.Errorf("begin tx for token refresh: %w", txErr)
	}
	if owned {
		defer db.RollbackTx(ctx, tx) //nolint:errcheck // best-effort cleanup
	}

	vaultOps := oauth.VaultOperations{
		DeleteSecret: func(ctx context.Context, id string) error { return deps.Vault.DeleteSecret(ctx, tx, id) },
		CreateSecret: func(ctx context.Context, name string, secret []byte) (string, error) {
			return deps.Vault.CreateSecret(ctx, tx, name, secret)
		},
	}

	newAccessVaultID, err := oauth.StoreRefreshedTokens(ctx, vaultOps,
		oauth.StoreRefreshedTokensParams{
			ConnID:            conn.ID,
			OldAccessVaultID:  conn.AccessTokenVaultID,
			OldRefreshVaultID: conn.RefreshTokenVaultID,
			Result:            result,
			OldRefreshToken:   string(refreshTokenBytes),
		},
		func(what, vaultID string, delErr error) {
			log.Printf("oauth refresh: failed to delete old %s %s from vault: %v", what, vaultID, delErr)
		},
		func(ctx context.Context, accessVaultID string, refreshVaultID *string, expiry *time.Time) error {
			return db.UpdateOAuthConnectionTokens(ctx, tx, conn.ID, conn.UserID, accessVaultID, refreshVaultID, expiry)
		},
	)
	if err != nil {
		return err
	}

	if owned {
		if err := db.CommitTx(ctx, tx); err != nil {
			return fmt.Errorf("commit token refresh: %w", err)
		}
	}

	// Update the in-memory connection so the caller reads the new vault ID.
	conn.AccessTokenVaultID = newAccessVaultID

	return nil
}


