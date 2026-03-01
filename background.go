package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/supersuit-tech/permission-slip-web/db"
)

// startAuditPurge runs db.PurgeExpiredAuditEvents on a recurring ticker,
// logging results each run. It returns a channel that is closed when the
// goroutine exits, so callers can wait for it before closing the DB pool.
// Cancel ctx to stop the goroutine cleanly.
func startAuditPurge(ctx context.Context, pool db.DBTX, logger *slog.Logger) <-chan struct{} {
	interval := auditPurgeInterval(logger)
	logger.Info("audit purge: scheduled", "interval", interval.String(),
		"env", "AUDIT_PURGE_INTERVAL")

	done := make(chan struct{})
	go func() {
		defer close(done)

		// Run once immediately on startup to catch up after downtime.
		runAuditPurge(ctx, pool, logger)

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				logger.Info("audit purge: stopped")
				return
			case <-ticker.C:
				runAuditPurge(ctx, pool, logger)
			}
		}
	}()
	return done
}

// runAuditPurge executes a single purge pass with a 60-second timeout.
// Context cancellation errors (e.g. during shutdown) are logged at info
// level and not reported to Sentry.
func runAuditPurge(ctx context.Context, pool db.DBTX, logger *slog.Logger) {
	purgeCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	deleted, err := db.PurgeExpiredAuditEvents(purgeCtx, pool)
	if err != nil {
		if ctx.Err() != nil {
			logger.Info("audit purge: cancelled", "error", err)
			return
		}
		logger.Error("audit purge: failed", "error", err)
		sentry.CaptureException(err)
	} else if deleted > 0 {
		logger.Info("audit purge: completed", "rows_deleted", deleted)
	} else {
		logger.Debug("audit purge: completed", "rows_deleted", 0)
	}
}

// auditPurgeInterval returns the purge ticker interval from the
// AUDIT_PURGE_INTERVAL env var, defaulting to 1 hour. Values below
// 1 minute are rejected to prevent accidental resource exhaustion.
func auditPurgeInterval(logger *slog.Logger) time.Duration {
	const minInterval = time.Minute

	if v := os.Getenv("AUDIT_PURGE_INTERVAL"); v != "" {
		d, err := time.ParseDuration(v)
		if err == nil && d >= minInterval {
			return d
		}
		logger.Warn("invalid AUDIT_PURGE_INTERVAL, using default 1h",
			"value", v, "error", err, "min", minInterval.String())
	}
	return time.Hour
}
