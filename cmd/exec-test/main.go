// Command exec-test is a development utility for testing connector execution
// wiring without going through the full approval flow. It supports two modes:
//
//  1. Direct credentials — pass credentials via -cred flags for quick connector tests.
//  2. Database credentials — specify -user and -agent-id to resolve credentials
//     from the local database exactly like production does (vault decrypt, OAuth
//     token refresh, agent credential bindings).
//
// Usage:
//
//	# List all registered connectors and actions
//	go run ./cmd/exec-test -list
//
//	# Direct credential mode (quick test)
//	go run ./cmd/exec-test -action github.list_repos \
//	  -cred api_key=ghp_xxx -params '{"owner":"supersuit-tech"}'
//
//	# Database credential mode (full pipeline test)
//	go run ./cmd/exec-test -action github.list_repos \
//	  -user "user-uuid" -agent-id 42 -params '{"owner":"supersuit-tech"}'
//
//	# Dry-run (validate wiring without calling external service)
//	go run ./cmd/exec-test -action github.list_repos \
//	  -cred api_key=ghp_xxx -params '{}' -dry-run
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/supersuit-tech/permission-slip-web/connectors"
	_ "github.com/supersuit-tech/permission-slip-web/connectors/all"
	"github.com/supersuit-tech/permission-slip-web/db"
	poauth "github.com/supersuit-tech/permission-slip-web/oauth"
	_ "github.com/supersuit-tech/permission-slip-web/oauth/providers"
	"github.com/supersuit-tech/permission-slip-web/vault"
)

// credFlag collects multiple -cred key=value flags.
type credFlag []string

func (f *credFlag) String() string { return strings.Join(*f, ", ") }
func (f *credFlag) Set(v string) error {
	*f = append(*f, v)
	return nil
}

