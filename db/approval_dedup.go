package db

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"
)

// DefaultDenialCooldown is the default window during which a recently denied
// action fingerprint is short-circuited instead of creating a new approval.
var DefaultDenialCooldown = 10 * time.Minute

// DenialCooldownFromEnv reads APPROVAL_DENIAL_COOLDOWN and returns a duration
// clamped to [1m, 24h]. Unset, unparseable, or out-of-bounds values fall back
// to DefaultDenialCooldown.
func DenialCooldownFromEnv(logger *slog.Logger) time.Duration {
	const minCooldown = time.Minute
	const maxCooldown = 24 * time.Hour

	v := os.Getenv("APPROVAL_DENIAL_COOLDOWN")
	if v == "" {
		return DefaultDenialCooldown
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		if logger != nil {
			logger.Warn("invalid APPROVAL_DENIAL_COOLDOWN, using default",
				"value", v, "error", err, "default", DefaultDenialCooldown.String())
		}
		return DefaultDenialCooldown
	}
	if d < minCooldown || d > maxCooldown {
		if logger != nil {
			logger.Warn("APPROVAL_DENIAL_COOLDOWN out of bounds, using default",
				"value", v, "min", minCooldown.String(), "max", maxCooldown.String(),
				"default", DefaultDenialCooldown.String())
		}
		return DefaultDenialCooldown
	}
	return d
}

// ComputeActionFingerprint returns a stable SHA-256 hex digest for deduplicating
// approval requests. The fingerprint covers agent, approver, and the normalized
// action JSON payload.
func ComputeActionFingerprint(agentID int64, approverID string, action []byte) string {
	h := sha256.New()
	fmt.Fprintf(h, "%d:%s:", agentID, approverID)
	h.Write(action)
	return hex.EncodeToString(h.Sum(nil))
}

// FindRecentDeniedApproval returns the most recent denied approval with the
// same fingerprint within the cooldown window, or nil if none exists.
func FindRecentDeniedApproval(ctx context.Context, d DBTX, agentID int64, approverID, fingerprint string, since time.Time) (*Approval, error) {
	if fingerprint == "" {
		return nil, nil
	}

	row := d.QueryRow(ctx,
		`SELECT `+approvalColumns+`
		 FROM approvals
		 WHERE agent_id = $1
		   AND approver_id = $2
		   AND action_fingerprint = $3
		   AND status = 'denied'
		   AND denied_at IS NOT NULL
		   AND datetime(denied_at) > datetime($4)
		 ORDER BY denied_at DESC
		 LIMIT 1`,
		agentID, approverID, fingerprint, TimestampForSQLite(since),
	)
	appr, err := scanApproval(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return appr, nil
}
