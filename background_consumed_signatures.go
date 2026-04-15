package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/supersuit-tech/permission-slip/api"
	"github.com/supersuit-tech/permission-slip/db"
)

func init() {
	RegisterBackgroundJob(BackgroundJob{
		Name: "cleanup consumed signatures",
		Start: func(ctx context.Context, deps *api.Deps, logger *slog.Logger) <-chan struct{} {
			if deps.DB == nil {
				return nil
			}
			return startCleanupConsumedSignatures(ctx, deps.DB, logger)
		},
	})
}

// startCleanupConsumedSignatures runs db.CleanupExpiredConsumedSignatures on a
// recurring ticker, deleting rows whose expires_at has passed. pg_cron runs
// the same DELETE on its own schedule (see migration 20260415134017); either
// is sufficient. Running both keeps the table bounded in deployments where
// pg_cron is unavailable. Returns a channel closed when the goroutine exits.
func startCleanupConsumedSignatures(ctx context.Context, pool db.DBTX, logger *slog.Logger) <-chan struct{} {
	interval := consumedSignatureCleanupInterval(logger)
	logger.Info("cleanup consumed signatures: scheduled", "interval", interval.String())

	done := make(chan struct{})
	go func() {
		defer close(done)

		// Run once immediately on startup to catch up after downtime.
		runCleanupConsumedSignatures(ctx, pool, logger)

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				logger.Info("cleanup consumed signatures: stopped")
				return
			case <-ticker.C:
				runCleanupConsumedSignatures(ctx, pool, logger)
			}
		}
	}()
	return done
}

// runCleanupConsumedSignatures executes a single cleanup pass with a 60-second timeout.
func runCleanupConsumedSignatures(ctx context.Context, pool db.DBTX, logger *slog.Logger) {
	cleanupCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	deleted, err := db.CleanupExpiredConsumedSignatures(cleanupCtx, pool)
	if err != nil {
		if ctx.Err() != nil {
			logger.Info("cleanup consumed signatures: cancelled", "error", err)
			return
		}
		logger.Error("cleanup consumed signatures: failed", "error", err)
		sentry.CaptureException(err)
	} else if deleted > 0 {
		logger.Info("cleanup consumed signatures: completed", "rows_deleted", deleted)
	} else {
		logger.Debug("cleanup consumed signatures: completed", "rows_deleted", 0)
	}
}

// consumedSignatureCleanupInterval returns the cleanup ticker interval from
// the CLEANUP_CONSUMED_SIGNATURES_INTERVAL env var, defaulting to 5 minutes.
// Values below 1 minute are rejected to prevent resource exhaustion.
func consumedSignatureCleanupInterval(logger *slog.Logger) time.Duration {
	const minInterval = time.Minute

	if v := os.Getenv("CLEANUP_CONSUMED_SIGNATURES_INTERVAL"); v != "" {
		d, err := time.ParseDuration(v)
		if err == nil && d >= minInterval {
			return d
		}
		if err != nil {
			logger.Warn("invalid CLEANUP_CONSUMED_SIGNATURES_INTERVAL, using default 5m",
				"value", v, "error", err, "min", minInterval.String())
		} else {
			logger.Warn("CLEANUP_CONSUMED_SIGNATURES_INTERVAL is below minimum, using default 5m",
				"value", v, "min", minInterval.String())
		}
	}
	return 5 * time.Minute
}
