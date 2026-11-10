package api

import (
	"context"
	"log"

	"github.com/supersuit-tech/permission-slip/db"
)

// MarkOAuthConnectionNeedsReauth sets an OAuth connection to needs_reauth and
// cancels pending approvals for agents that depend on that connection.
func MarkOAuthConnectionNeedsReauth(ctx context.Context, deps *Deps, connID, userID string) error {
	if deps == nil || deps.DB == nil {
		return nil
	}
	if err := db.UpdateOAuthConnectionStatus(ctx, deps.DB, connID, userID, db.OAuthStatusNeedsReauth); err != nil {
		return err
	}
	cancelled, err := db.CancelPendingApprovalsForOAuthConnection(ctx, deps.DB, connID)
	if err != nil {
		log.Printf("[%s] cancel pending approvals for oauth connection %s: %v", TraceID(ctx), connID, err)
		CaptureError(ctx, err)
		return nil
	}
	if cancelled > 0 {
		log.Printf("[%s] cancelled %d pending approval(s) after oauth connection %s marked needs_reauth",
			TraceID(ctx), cancelled, connID)
	}
	return nil
}
