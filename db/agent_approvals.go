package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"
)

// DefaultApprovalTTL is the default time-to-live for a new approval request
// when the agent does not supply an `expires_in` override. It is a var rather
// than a const so it can be reassigned at process startup from the
// APPROVAL_REQUEST_TTL env var via ApprovalTTLFromEnv.
var DefaultApprovalTTL = 10 * time.Minute

// ApprovalTTLFromEnv reads APPROVAL_REQUEST_TTL and returns a duration clamped
// to [1m, 24h]. Unset, unparseable, or out-of-bounds values fall back to 10m
// and emit a warning via the supplied logger.
func ApprovalTTLFromEnv(logger *slog.Logger) time.Duration {
	const def = 10 * time.Minute
	const minTTL = time.Minute
	const maxTTL = 24 * time.Hour

	v := os.Getenv("APPROVAL_REQUEST_TTL")
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		if logger != nil {
			logger.Warn("invalid APPROVAL_REQUEST_TTL, using default 10m",
				"value", v, "error", err)
		}
		return def
	}
	if d < minTTL || d > maxTTL {
		if logger != nil {
			logger.Warn("APPROVAL_REQUEST_TTL out of bounds, using default 10m",
				"value", v, "min", minTTL.String(), "max", maxTTL.String())
		}
		return def
	}
	return d
}

// InsertApprovalParams holds the parameters for creating a new approval.
type InsertApprovalParams struct {
	ApprovalID      string
	AgentID         int64
	ApproverID      string
	Action          []byte // raw JSONB
	Context         []byte // raw JSONB
	ResourceDetails []byte // raw JSONB — human-readable resource metadata (optional)
	ExpiresAt       time.Time
}

// InsertApproval creates a new pending approval row. It also inserts the
// request_id into the request_ids table for idempotency deduplication.
// Both inserts run in the same transaction so that a request_id is never
// consumed without a corresponding approval row.
// Returns the newly created approval.
func InsertApproval(ctx context.Context, d DBTX, p InsertApprovalParams, requestID string) (*Approval, error) {
	tx, owned, err := BeginOrContinue(ctx, d)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	if owned {
		defer func() { _ = RollbackTx(ctx, tx) }()
	}

	// Insert request_id for idempotency. If it already exists, return a
	// specific error so the handler can return the existing approval.
	_, err = tx.Exec(ctx,
		`INSERT INTO request_ids (request_id, agent_id, approver_id) VALUES ($1, $2, $3)`,
		requestID, p.AgentID, p.ApproverID,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, &ApprovalError{Code: ApprovalErrDuplicateRequest}
		}
		return nil, fmt.Errorf("insert request_id: %w", err)
	}

	row := tx.QueryRow(ctx,
		`INSERT INTO approvals (approval_id, agent_id, approver_id, action, context, resource_details, status, expires_at)
		 VALUES ($1, $2, $3, $4, $5, $6, 'pending', $7)
		 RETURNING `+approvalColumns,
		p.ApprovalID, p.AgentID, p.ApproverID, p.Action, p.Context, p.ResourceDetails, TimestampForSQLite(p.ExpiresAt),
	)
	appr, err := scanApproval(row)
	if err != nil {
		return nil, fmt.Errorf("insert approval: %w", err)
	}

	if owned {
		if err := CommitTx(ctx, tx); err != nil {
			return nil, fmt.Errorf("commit: %w", err)
		}
	}
	return appr, nil
}

// GetApprovalByIDAndAgent returns the approval with the given ID belonging to
// the given agent, or nil if not found. Used by agent-facing endpoints where
// the agent can only access its own approvals.
func GetApprovalByIDAndAgent(ctx context.Context, db DBTX, approvalID string, agentID int64) (*Approval, error) {
	row := db.QueryRow(ctx,
		`SELECT `+approvalColumns+`
		 FROM approvals
		 WHERE approval_id = $1 AND agent_id = $2`,
		approvalID, agentID,
	)
	a, err := scanApproval(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return a, nil
}

// CancelApproval atomically sets an approval to 'cancelled' if it is currently
// pending and not expired. Only the owning agent can cancel. Returns the updated
// approval on success.
func CancelApproval(ctx context.Context, db DBTX, approvalID string, agentID int64) (*Approval, error) {
	row := db.QueryRow(ctx,
		`UPDATE approvals
		 SET status = 'cancelled', cancelled_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
		 WHERE approval_id = $1 AND agent_id = $2
		   AND status = 'pending' AND datetime(expires_at) > datetime('now')
		 RETURNING `+approvalColumns,
		approvalID, agentID,
	)
	appr, err := scanApproval(row)
	if err == nil {
		return appr, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	return nil, diagnoseAgentApprovalFailure(ctx, db, approvalID, agentID)
}

// diagnoseAgentApprovalFailure reads the current approval row to determine why
// an atomic UPDATE from the agent's perspective matched zero rows.
func diagnoseAgentApprovalFailure(ctx context.Context, db DBTX, approvalID string, agentID int64) error {
	appr, err := GetApprovalByIDAndAgent(ctx, db, approvalID, agentID)
	if err != nil {
		return err
	}
	if appr == nil {
		return &ApprovalError{Code: ApprovalErrNotFound}
	}
	if appr.Status != "pending" {
		return &ApprovalError{Code: ApprovalErrAlreadyResolved, Status: appr.Status}
	}
	return &ApprovalError{Code: ApprovalErrExpired}
}

// Additional ApprovalError codes for agent-facing operations.
const (
	ApprovalErrDuplicateRequest = "duplicate_request"
)
