package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/supersuit-tech/permission-slip/api"
	"github.com/supersuit-tech/permission-slip/db"
	"github.com/supersuit-tech/permission-slip/oauth"
	"github.com/supersuit-tech/permission-slip/vault"
)

func init() {
	RegisterBackgroundJob(BackgroundJob{
		Name: "oauth refresh",
		Start: func(ctx context.Context, deps *api.Deps, logger *slog.Logger) <-chan struct{} {
			if deps.DB == nil || deps.Vault == nil || deps.OAuthProviders == nil {
				return nil
			}
			return startOAuthRefresh(ctx, OAuthRefreshDeps{
				DB:       deps.DB,
				Vault:    deps.Vault,
				Registry: deps.OAuthProviders,
			}, logger)
		},
	})
}

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
	lastKnownAccessVaultID := conn.AccessTokenVaultID
	lastKnownTokenExpiry := conn.TokenExpiry

	provider, ok := deps.Registry.Get(conn.Provider)
	if !ok {
		return fmt.Errorf("provider %q not found in registry", conn.Provider)
	}

	// Create a scoped logger with connection-level attributes so all log lines
	// from this function include the connection context automatically.
	connLog := logger.With(
		"connection_id", conn.ID,
		"provider", conn.Provider,
		"user_id", conn.UserID,
	)

	if conn.RefreshTokenVaultID == nil {
		fresh, skipFlip, err := db.ReloadOAuthConnectionIfConcurrentRefreshSucceeded(ctx, deps.DB, conn.ID, conn.UserID, lastKnownAccessVaultID, lastKnownTokenExpiry)
		if err != nil {
			return err
		}
		if skipFlip && fresh != nil && fresh.Status == db.OAuthStatusActive && fresh.RefreshTokenVaultID != nil {
			conn = *fresh
			lastKnownAccessVaultID = conn.AccessTokenVaultID
			lastKnownTokenExpiry = conn.TokenExpiry
		} else {
			if err := db.UpdateOAuthConnectionStatus(ctx, deps.DB, conn.ID, conn.UserID, db.OAuthStatusNeedsReauth); err != nil {
				connLog.Error("oauth refresh: failed to mark connection as needs_reauth", "error", err)
				sentry.CaptureException(fmt.Errorf("oauth refresh status update failed for %s: %w", conn.ID, err))
			}
			return fmt.Errorf("no refresh token for connection %s", conn.ID)
		}
	}

	refreshTokenBytes, err := deps.Vault.ReadSecret(ctx, deps.DB, *conn.RefreshTokenVaultID)
	if err != nil {
		return fmt.Errorf("read refresh token: %w", err)
	}

	result, err := oauth.RefreshTokens(ctx, provider, string(refreshTokenBytes))
	if err != nil {
		fresh, skipFlip, reloadErr := db.ReloadOAuthConnectionIfConcurrentRefreshSucceeded(ctx, deps.DB, conn.ID, conn.UserID, lastKnownAccessVaultID, lastKnownTokenExpiry)
		if reloadErr != nil {
			return reloadErr
		}
		if skipFlip && fresh != nil && fresh.Status == db.OAuthStatusActive {
			return nil
		}
		if statusErr := db.UpdateOAuthConnectionStatus(ctx, deps.DB, conn.ID, conn.UserID, db.OAuthStatusNeedsReauth); statusErr != nil {
			connLog.Error("oauth refresh: failed to mark connection as needs_reauth", "error", statusErr)
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

	vaultOps := oauth.VaultOperations{
		DeleteSecret: func(ctx context.Context, id string) error { return deps.Vault.DeleteSecret(ctx, tx, id) },
		CreateSecret: func(ctx context.Context, name string, secret []byte) (string, error) {
			return deps.Vault.CreateSecret(ctx, tx, name, secret)
		},
	}

	_, err = oauth.StoreRefreshedTokens(ctx, vaultOps,
		oauth.StoreRefreshedTokensParams{
			ConnID:            conn.ID,
			OldAccessVaultID:  conn.AccessTokenVaultID,
			OldRefreshVaultID: conn.RefreshTokenVaultID,
			Result:            result,
			OldRefreshToken:   string(refreshTokenBytes),
		},
		func(what, vaultID string, delErr error) {
			connLog.Warn("oauth refresh: failed to delete old "+what+" from vault",
				"vault_id", vaultID, "error", delErr)
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
