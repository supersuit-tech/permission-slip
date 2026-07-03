package db

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

// AgentApprovalSweepItem is a pending or recently resolved approval for heartbeat sweeps.
type AgentApprovalSweepItem struct {
	ApprovalID   string
	Status       string
	ExpiresAt    time.Time
	ResolvedAt   *time.Time
	DenialReason *string
}

// ListAgentApprovalsForSweep returns pending (non-expired) approvals and terminal
// approvals resolved on or after resolvedSince for the given agent.
func ListAgentApprovalsForSweep(ctx context.Context, db DBTX, agentID int64, resolvedSince time.Time) ([]AgentApprovalSweepItem, error) {
	since := TimestampForSQLite(resolvedSince.UTC())
	rows, err := db.Query(ctx,
		`SELECT approval_id, status, expires_at, approved_at, denied_at, cancelled_at, denial_reason
		 FROM approvals
		 WHERE agent_id = $1
		   AND (
		     (status = 'pending' AND datetime(expires_at) > datetime('now'))
		     OR (
		       status IN ('approved', 'denied', 'cancelled')
		       AND COALESCE(approved_at, denied_at, cancelled_at) >= $2
		     )
		     OR (
		       status = 'pending' AND datetime(expires_at) <= datetime('now') AND datetime(expires_at) >= $2
		     )
		   )
		 ORDER BY created_at ASC`,
		agentID, since,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []AgentApprovalSweepItem
	for rows.Next() {
		var item AgentApprovalSweepItem
		var expiresAt, approvedAt, deniedAt, cancelledAt sql.NullString
		var denialReason sql.NullString
		if err := rows.Scan(&item.ApprovalID, &item.Status, &expiresAt, &approvedAt, &deniedAt, &cancelledAt, &denialReason); err != nil {
			return nil, err
		}
		exp, err := sqliteTimeRequired(expiresAt)
		if err != nil {
			return nil, err
		}
		item.ExpiresAt = exp

		if item.Status == "pending" && !exp.IsZero() && exp.Before(time.Now()) {
			item.Status = "expired"
			item.ResolvedAt = &exp
		} else {
			item.ResolvedAt = coalesceTimePtr(approvedAt, deniedAt, cancelledAt)
		}
		if denialReason.Valid {
			s := denialReason.String
			item.DenialReason = &s
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// ListExpiredPendingApprovalIDs returns approval IDs that are pending past expires_at.
func ListExpiredPendingApprovalIDs(ctx context.Context, db DBTX, limit int) ([]string, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := db.Query(ctx,
		`SELECT approval_id FROM approvals
		 WHERE status = 'pending' AND datetime(expires_at) <= datetime('now')
		 ORDER BY expires_at ASC
		 LIMIT $1`,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// GetApprovalByID returns an approval by ID regardless of agent ownership.
func GetApprovalByID(ctx context.Context, db DBTX, approvalID string) (*Approval, error) {
	row := db.QueryRow(ctx,
		`SELECT `+approvalColumns+` FROM approvals WHERE approval_id = $1`,
		approvalID,
	)
	a, err := scanApproval(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return a, err
}

func coalesceTimePtr(vals ...sql.NullString) *time.Time {
	for _, v := range vals {
		if v.Valid {
			t, err := sqliteTimeRequired(v)
			if err == nil {
				return &t
			}
		}
	}
	return nil
}
