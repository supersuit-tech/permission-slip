package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// ApprovalBulkGroup represents a row from approval_bulk_groups.
type ApprovalBulkGroup struct {
	BulkGroupID string
	AgentID     int64
	ApproverID  string
	ActionType  string
	ItemCount   int
	AllowMixed  bool
	ExpiresAt   time.Time
	CreatedAt   time.Time
}

const approvalBulkGroupColumns = `bulk_group_id, agent_id, approver_id, action_type, item_count, allow_mixed, expires_at, created_at`

func scanApprovalBulkGroup(row rowScanner) (*ApprovalBulkGroup, error) {
	var g ApprovalBulkGroup
	var allowMixed int
	var expiresAt, createdAt sql.NullString
	err := row.Scan(
		&g.BulkGroupID, &g.AgentID, &g.ApproverID, &g.ActionType, &g.ItemCount,
		&allowMixed, &expiresAt, &createdAt,
	)
	if err != nil {
		return nil, err
	}
	g.AllowMixed = allowMixed != 0
	var err2 error
	g.ExpiresAt, err2 = sqliteTimeRequired(expiresAt)
	if err2 != nil {
		return nil, err2
	}
	g.CreatedAt, err2 = sqliteTimeRequired(createdAt)
	if err2 != nil {
		return nil, err2
	}
	return &g, nil
}

// InsertApprovalBulkGroupParams holds parameters for creating a bulk group row.
type InsertApprovalBulkGroupParams struct {
	BulkGroupID string
	AgentID     int64
	ApproverID  string
	ActionType  string
	ItemCount   int
	ExpiresAt   time.Time
}

// InsertApprovalBulkGroup inserts a bulk group row within the current transaction.
func InsertApprovalBulkGroup(ctx context.Context, d DBTX, p InsertApprovalBulkGroupParams) (*ApprovalBulkGroup, error) {
	row := d.QueryRow(ctx,
		`INSERT INTO approval_bulk_groups (bulk_group_id, agent_id, approver_id, action_type, item_count, expires_at)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING `+approvalBulkGroupColumns,
		p.BulkGroupID, p.AgentID, p.ApproverID, p.ActionType, p.ItemCount, TimestampForSQLite(p.ExpiresAt),
	)
	return scanApprovalBulkGroup(row)
}

// GetApprovalBulkGroupByIDAndApprover returns a bulk group owned by the approver.
func GetApprovalBulkGroupByIDAndApprover(ctx context.Context, d DBTX, bulkGroupID, approverID string) (*ApprovalBulkGroup, error) {
	row := d.QueryRow(ctx,
		`SELECT `+approvalBulkGroupColumns+`
		 FROM approval_bulk_groups
		 WHERE bulk_group_id = $1 AND approver_id = $2`,
		bulkGroupID, approverID,
	)
	g, err := scanApprovalBulkGroup(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return g, nil
}

// GetApprovalBulkGroupByIDAndAgent returns a bulk group owned by the agent.
func GetApprovalBulkGroupByIDAndAgent(ctx context.Context, d DBTX, bulkGroupID string, agentID int64) (*ApprovalBulkGroup, error) {
	row := d.QueryRow(ctx,
		`SELECT `+approvalBulkGroupColumns+`
		 FROM approval_bulk_groups
		 WHERE bulk_group_id = $1 AND agent_id = $2`,
		bulkGroupID, agentID,
	)
	g, err := scanApprovalBulkGroup(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return g, nil
}

// ListApprovalsByBulkGroupID returns all approvals in a bulk group, ordered by created_at.
func ListApprovalsByBulkGroupID(ctx context.Context, d DBTX, bulkGroupID string) ([]Approval, error) {
	rows, err := d.Query(ctx,
		`SELECT `+approvalColumns+`
		 FROM approvals
		 WHERE bulk_group_id = $1
		 ORDER BY created_at ASC, approval_id ASC`,
		bulkGroupID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var approvals []Approval
	for rows.Next() {
		a, err := scanApproval(rows)
		if err != nil {
			return nil, err
		}
		approvals = append(approvals, *a)
	}
	return approvals, rows.Err()
}

// BulkGroupAggregateStatus computes pending/partial/resolved from child approvals.
func BulkGroupAggregateStatus(approvals []Approval, now time.Time) string {
	if len(approvals) == 0 {
		return "resolved"
	}
	pendingCount := 0
	for _, a := range approvals {
		status := a.Status
		if status == "pending" && !a.ExpiresAt.IsZero() && a.ExpiresAt.Before(now) {
			status = "expired"
		}
		if status == "pending" {
			pendingCount++
		}
	}
	if pendingCount == len(approvals) {
		return "pending"
	}
	if pendingCount == 0 {
		return "resolved"
	}
	return "partial"
}

// ListDistinctPendingBulkGroupIDs returns bulk group IDs with pending non-expired
// items for the given approver, for dashboard list deduplication.
func ListDistinctPendingBulkGroupIDs(ctx context.Context, d DBTX, approverID string) ([]string, error) {
	rows, err := d.Query(ctx,
		`SELECT DISTINCT bulk_group_id
		 FROM approvals
		 WHERE approver_id = $1
		   AND bulk_group_id IS NOT NULL
		   AND status = 'pending'
		   AND datetime(expires_at) > datetime('now')`,
		approverID,
	)
	if err != nil {
		return nil, fmt.Errorf("list bulk group ids: %w", err)
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
