package db

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// maxApprovalVerificationAttempts is the maximum number of failed confirmation
// code verification attempts before an approval is locked.
const maxApprovalVerificationAttempts = 5

// DefaultApprovalTTL is the default time-to-live for a new approval request.
const DefaultApprovalTTL = 10 * time.Minute

// InsertApprovalParams holds the parameters for creating a new approval.
type InsertApprovalParams struct {
	ApprovalID string
	AgentID    int64
	ApproverID string
	Action     []byte // raw JSONB
	Context    []byte // raw JSONB
	ExpiresAt  time.Time
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
		`INSERT INTO approvals (approval_id, agent_id, approver_id, action, context, status, expires_at)
		 VALUES ($1, $2, $3, $4, $5, 'pending', $6)
		 RETURNING `+approvalColumns,
		p.ApprovalID, p.AgentID, p.ApproverID, p.Action, p.Context, p.ExpiresAt,
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
	if errors.Is(err, pgx.ErrNoRows) {
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
		 SET status = 'cancelled', cancelled_at = now()
		 WHERE approval_id = $1 AND agent_id = $2
		   AND status = 'pending' AND expires_at > now()
		 RETURNING `+approvalColumns,
		approvalID, agentID,
	)
	appr, err := scanApproval(row)
	if err == nil {
		return appr, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	return nil, diagnoseAgentApprovalFailure(ctx, db, approvalID, agentID)
}

// VerifyApprovalConfirmationCode atomically increments verification_attempts
// and checks the submitted code hash against the stored hash. The approval must
// be in 'approved' status (user has already approved) and not expired.
//
// Returns the approval on success. On failure, returns the appropriate
// ApprovalError (not_found, expired, already_resolved, verification_locked,
// invalid_code).
func VerifyApprovalConfirmationCode(ctx context.Context, db DBTX, approvalID string, agentID int64, submittedCodeHash string) (*Approval, error) {
	// Atomically increment attempts for approved approvals under the limit.
	row := db.QueryRow(ctx,
		`UPDATE approvals
		 SET verification_attempts = verification_attempts + 1
		 WHERE approval_id = $1 AND agent_id = $2
		   AND status = 'approved' AND expires_at > now()
		   AND verification_attempts < $3
		 RETURNING `+approvalColumns,
		approvalID, agentID, maxApprovalVerificationAttempts,
	)
	appr, err := scanApproval(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, diagnoseApprovalVerifyFailure(ctx, db, approvalID, agentID)
	}
	if err != nil {
		return nil, err
	}

	// Constant-time comparison of hex-encoded hashes. Both strings are
	// fixed-length SHA-256/HMAC-SHA256 hex digests. The 5-attempt lockout
	// already makes timing attacks infeasible, but we use constant-time
	// comparison for defense-in-depth (matching the pattern in
	// VerifyAgentConfirmationCode).
	storedHash := ""
	if appr.ConfirmationCodeHash != nil {
		storedHash = *appr.ConfirmationCodeHash
	}
	if subtle.ConstantTimeCompare([]byte(storedHash), []byte(submittedCodeHash)) != 1 {
		return appr, &ApprovalError{Code: ApprovalErrInvalidCode}
	}

	// Success — invalidate the confirmation code so it cannot be reused, and
	// undo the attempt increment (successful verifications should not count
	// toward the lockout limit). Clearing the hash prevents repeated
	// verifications against the same approval.
	_, err = db.Exec(ctx,
		`UPDATE approvals
		 SET confirmation_code_hash = NULL,
		     verification_attempts = GREATEST(verification_attempts - 1, 0)
		 WHERE approval_id = $1`,
		approvalID,
	)
	if err != nil {
		// Log but don't fail — the verification itself succeeded.
		return appr, nil
	}

	return appr, nil
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

// diagnoseApprovalVerifyFailure reads the current approval row to determine
// why the atomic verification attempt increment matched zero rows.
func diagnoseApprovalVerifyFailure(ctx context.Context, db DBTX, approvalID string, agentID int64) error {
	appr, err := GetApprovalByIDAndAgent(ctx, db, approvalID, agentID)
	if err != nil {
		return err
	}
	if appr == nil {
		return &ApprovalError{Code: ApprovalErrNotFound}
	}
	switch appr.Status {
	case "approved":
		// Status is correct but UPDATE didn't match — could be expired or locked.
		if time.Now().After(appr.ExpiresAt) {
			return &ApprovalError{Code: ApprovalErrExpired}
		}
		if appr.VerificationAttempts >= maxApprovalVerificationAttempts {
			return &ApprovalError{Code: ApprovalErrVerificationLocked}
		}
		// Defensive fallback.
		return &ApprovalError{Code: ApprovalErrVerificationLocked}
	case "pending":
		return &ApprovalError{Code: ApprovalErrNotYetApproved}
	default:
		return &ApprovalError{Code: ApprovalErrAlreadyResolved, Status: appr.Status}
	}
}

// SetApprovalTokenJTI writes the token JTI to the approval row. This is called
// after successfully minting an action token to enable single-use enforcement.
// The WHERE clause ensures a JTI can only be set once; attempting to overwrite
// an existing JTI returns an error.
func SetApprovalTokenJTI(ctx context.Context, db DBTX, approvalID, jti string) error {
	tag, err := db.Exec(ctx,
		`UPDATE approvals SET token_jti = $2 WHERE approval_id = $1 AND token_jti IS NULL`,
		approvalID, jti,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("approval %s already has a token_jti assigned", approvalID)
	}
	return nil
}

// Additional ApprovalError codes for agent-facing operations.
const (
	ApprovalErrDuplicateRequest   = "duplicate_request"
	ApprovalErrInvalidCode        = "invalid_code"
	ApprovalErrVerificationLocked = "verification_locked"
	ApprovalErrNotYetApproved     = "not_yet_approved"
)
