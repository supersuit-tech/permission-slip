package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// Approval represents a row from the approvals table.
type Approval struct {
	ApprovalID      string
	AgentID         int64
	ApproverID      string
	Action          []byte // raw JSONB
	Context         []byte // raw JSONB
	Status          string
	ExecutionStatus *string
	ExecutionResult []byte // raw JSONB
	ExecutedAt      *time.Time
	ExpiresAt       time.Time
	ApprovedAt      *time.Time
	DeniedAt        *time.Time
	CancelledAt     *time.Time
	CreatedAt       time.Time
}

// approvalColumns is the canonical column list for SELECT on the approvals table.
// Keep in sync with scanApproval.
const approvalColumns = `approval_id, agent_id, approver_id, action, context,
	status, execution_status, execution_result, executed_at,
	expires_at, approved_at, denied_at, cancelled_at, created_at`

// ApprovalCursor identifies the position of the last item on a page,
// using both created_at and approval_id as a compound key to avoid
// skipping rows when multiple approvals share the same created_at.
type ApprovalCursor struct {
	CreatedAt  time.Time
	ApprovalID string
}

// ApprovalPage holds a page of approvals plus a flag indicating whether more exist.
type ApprovalPage struct {
	Approvals []Approval
	HasMore   bool
}

// scanApproval scans a single row into an Approval. The row must select approvalColumns.
func scanApproval(row pgx.Row) (*Approval, error) {
	var a Approval
	err := row.Scan(
		&a.ApprovalID, &a.AgentID, &a.ApproverID, &a.Action, &a.Context,
		&a.Status, &a.ExecutionStatus, &a.ExecutionResult, &a.ExecutedAt,
		&a.ExpiresAt, &a.ApprovedAt, &a.DeniedAt, &a.CancelledAt, &a.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// scanApprovalWithMeta scans approvalColumns plus an extra agents.metadata column.
func scanApprovalWithMeta(row pgx.Row) (*Approval, []byte, error) {
	var a Approval
	var agentMeta []byte
	err := row.Scan(
		&a.ApprovalID, &a.AgentID, &a.ApproverID, &a.Action, &a.Context,
		&a.Status, &a.ExecutionStatus, &a.ExecutionResult, &a.ExecutedAt,
		&a.ExpiresAt, &a.ApprovedAt, &a.DeniedAt, &a.CancelledAt, &a.CreatedAt,
		&agentMeta,
	)
	if err != nil {
		return nil, nil, err
	}
	return &a, agentMeta, nil
}

// ListApprovalsByApproverPaginated returns approvals for the given approver with
// cursor-based pagination, ordered by creation time descending (newest first),
// with approval_id as a tiebreaker. Pass a nil cursor to start from the
// beginning. Limit is clamped to [1, 100] with a default of 50 when <= 0.
//
// When statusFilter is "pending", expired approvals are excluded. When
// statusFilter is "all", all approvals are returned regardless of status.
// Otherwise only approvals matching the given status are returned.
func ListApprovalsByApproverPaginated(ctx context.Context, db DBTX, approverID, statusFilter string, limit int, cursor *ApprovalCursor) (*ApprovalPage, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	// Fetch one extra row to determine has_more.
	fetchLimit := limit + 1

	// Build the WHERE clause dynamically to avoid duplicating the full query
	// across every (statusFilter × cursor) combination.
	args := []any{approverID}
	where := "approver_id = $1"
	nextParam := 2

	switch statusFilter {
	case "", "pending":
		where += " AND status = 'pending' AND expires_at > now()"
	case "all":
		// No additional status filter.
	default:
		where += fmt.Sprintf(" AND status = $%d", nextParam)
		args = append(args, statusFilter)
		nextParam++
	}

	if cursor != nil {
		where += fmt.Sprintf(" AND (created_at, approval_id) < ($%d, $%d)", nextParam, nextParam+1)
		args = append(args, cursor.CreatedAt, cursor.ApprovalID)
		nextParam += 2
	}

	args = append(args, fetchLimit)
	query := fmt.Sprintf(
		`SELECT %s FROM approvals WHERE %s ORDER BY created_at DESC, approval_id DESC LIMIT $%d`,
		approvalColumns, where, nextParam,
	)

	rows, err := db.Query(ctx, query, args...)
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
	if err := rows.Err(); err != nil {
		return nil, err
	}

	hasMore := len(approvals) > limit
	if hasMore {
		approvals = approvals[:limit]
	}

	return &ApprovalPage{Approvals: approvals, HasMore: hasMore}, nil
}

// ApprovalError represents a domain error from approval operations.
type ApprovalError struct {
	Code   string
	Status string // current status if relevant
}

func (e *ApprovalError) Error() string { return e.Code }

const (
	ApprovalErrNotFound        = "not_found"
	ApprovalErrAlreadyResolved = "already_resolved"
	ApprovalErrExpired         = "expired"
)

// GetApprovalByIDAndApprover returns the approval with the given ID belonging
// to the given approver, or nil if not found.
func GetApprovalByIDAndApprover(ctx context.Context, db DBTX, approvalID, approverID string) (*Approval, error) {
	row := db.QueryRow(ctx,
		`SELECT `+approvalColumns+`
		 FROM approvals
		 WHERE approval_id = $1 AND approver_id = $2`,
		approvalID, approverID,
	)
	a, err := scanApproval(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return a, nil
}

// ApproveApproval atomically sets the approval status to 'approved' and records
// the timestamp. Returns the updated approval and the agent's metadata snapshot.
func ApproveApproval(ctx context.Context, db DBTX, approvalID, approverID string) (*Approval, []byte, error) {
	return resolveApproval(ctx, db, approvalID, approverID, "approved", "approved_at")
}

// DenyApproval atomically sets the approval status to 'denied' and records the
// timestamp. Returns the updated approval and the agent's metadata snapshot.
func DenyApproval(ctx context.Context, db DBTX, approvalID, approverID string) (*Approval, []byte, error) {
	return resolveApproval(ctx, db, approvalID, approverID, "denied", "denied_at")
}

// resolveApproval is the shared implementation for ApproveApproval and
// DenyApproval. It atomically updates the approval status while enforcing
// status='pending' AND expires_at > now() to eliminate TOCTOU races. On
// failure it reads the current row to produce a precise error.
func resolveApproval(ctx context.Context, db DBTX, approvalID, approverID, newStatus, timestampCol string) (*Approval, []byte, error) {
	query := fmt.Sprintf(
		`WITH updated AS (
			UPDATE approvals
			SET status = $3, %s = now()
			WHERE approval_id = $1 AND approver_id = $2
			  AND status = 'pending' AND expires_at > now()
			RETURNING %s
		)
		SELECT updated.*, a.metadata
		FROM updated
		LEFT JOIN agents a ON a.agent_id = updated.agent_id`,
		timestampCol, approvalColumns,
	)
	row := db.QueryRow(ctx, query, approvalID, approverID, newStatus)
	appr, agentMeta, err := scanApprovalWithMeta(row)
	if err == nil {
		return appr, agentMeta, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, err
	}

	// UPDATE matched zero rows — determine why.
	return nil, nil, diagnoseApprovalFailure(ctx, db, approvalID, approverID)
}

// diagnoseApprovalFailure reads the current approval row to determine why an
// atomic UPDATE matched zero rows. Returns the appropriate ApprovalError.
func diagnoseApprovalFailure(ctx context.Context, db DBTX, approvalID, approverID string) error {
	appr, err := GetApprovalByIDAndApprover(ctx, db, approvalID, approverID)
	if err != nil {
		return err
	}
	if appr == nil {
		return &ApprovalError{Code: ApprovalErrNotFound}
	}
	if appr.Status != "pending" {
		return &ApprovalError{Code: ApprovalErrAlreadyResolved, Status: appr.Status}
	}
	// Status is pending but UPDATE didn't match → must be expired.
	return &ApprovalError{Code: ApprovalErrExpired}
}
