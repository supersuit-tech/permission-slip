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
func executeConnectorAction(ctx context.Context, deps *Deps, userID, actionType string, parameters json.RawMessage, pp *paymentParams) (*connectors.ActionResult, error) {
	if deps.Connectors == nil {
		return nil, nil
	}

	action, ok := deps.Connectors.GetAction(actionType)
	if !ok {
		return nil, nil
	}

	// Resolve credentials. Use the multi-credential query to support connectors
	// that offer both OAuth and API key auth (e.g. Netlify). The query returns
	// credentials ordered with oauth2 first so we try it before falling back.
	allReqCreds, err := db.GetAllRequiredCredentialsByActionType(ctx, deps.DB, actionType)
	if err != nil {
		return nil, fmt.Errorf("look up required credentials: %w", err)
	}

	var creds connectors.Credentials
	creds, err = resolveCredentialsWithFallback(ctx, deps, userID, actionType, allReqCreds)
	if err != nil {
		return nil, err
	}

	// Validate credentials before executing the action.
	connectorID := strings.SplitN(actionType, ".", 2)[0]
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

// resolveCredentialsWithFallback tries to resolve credentials from the list of
// required credentials. For connectors that support multiple auth methods (e.g.
// both OAuth and API key), it tries OAuth first and falls back to static
// credentials if the user hasn't connected via OAuth. For single-auth connectors,
// it behaves identically to the previous logic.
func resolveCredentialsWithFallback(ctx context.Context, deps *Deps, userID, actionType string, reqCreds []db.RequiredCredential) (connectors.Credentials, error) {
	if len(reqCreds) == 0 {
		return connectors.Credentials{}, nil
	}

	// Separate OAuth and static credential entries.
	var oauthCred *db.RequiredCredential
	hasStaticCred := false
	for i := range reqCreds {
		if reqCreds[i].AuthType == "oauth2" && oauthCred == nil {
			oauthCred = &reqCreds[i]
		} else if reqCreds[i].AuthType != "oauth2" {
			hasStaticCred = true
		}
	}

	// If OAuth is available, try it first.
	if oauthCred != nil {
		creds, err := resolveOAuthCredentials(ctx, deps, userID, oauthCred)
		if err == nil {
			return creds, nil
		}
		// If there's a static fallback and the OAuth error indicates no connection
		// (not a refresh failure or status issue), fall back to static credentials.
		if hasStaticCred {
			var oauthErr *connectors.OAuthRefreshError
			if errors.As(err, &oauthErr) && strings.Contains(oauthErr.Message, "no OAuth connection") {
				return resolveStaticCredentials(ctx, deps, userID, actionType)
			}
		}
		return connectors.Credentials{}, err
	}

	// No OAuth credential — use static path.
	return resolveStaticCredentials(ctx, deps, userID, actionType)
}

// resolveStaticCredentials fetches and decrypts static credentials (api_key, basic, custom)
// for the given action type.
func resolveStaticCredentials(ctx context.Context, deps *Deps, userID, actionType string) (connectors.Credentials, error) {
	services, err := db.GetRequiredServicesByActionType(ctx, deps.DB, actionType)
	if err != nil {
		return connectors.Credentials{}, fmt.Errorf("look up required services: %w", err)
	}

	var zero connectors.Credentials
	credMap := make(map[string]string, len(services))
	for _, service := range services {
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
		// Flatten decrypted JSON map into string values for the Credentials type.
		for k, v := range decrypted {
			switch vv := v.(type) {
			case string:
				credMap[k] = vv
			default:
				b, err := json.Marshal(v)
				if err != nil {
					return zero, fmt.Errorf("marshal credential %q for service %q: %w", k, service, err)
				}
				credMap[k] = string(b)
			}
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
			Provider: providerID,
			Message:  fmt.Sprintf("no OAuth connection for provider %q — user must connect via Settings", providerID),
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
