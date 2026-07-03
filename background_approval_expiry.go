package main

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/supersuit-tech/permission-slip/api"
	"github.com/supersuit-tech/permission-slip/db"
)

func init() {
	RegisterBackgroundJob(BackgroundJob{
		Name: "approval expiry wake",
		Start: func(ctx context.Context, deps *api.Deps, logger *slog.Logger) <-chan struct{} {
			if deps.DB == nil {
				return nil
			}
			return startApprovalExpiryWake(ctx, deps, logger)
		},
	})
}

var expiryWakeDispatched sync.Map // approval_id → time.Time

func startApprovalExpiryWake(ctx context.Context, deps *api.Deps, logger *slog.Logger) <-chan struct{} {
	interval := 30 * time.Second
	logger.Info("approval expiry wake: scheduled", "interval", interval.String())

	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		runApprovalExpiryWake(ctx, deps, logger)

		for {
			select {
			case <-ctx.Done():
				logger.Info("approval expiry wake: stopped")
				return
			case <-ticker.C:
				runApprovalExpiryWake(ctx, deps, logger)
			}
		}
	}()
	return done
}

func runApprovalExpiryWake(ctx context.Context, deps *api.Deps, logger *slog.Logger) {
	runCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	ids, err := db.ListExpiredPendingApprovalIDs(runCtx, deps.DB, 50)
	if err != nil {
		if ctx.Err() != nil {
			return
		}
		logger.Error("approval expiry wake: list failed", "error", err)
		return
	}
	for _, id := range ids {
		if _, loaded := expiryWakeDispatched.LoadOrStore(id, time.Now()); loaded {
			continue
		}
		appr, err := db.GetApprovalByID(runCtx, deps.DB, id)
		if err != nil || appr == nil {
			continue
		}
		api.NotifyAgentApprovalResolvedSync(deps, appr)
		logger.Debug("approval expiry wake: dispatched", "approval_id", id)
	}
}