func main() {
	var (
		listActions bool
		actionType  string
		paramsJSON  string
		userID      string
		agentID     int64
		dryRun      bool
		timeout     time.Duration
		creds       credFlag
	)

	flag.BoolVar(&listActions, "list", false, "List all registered connectors and their actions")
	flag.StringVar(&actionType, "action", "", "Action type to execute (e.g. github.create_issue)")
	flag.Var(&creds, "cred", "Credential key=value pair (repeatable, e.g. -cred api_key=ghp_xxx)")
	flag.StringVar(&paramsJSON, "params", "{}", "Action parameters as JSON")
	flag.StringVar(&userID, "user", "", "User ID for database credential resolution")
	flag.Int64Var(&agentID, "agent-id", 0, "Agent ID for database credential resolution")
	flag.BoolVar(&dryRun, "dry-run", false, "Validate wiring without calling the external service")
	flag.DurationVar(&timeout, "timeout", 30*time.Second, "Execution timeout")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "exec-test — development utility for testing connector execution\n\n")
		fmt.Fprintf(os.Stderr, "Modes:\n")
		fmt.Fprintf(os.Stderr, "  Direct:   -action <type> -cred key=value [-cred ...] -params '{...}'\n")
		fmt.Fprintf(os.Stderr, "  Database: -action <type> -user <uuid> -agent-id <id> -params '{...}'\n")
		fmt.Fprintf(os.Stderr, "  List:     -list\n\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	// Load .env for DATABASE_URL etc.
	_ = godotenv.Load()

	// Build connector registry (same as production).
	registry := connectors.NewRegistry()
	for _, c := range connectors.BuiltInConnectors() {
		registry.Register(c)
	}

	if listActions {
		printActions(registry)
		return
	}

	if actionType == "" {
		fmt.Fprintln(os.Stderr, "error: -action is required (or use -list)")
		flag.Usage()
		os.Exit(1)
	}

	// Validate the action exists in the registry.
	action, ok := registry.GetAction(actionType)
	if !ok {
		fmt.Fprintf(os.Stderr, "error: action %q not found in registry\n", actionType)
		fmt.Fprintln(os.Stderr, "Use -list to see available actions.")
		os.Exit(1)
	}

	// Parse parameters.
	var params json.RawMessage
	if err := json.Unmarshal([]byte(paramsJSON), &params); err != nil {
		fmt.Fprintf(os.Stderr, "error: invalid -params JSON: %v\n", err)
		os.Exit(1)
	}

	// Resolve credentials.
	connectorID := strings.SplitN(actionType, ".", 2)[0]
	resolved, err := resolveCredentials(creds, userID, agentID, connectorID, registry)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error resolving credentials: %v\n", err)
		os.Exit(1)
	}

	// Validate credentials with the connector.
	conn, connOK := registry.Get(connectorID)
	if connOK {
		ctx := context.Background()
		if err := conn.ValidateCredentials(ctx, resolved); err != nil {
			fmt.Fprintf(os.Stderr, "credential validation failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "[ok] credentials validated for connector %q (keys: %v)\n", connectorID, resolved.Keys())
	}

	if dryRun {
		fmt.Fprintln(os.Stderr, "[dry-run] would execute:")
		fmt.Fprintf(os.Stderr, "  action:     %s\n", actionType)
		fmt.Fprintf(os.Stderr, "  connector:  %s\n", connectorID)
		fmt.Fprintf(os.Stderr, "  cred keys:  %v\n", resolved.Keys())
		fmt.Fprintf(os.Stderr, "  params:     %s\n", string(params))
		fmt.Fprintln(os.Stderr, "[dry-run] skipping actual execution")
		return
	}

	// Execute.
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	fmt.Fprintf(os.Stderr, "[exec] executing %s ...\n", actionType)
	start := time.Now()

	result, err := action.Execute(ctx, connectors.ActionRequest{
		ActionType:  actionType,
		Parameters:  params,
		Credentials: resolved,
	})
	elapsed := time.Since(start)

	if err != nil {
		fmt.Fprintf(os.Stderr, "[fail] %s after %s\n", actionType, elapsed.Round(time.Millisecond))
		printErrorDetails(err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "[ok] %s completed in %s\n", actionType, elapsed.Round(time.Millisecond))

	if result != nil && result.Data != nil {
		// Pretty-print the result JSON to stdout.
		var pretty json.RawMessage
		if json.Unmarshal(result.Data, &pretty) == nil {
			out, _ := json.MarshalIndent(pretty, "", "  ")
			fmt.Println(string(out))
		} else {
			fmt.Println(string(result.Data))
		}
	} else {
		fmt.Fprintln(os.Stderr, "(no result data)")
	}
}

// printActions lists all registered connectors and their actions.
func printActions(registry *connectors.Registry) {
	ids := registry.IDs()
	sort.Strings(ids)

	if len(ids) == 0 {
		fmt.Println("No connectors registered.")
		return
	}

	total := 0
	for _, id := range ids {
		conn, _ := registry.Get(id)
		actions := conn.Actions()
		actionNames := make([]string, 0, len(actions))
		for name := range actions {
			actionNames = append(actionNames, name)
		}
		sort.Strings(actionNames)
		total += len(actionNames)

		fmt.Printf("\n%s (%d actions)\n", id, len(actionNames))
		for _, name := range actionNames {
			fmt.Printf("  %s\n", name)
		}
	}
	fmt.Printf("\n%d connectors, %d actions total\n", len(ids), total)
}

// resolveCredentials resolves credentials either from -cred flags (direct mode)
// or from the database (database mode).
func resolveCredentials(creds credFlag, userID string, agentID int64, connectorID string, registry *connectors.Registry) (connectors.Credentials, error) {
	if len(creds) > 0 {
		return resolveDirectCredentials(creds)
	}
	if userID != "" {
		return resolveDatabaseCredentials(userID, agentID, connectorID, registry)
	}
	// No credentials provided — some connectors may not require them.
	return connectors.NewCredentials(nil), nil
}

// resolveDirectCredentials parses key=value pairs from -cred flags.
func resolveDirectCredentials(creds credFlag) (connectors.Credentials, error) {
	m := make(map[string]string, len(creds))
	for _, kv := range creds {
		k, v, ok := strings.Cut(kv, "=")
		if !ok {
			return connectors.Credentials{}, fmt.Errorf("invalid -cred format %q (expected key=value)", kv)
		}
		m[k] = v
	}
	return connectors.NewCredentials(m), nil
}

// resolveDatabaseCredentials connects to the local database, looks up the
// agent's credential binding, decrypts from vault, and refreshes OAuth tokens
// if needed — exactly the same code path as production.
func resolveDatabaseCredentials(userID string, agentID int64, connectorID string, registry *connectors.Registry) (connectors.Credentials, error) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return connectors.Credentials{}, fmt.Errorf("DATABASE_URL is required for database credential resolution (set it in .env)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := db.Connect(ctx, dbURL)
	if err != nil {
		return connectors.Credentials{}, fmt.Errorf("connect to database: %w", err)
	}
	defer pool.Close()

	fmt.Fprintf(os.Stderr, "[db] connected to database\n")

	// Look up the agent's credential binding for this connector.
	binding, err := db.GetAgentConnectorCredential(ctx, pool, agentID, connectorID)
	if err != nil {
		return connectors.Credentials{}, fmt.Errorf("look up agent connector credential: %w", err)
	}
	if binding == nil {
		return connectors.Credentials{}, fmt.Errorf("no credential binding found for agent %d + connector %q — assign one in the dashboard first", agentID, connectorID)
	}

	fmt.Fprintf(os.Stderr, "[db] found credential binding (oauth=%v, static=%v)\n",
		binding.OAuthConnectionID != nil, binding.CredentialID != nil)

	// Resolve based on credential type.
	if binding.OAuthConnectionID != nil {
		return resolveOAuthFromDB(ctx, pool, userID, *binding.OAuthConnectionID)
	}
	if binding.CredentialID != nil {
		return resolveStaticFromDB(ctx, pool, userID, *binding.CredentialID)
	}

	return connectors.Credentials{}, fmt.Errorf("credential binding is incomplete — no OAuth connection or static credential assigned")
}

// resolveOAuthFromDB fetches an OAuth connection, refreshes if needed, and
// returns the decrypted access token as credentials.
func resolveOAuthFromDB(ctx context.Context, pool db.DBTX, userID, oauthConnID string) (connectors.Credentials, error) {
	conn, err := db.GetOAuthConnectionByID(ctx, pool, oauthConnID)
	if err != nil {
		return connectors.Credentials{}, fmt.Errorf("look up OAuth connection: %w", err)
	}
	if conn == nil {
		return connectors.Credentials{}, fmt.Errorf("OAuth connection %q not found", oauthConnID)
	}
	if conn.UserID != userID {
		return connectors.Credentials{}, fmt.Errorf("OAuth connection %q does not belong to user %q", oauthConnID, userID)
	}

	fmt.Fprintf(os.Stderr, "[oauth] connection found: provider=%s status=%s\n", conn.Provider, conn.Status)

	if conn.Status != db.OAuthStatusActive {
		return connectors.Credentials{}, fmt.Errorf("OAuth connection status is %q — user must re-authorize", conn.Status)
	}

	// Check token expiry.
	if conn.TokenExpiry != nil {
		remaining := time.Until(*conn.TokenExpiry)
		fmt.Fprintf(os.Stderr, "[oauth] token expires in %s\n", remaining.Round(time.Second))
		if remaining < 30*time.Second {
			fmt.Fprintf(os.Stderr, "[oauth] token is expired or near-expiry, attempting refresh...\n")
			if err := refreshOAuthToken(ctx, pool, conn); err != nil {
				return connectors.Credentials{}, fmt.Errorf("OAuth token refresh failed: %w", err)
			}
			fmt.Fprintf(os.Stderr, "[oauth] token refreshed successfully\n")
		}
	}

	// Read access token from vault.
	v := vault.NewSupabaseVaultStore()
	accessTokenBytes, err := v.ReadSecret(ctx, pool, conn.AccessTokenVaultID)
	if err != nil {
		return connectors.Credentials{}, fmt.Errorf("read access token from vault: %w", err)
	}

	creds := map[string]string{}
	// Merge provider-specific extra data.
	if len(conn.ExtraData) > 0 {
		var extra map[string]string
		if json.Unmarshal(conn.ExtraData, &extra) == nil {
			for k, v := range extra {
				creds[k] = v
			}
		}
	}
	creds["access_token"] = string(accessTokenBytes)

	fmt.Fprintf(os.Stderr, "[oauth] credentials resolved (keys: %v)\n", keysOf(creds))
	return connectors.NewCredentials(creds), nil
}

// refreshOAuthToken attempts to refresh an expired OAuth token.
func refreshOAuthToken(ctx context.Context, pool db.DBTX, conn *db.OAuthConnection) error {
	oauthRegistry := poauth.NewRegistryWithBuiltIns()
	provider, ok := oauthRegistry.Get(conn.Provider)
	if !ok {
		return fmt.Errorf("OAuth provider %q not registered", conn.Provider)
	}

	v := vault.NewSupabaseVaultStore()
	if conn.RefreshTokenVaultID == nil {
		return fmt.Errorf("no refresh token available — user must re-authorize")
	}

	refreshTokenBytes, err := v.ReadSecret(ctx, pool, *conn.RefreshTokenVaultID)
	if err != nil {
		return fmt.Errorf("read refresh token from vault: %w", err)
	}

	refreshCtx, refreshCancel := context.WithTimeout(ctx, 10*time.Second)
	defer refreshCancel()

	result, err := poauth.RefreshTokens(refreshCtx, provider, string(refreshTokenBytes))
	if err != nil {
		return fmt.Errorf("token refresh request failed: %w", err)
	}

	// Store new tokens. We need a real transaction for vault operations.
	txStarter, ok := pool.(db.TxStarter)
	if !ok {
		return fmt.Errorf("database connection does not support transactions (needed for vault writes)")
	}

	tx, err := txStarter.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// Delete old access token, store new one.
	_ = v.DeleteSecret(ctx, tx, conn.AccessTokenVaultID)
	newAccessVaultID, err := v.CreateSecret(ctx, tx, fmt.Sprintf("oauth_access_%s", conn.ID), []byte(result.AccessToken))
	if err != nil {
		return fmt.Errorf("store new access token: %w", err)
	}

	// Handle refresh token rotation.
	var newRefreshVaultID *string
	newRefreshToken := result.RefreshToken
	if newRefreshToken == "" {
		newRefreshToken = string(refreshTokenBytes) // keep old refresh token
	}
	if conn.RefreshTokenVaultID != nil {
		_ = v.DeleteSecret(ctx, tx, *conn.RefreshTokenVaultID)
	}
	rtVaultID, err := v.CreateSecret(ctx, tx, fmt.Sprintf("oauth_refresh_%s", conn.ID), []byte(newRefreshToken))
	if err != nil {
		return fmt.Errorf("store refresh token: %w", err)
	}
	newRefreshVaultID = &rtVaultID

	// Update DB row.
	expiry := result.Expiry
	if err := db.UpdateOAuthConnectionTokens(ctx, tx, conn.ID, conn.UserID, newAccessVaultID, newRefreshVaultID, &expiry); err != nil {
		return fmt.Errorf("update OAuth connection tokens: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit token refresh: %w", err)
	}

	// Update in-memory so caller reads the new vault ID.
	conn.AccessTokenVaultID = newAccessVaultID
	return nil
}

// resolveStaticFromDB fetches and decrypts a static credential from the vault.
func resolveStaticFromDB(ctx context.Context, pool db.DBTX, userID, credentialID string) (connectors.Credentials, error) {
	// Verify credential ownership.
	owned, err := db.CredentialBelongsToUser(ctx, pool, credentialID, userID)
	if err != nil {
		return connectors.Credentials{}, fmt.Errorf("check credential ownership: %w", err)
	}
	if !owned {
		return connectors.Credentials{}, fmt.Errorf("credential %q does not belong to user %q", credentialID, userID)
	}

	vaultSecretID, err := db.GetVaultSecretIDByCredentialID(ctx, pool, credentialID)
	if err != nil {
		return connectors.Credentials{}, fmt.Errorf("look up vault secret: %w", err)
	}

	v := vault.NewSupabaseVaultStore()
	raw, err := v.ReadSecret(ctx, pool, vaultSecretID)
	if err != nil {
		return connectors.Credentials{}, fmt.Errorf("decrypt credential: %w", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return connectors.Credentials{}, fmt.Errorf("unmarshal credential: %w", err)
	}

	credMap := make(map[string]string, len(parsed))
	for k, val := range parsed {
		switch vv := val.(type) {
		case string:
			credMap[k] = vv
		default:
			b, _ := json.Marshal(val)
			credMap[k] = string(b)
		}
	}

	fmt.Fprintf(os.Stderr, "[db] static credential resolved (keys: %v)\n", keysOf(credMap))
	return connectors.NewCredentials(credMap), nil
}

// printErrorDetails prints a structured breakdown of connector errors.
func printErrorDetails(err error) {
	fmt.Fprintf(os.Stderr, "  error: %v\n", err)

	switch {
	case connectors.IsValidationError(err):
		fmt.Fprintln(os.Stderr, "  type:  ValidationError (bad parameters or credentials)")
	case connectors.IsAuthError(err):
		fmt.Fprintln(os.Stderr, "  type:  AuthError (credentials rejected by external service)")
	case connectors.IsExternalError(err):
		fmt.Fprintln(os.Stderr, "  type:  ExternalError (external service returned an error)")
	case connectors.IsTimeoutError(err):
		fmt.Fprintln(os.Stderr, "  type:  TimeoutError (external service didn't respond)")
	case connectors.IsRateLimitError(err):
		fmt.Fprintln(os.Stderr, "  type:  RateLimitError (rate limited by external service)")
	case connectors.IsOAuthRefreshError(err):
		fmt.Fprintln(os.Stderr, "  type:  OAuthRefreshError (token refresh failed — user must re-authorize)")
	default:
		fmt.Fprintln(os.Stderr, "  type:  unexpected error (possible bug in connector or wiring)")
	}
}

func keysOf(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
