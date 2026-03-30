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
		Name: "scrub sensitive execution data",
		Start: func(ctx context.Context, deps *api.Deps, logger *slog.Logger) <-chan struct{} {
			if deps.DB == nil {
				return nil
			}
			return startScrubExecutionData(ctx, deps.DB, logger)
		},
	})
}

// startScrubExecutionData runs db.ScrubSensitiveExecutionData on a recurring
// ticker, scrubbing execution_result, action parameters, and standing approval
// execution parameters older than 30 minutes. Returns a channel that is closed
// when the goroutine exits for clean shutdown coordination.
func startScrubExecutionData(ctx context.Context, pool db.DBTX, logger *slog.Logger) <-chan struct{} {
	interval := scrubInterval(logger)
	logger.Info("scrub execution data: scheduled", "interval", interval.String())

	done := make(chan struct{})
	go func() {
		defer close(done)

		// Run once immediately on startup to catch up after downtime.
		runScrubExecutionData(ctx, pool, logger)

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				logger.Info("scrub execution data: stopped")
				return
			case <-ticker.C:
				runScrubExecutionData(ctx, pool, logger)
			}
		}
	}()
	return done
}

// runScrubExecutionData executes a single scrub pass with a 60-second timeout.
func runScrubExecutionData(ctx context.Context, pool db.DBTX, logger *slog.Logger) {
	scrubCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	scrubbed, err := db.ScrubSensitiveExecutionData(scrubCtx, pool)
	if err != nil {
		if ctx.Err() != nil {
			logger.Info("scrub execution data: cancelled", "error", err)
			return
		}
		logger.Error("scrub execution data: failed", "error", err)
		sentry.CaptureException(err)
	} else if scrubbed > 0 {
		logger.Info("scrub execution data: completed", "rows_scrubbed", scrubbed)
	} else {
		logger.Debug("scrub execution data: completed", "rows_scrubbed", 0)
	}
}

// scrubInterval returns the scrub ticker interval from the
// SCRUB_EXECUTION_DATA_INTERVAL env var, defaulting to 5 minutes.
// Values below 1 minute are rejected to prevent resource exhaustion.
func scrubInterval(logger *slog.Logger) time.Duration {
	const minInterval = time.Minute

	if v := os.Getenv("SCRUB_EXECUTION_DATA_INTERVAL"); v != "" {
		d, err := time.ParseDuration(v)
		if err == nil && d >= minInterval {
			return d
		}
		if err != nil {
			logger.Warn("invalid SCRUB_EXECUTION_DATA_INTERVAL, using default 5m",
				"value", v, "error", err, "min", minInterval.String())
		} else {
			logger.Warn("SCRUB_EXECUTION_DATA_INTERVAL is below minimum, using default 5m",
				"value", v, "min", minInterval.String())
		}
	}
	return 5 * time.Minute
}
