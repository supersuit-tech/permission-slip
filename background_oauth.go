package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/supersuit-tech/permission-slip-web/db"
	"github.com/supersuit-tech/permission-slip-web/oauth"
	"github.com/supersuit-tech/permission-slip-web/vault"
)

// oauthRefreshHorizon is the time window before token expiry within which
// the background job proactively refreshes tokens. This prevents execution-time
// latency from refresh calls.
const oauthRefreshHorizon = 15 * time.Minute

// OAuthRefreshDeps holds the dependencies for the background OAuth token refresh job.
type OAuthRefreshDeps struct {
	DB       db.DBTX
	Vault    vault.VaultStore
	Registry *oauth.Registry
}

// startOAuthRefresh runs a periodic background job that proactively refreshes
// OAuth tokens expiring within the refresh horizon. It returns a channel that
// is closed when the goroutine exits, so callers can wait for it during shutdown.
func startOAuthRefresh(ctx context.Context, deps OAuthRefreshDeps, logger *slog.Logger) <-chan struct{} {
	interval := oauthRefreshInterval(logger)
	logger.Info("oauth refresh: scheduled", "interval", interval.String(),
		"horizon", oauthRefreshHorizon.String())

	done := make(chan struct{})
	go func() {
		defer close(done)

		// Run once immediately on startup.
		runOAuthRefresh(ctx, deps, logger)

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				logger.Info("oauth refresh: stopped")
				return
			case <-ticker.C:
				runOAuthRefresh(ctx, deps, logger)
			}
		}
	}()
	return done
}

// runOAuthRefresh executes a single refresh pass with a 2-minute timeout.
func runOAuthRefresh(ctx context.Context, deps OAuthRefreshDeps, logger *slog.Logger) {
	if deps.DB == nil {
		logger.Debug("oauth refresh: skipped (no database)")
		return
	}

	refreshCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	conns, err := db.ListExpiringOAuthConnections(refreshCtx, deps.DB, oauthRefreshHorizon)
	if err != nil {
		if ctx.Err() != nil {
			logger.Info("oauth refresh: cancelled", "error", err)
			return
		}
		logger.Error("oauth refresh: failed to list expiring connections", "error", err)
		sentry.CaptureException(err)
		return
	}

	if len(conns) == 0 {
		logger.Debug("oauth refresh: no tokens expiring soon")
		return
	}

	start := time.Now()
	var refreshed, failed int
	for _, conn := range conns {
		if err := refreshSingleConnection(refreshCtx, deps, logger, conn); err != nil {
			failed++
			logger.Warn("oauth refresh: failed",
				"connection_id", conn.ID,
				"provider", conn.Provider,
				"error", err)
		} else {
			refreshed++
		}
	}

	logger.Info("oauth refresh: completed",
		"total", len(conns), "refreshed", refreshed, "failed", failed,
		"duration_ms", time.Since(start).Milliseconds())
}

// refreshSingleConnection refreshes the tokens for a single OAuth connection.
func refreshSingleConnection(ctx context.Context, deps OAuthRefreshDeps, logger *slog.Logger, conn db.OAuthConnection) error {
	provider, ok := deps.Registry.Get(conn.Provider)
	if !ok {
		return fmt.Errorf("provider %q not found in registry", conn.Provider)
	}

	logAttrs := []any{
		"connection_id", conn.ID,
		"provider", conn.Provider,
		"user_id", conn.UserID,
	}

	if conn.RefreshTokenVaultID == nil {
		// Mark as needs_reauth — no refresh token available.
		if err := db.UpdateOAuthConnectionStatus(ctx, deps.DB, conn.ID, conn.UserID, db.OAuthStatusNeedsReauth); err != nil {
			logger.Error("oauth refresh: failed to mark connection as needs_reauth",
				append(logAttrs, "error", err)...)
			sentry.CaptureException(fmt.Errorf("oauth refresh status update failed for %s: %w", conn.ID, err))
		}
		return fmt.Errorf("no refresh token for connection %s", conn.ID)
	}

	refreshTokenBytes, err := deps.Vault.ReadSecret(ctx, deps.DB, *conn.RefreshTokenVaultID)
	if err != nil {
		return fmt.Errorf("read refresh token: %w", err)
	}

	result, err := oauth.RefreshTokens(ctx, provider, string(refreshTokenBytes))
	if err != nil {
		// Refresh failed — mark as needs_reauth.
		if statusErr := db.UpdateOAuthConnectionStatus(ctx, deps.DB, conn.ID, conn.UserID, db.OAuthStatusNeedsReauth); statusErr != nil {
			logger.Error("oauth refresh: failed to mark connection as needs_reauth",
				append(logAttrs, "error", statusErr)...)
			sentry.CaptureException(fmt.Errorf("oauth refresh status update failed for %s: %w", conn.ID, statusErr))
		}
		return fmt.Errorf("token refresh: %w", err)
	}

	// Store refreshed tokens in a transaction.
	tx, owned, txErr := db.BeginOrContinue(ctx, deps.DB)
	if txErr != nil {
		return fmt.Errorf("begin tx: %w", txErr)
	}
	if owned {
		defer db.RollbackTx(ctx, tx) //nolint:errcheck // best-effort cleanup
	}

	if err := deps.Vault.DeleteSecret(ctx, tx, conn.AccessTokenVaultID); err != nil {
		logger.Warn("oauth refresh: failed to delete old access token from vault",
			append(logAttrs, "vault_id", conn.AccessTokenVaultID, "error", err)...)
	}
	newAccessVaultID, err := deps.Vault.CreateSecret(ctx, tx, conn.ID+"_access", []byte(result.AccessToken))
	if err != nil {
		return fmt.Errorf("vault create access token: %w", err)
	}

	var newRefreshVaultID *string
	if result.RefreshToken != string(refreshTokenBytes) {
		if err := deps.Vault.DeleteSecret(ctx, tx, *conn.RefreshTokenVaultID); err != nil {
			logger.Warn("oauth refresh: failed to delete old refresh token from vault",
				append(logAttrs, "vault_id", *conn.RefreshTokenVaultID, "error", err)...)
		}
		id, err := deps.Vault.CreateSecret(ctx, tx, conn.ID+"_refresh", []byte(result.RefreshToken))
		if err != nil {
			return fmt.Errorf("vault create refresh token: %w", err)
		}
		newRefreshVaultID = &id
	} else {
		newRefreshVaultID = conn.RefreshTokenVaultID
	}

	var tokenExpiry *time.Time
	if !result.Expiry.IsZero() {
		tokenExpiry = &result.Expiry
	}

	if err := db.UpdateOAuthConnectionTokens(ctx, tx, conn.ID, conn.UserID, newAccessVaultID, newRefreshVaultID, tokenExpiry); err != nil {
		return fmt.Errorf("update connection tokens: %w", err)
	}

	if owned {
		if err := db.CommitTx(ctx, tx); err != nil {
			return fmt.Errorf("commit: %w", err)
		}
	}

	return nil
}

// oauthRefreshInterval returns the refresh ticker interval from the
// OAUTH_REFRESH_INTERVAL env var, defaulting to 10 minutes. Values below
// 1 minute are rejected.
func oauthRefreshInterval(logger *slog.Logger) time.Duration {
	const minInterval = time.Minute

	if v := os.Getenv("OAUTH_REFRESH_INTERVAL"); v != "" {
		d, err := time.ParseDuration(v)
		if err == nil && d >= minInterval {
			return d
		}
		logger.Warn("invalid OAUTH_REFRESH_INTERVAL, using default 10m",
			"value", v, "error", err, "min", minInterval.String())
	}
	return 10 * time.Minute
}
