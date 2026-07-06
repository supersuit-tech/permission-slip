package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// StandingApproval represents a row from the standing_approvals table.
type StandingApproval struct {
	StandingApprovalID  string
	AgentID             int64
	UserID              string
	ActionType          string
	ActionVersion       string
	Constraints         []byte // raw JSONB
	Name                *string
	Description         *string
	ConnectorInstanceID *string // nil = type-wide (legacy); non-nil = instance-scoped
	Status              string
	StartsAt            time.Time
	ExpiresAt           *time.Time // nil means no expiry (until revoked)
	CreatedAt           time.Time
	RevokedAt           *time.Time
	ExpiredAt           *time.Time
}

// standingApprovalColumns is the canonical column list for SELECT on the standing_approvals table.
// Keep in sync with scanStandingApproval.
const standingApprovalColumns = `standing_approval_id, agent_id, user_id, action_type, action_version,
	constraints, name, description, connector_instance_id, status,
	starts_at, expires_at, created_at, revoked_at, expired_at`

// WildcardActionType is the reserved action_type value that means
// "all actions on this connector".
const WildcardActionType = "*"

// MaxStandingApprovalListSize is the maximum number of standing approvals returned per page.
const MaxStandingApprovalListSize = 100

// DefaultStandingApprovalLimit is the default page size when no limit is specified.
const DefaultStandingApprovalLimit = 50

// StandingApprovalCursor identifies the position of the last item on a page,
// using both created_at and standing_approval_id as a compound key to avoid
// skipping rows when multiple approvals share the same created_at.
type StandingApprovalCursor struct {
	CreatedAt          time.Time
	StandingApprovalID string
}

// StandingApprovalPage holds a page of standing approvals plus a flag indicating whether more exist.
type StandingApprovalPage struct {
	Approvals []StandingApproval
	HasMore   bool
}

// scanStandingApproval scans a single row into a StandingApproval. The row must select standingApprovalColumns.
func scanStandingApproval(row rowScanner) (*StandingApproval, error) {
	var sa StandingApproval
	var startsAt, expiresAt, createdAt, revokedAt, expiredAt sql.NullString
	err := row.Scan(
		&sa.StandingApprovalID, &sa.AgentID, &sa.UserID, &sa.ActionType, &sa.ActionVersion,
		&sa.Constraints, &sa.Name, &sa.Description, &sa.ConnectorInstanceID, &sa.Status,
		&startsAt, &expiresAt, &createdAt, &revokedAt, &expiredAt,
	)
	if err != nil {
		return nil, err
	}
	var err2 error
	sa.StartsAt, err2 = sqliteTimeRequired(startsAt)
	if err2 != nil {
		return nil, err2
	}
	sa.ExpiresAt, err2 = sqliteTimePtr(expiresAt)
	if err2 != nil {
		return nil, err2
	}
	sa.CreatedAt, err2 = sqliteTimeRequired(createdAt)
	if err2 != nil {
		return nil, err2
	}
	sa.RevokedAt, err2 = sqliteTimePtr(revokedAt)
	if err2 != nil {
		return nil, err2
	}
	sa.ExpiredAt, err2 = sqliteTimePtr(expiredAt)
	if err2 != nil {
		return nil, err2
	}
	return &sa, nil
}

// CountActiveStandingApprovalsByUser returns the number of standing approvals
// that are currently active for the given user. An approval counts as active
// if its status is 'active' and either has no expiry (expires_at IS NULL) or
// has not yet expired (expires_at > strftime('%Y-%m-%dT%H:%M:%fZ', 'now')).
// This excludes approvals that have technically expired but whose status
// hasn't yet been updated by the cleanup job, so users aren't penalized
// by stale data.
//
// Note: starts_at is intentionally not checked here. Future-dated approvals
// (starts_at > strftime('%Y-%m-%dT%H:%M:%fZ', 'now')) still count toward the plan limit since the user
// created them deliberately — otherwise users could bypass limits by
// scheduling approvals far in the future.
func CountActiveStandingApprovalsByUser(ctx context.Context, db DBTX, userID string) (int, error) {
	var count int
	err := db.QueryRow(ctx,
		`SELECT COUNT(*) FROM standing_approvals
		 WHERE user_id = $1 AND status = 'active' AND (expires_at IS NULL OR datetime(expires_at) > datetime('now'))`,
		userID,
	).Scan(&count)
	return count, err
}

// StandingApprovalError represents a domain error from standing approval operations.
type StandingApprovalError struct {
	Code   string
	Status string // current status if relevant
}

func (e *StandingApprovalError) Error() string { return e.Code }

const (
	StandingApprovalErrNotFound         = "not_found"
	StandingApprovalErrAlreadyRevoked   = "already_revoked"
	StandingApprovalErrNotActive        = "not_active"
	StandingApprovalErrAgentNotFound    = "agent_not_found"
	StandingApprovalErrDuplicateRequest = "duplicate_request"
)

// CreateStandingApprovalParams holds the parameters for creating a standing approval.
type CreateStandingApprovalParams struct {
	StandingApprovalID  string
	AgentID             int64
	UserID              string
	ActionType          string
	ActionVersion       string
	Constraints         []byte // raw JSONB, may be nil
	Name                *string
	Description         *string
	ConnectorInstanceID *string
	StartsAt            time.Time
	ExpiresAt           *time.Time // nil means no expiry (until revoked)
}

// CreateStandingApproval inserts a new standing approval with status 'active'.
// The INSERT is guarded by an agent ownership check: if the agent does not
// belong to the user, no row is inserted and StandingApprovalErrAgentNotFound
// is returned.
func CreateStandingApproval(ctx context.Context, db DBTX, p CreateStandingApprovalParams) (*StandingApproval, error) {
	row := db.QueryRow(ctx,
		`WITH agent_check AS (
			SELECT 1 FROM agents WHERE agent_id = $2 AND approver_id = $3
		)
		INSERT INTO standing_approvals
		   (standing_approval_id, agent_id, user_id, action_type, action_version, constraints, name, description, connector_instance_id, status, starts_at, expires_at)
		 SELECT $1, $2, $3, $4, $5, $6, $7, $8, $9, 'active', $10, $11
		 WHERE EXISTS (SELECT 1 FROM agent_check)
		 RETURNING `+standingApprovalColumns,
		p.StandingApprovalID, p.AgentID, p.UserID, p.ActionType, p.ActionVersion,
		p.Constraints, p.Name, p.Description, p.ConnectorInstanceID,
		TimestampForSQLite(p.StartsAt), NullableTimestampForSQLite(p.ExpiresAt),
	)
	sa, err := scanStandingApproval(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, &StandingApprovalError{Code: StandingApprovalErrAgentNotFound}
		}
		return nil, err
	}
	return sa, nil
}

// ListStandingApprovalsByUser returns standing approvals for the given user
// with cursor-based pagination, ordered by creation time descending (newest
// first), with standing_approval_id as a tiebreaker. Pass a nil cursor to
// start from the beginning. Limit is clamped to [1, 100] with a default of 50
// when <= 0.
//
// If statusFilter is "active" (or empty), only active standing approvals are
// returned. Pass "all" to include all statuses.
func ListStandingApprovalsByUser(ctx context.Context, db DBTX, userID, statusFilter string, limit int, cursor *StandingApprovalCursor) (*StandingApprovalPage, error) {
	if limit <= 0 {
		limit = DefaultStandingApprovalLimit
	}
	if limit > MaxStandingApprovalListSize {
		limit = MaxStandingApprovalListSize
	}

	// Fetch one extra row to determine has_more.
	fetchLimit := limit + 1

	b := &queryBuilder{}
	b.addArg(userID) // $1

	where := []string{"user_id = $1"}

	switch statusFilter {
	case "", "active":
		where = append(where, "status = 'active'")
	case "all":
		// no status filter
	default:
		p := b.addArg(statusFilter)
		where = append(where, "status = "+p)
	}

	if cursor != nil {
		tsPlaceholder := b.addArg(TimestampForSQLite(cursor.CreatedAt))
		idPlaceholder := b.addArg(cursor.StandingApprovalID)
		where = append(where, fmt.Sprintf("(created_at, standing_approval_id) < (%s, %s)", tsPlaceholder, idPlaceholder))
	}

	limitPlaceholder := b.addArg(fetchLimit)

	query := fmt.Sprintf(
		`SELECT %s
		 FROM standing_approvals
		 WHERE %s
		 ORDER BY created_at DESC, standing_approval_id DESC
		 LIMIT %s`,
		standingApprovalColumns,
		strings.Join(where, " AND "),
		limitPlaceholder,
	)

	rows, err := db.Query(ctx, query, b.args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var approvals []StandingApproval
	for rows.Next() {
		sa, err := scanStandingApproval(rows)
		if err != nil {
			return nil, err
		}
		approvals = append(approvals, *sa)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	hasMore := len(approvals) > limit
	if hasMore {
		approvals = approvals[:limit]
	}

	return &StandingApprovalPage{Approvals: approvals, HasMore: hasMore}, nil
}

// ListStandingApprovalsByAgent returns standing approvals for the given agent,
// ordered by creation time descending (newest first). Only active standing
// approvals are returned (agents only need to see what they can currently use).
// Results are paginated using cursor-based pagination.
// Limit is clamped to [1, 100] with a default of 50 when <= 0.
func ListStandingApprovalsByAgent(ctx context.Context, db DBTX, agentID int64, limit int, cursor *StandingApprovalCursor) (*StandingApprovalPage, error) {
	if limit <= 0 {
		limit = DefaultStandingApprovalLimit
	}
	if limit > MaxStandingApprovalListSize {
		limit = MaxStandingApprovalListSize
	}

	var rows *sql.Rows
	var err error

	fetchLimit := limit + 1 // fetch one extra to detect has_more

	if cursor != nil {
		rows, err = db.Query(ctx,
			`SELECT `+standingApprovalColumns+`
			 FROM standing_approvals
			 WHERE agent_id = $1 AND status = 'active'
			   AND (created_at, standing_approval_id) < ($2, $3)
			 ORDER BY created_at DESC, standing_approval_id DESC
			 LIMIT $4`,
			agentID, TimestampForSQLite(cursor.CreatedAt), cursor.StandingApprovalID, fetchLimit,
		)
	} else {
		rows, err = db.Query(ctx,
			`SELECT `+standingApprovalColumns+`
			 FROM standing_approvals
			 WHERE agent_id = $1 AND status = 'active'
			 ORDER BY created_at DESC, standing_approval_id DESC
			 LIMIT $2`,
			agentID, fetchLimit,
		)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var approvals []StandingApproval
	for rows.Next() {
		sa, err := scanStandingApproval(rows)
		if err != nil {
			return nil, err
		}
		approvals = append(approvals, *sa)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	hasMore := len(approvals) > limit
	if hasMore {
		approvals = approvals[:limit]
	}

	return &StandingApprovalPage{Approvals: approvals, HasMore: hasMore}, nil
}

// GetStandingApprovalByIDAndUser returns the standing approval with the given ID
// belonging to the given user, or nil if not found.
func GetStandingApprovalByIDAndUser(ctx context.Context, db DBTX, saID, userID string) (*StandingApproval, error) {
	row := db.QueryRow(ctx,
		`SELECT `+standingApprovalColumns+`
		 FROM standing_approvals
		 WHERE standing_approval_id = $1 AND user_id = $2`,
		saID, userID,
	)
	sa, err := scanStandingApproval(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return sa, nil
}

// RevokeStandingApproval atomically sets the standing approval status to 'revoked'
// and records the timestamp. The UPDATE enforces status='active' to eliminate TOCTOU
// races. On failure it reads the current row to produce a precise error.
func RevokeStandingApproval(ctx context.Context, db DBTX, saID, userID string) (*StandingApproval, error) {
	row := db.QueryRow(ctx,
		`UPDATE standing_approvals
		 SET status = 'revoked', revoked_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
		 WHERE standing_approval_id = $1 AND user_id = $2
		   AND status = 'active'
		 RETURNING `+standingApprovalColumns,
		saID, userID,
	)
	updated, err := scanStandingApproval(row)
	if err == nil {
		return updated, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	// UPDATE matched zero rows — determine why.
	return nil, diagnoseStandingApprovalFailure(ctx, db, saID, userID)
}

// StandingApprovalExecution represents a single recorded execution of a standing approval.
// AgentID, UserID, ActionType, and AgentMeta are derived from related rows
// (not stored on the executions table) and populated via JOIN in queries.
type StandingApprovalExecution struct {
	ExecutionID         int64
	StandingApprovalID  string
	AgentID             int64
	UserID              string
	ActionType          string
	ConnectorInstanceID *string // from standing_approvals.connector_instance_id; nil = type-wide
	AgentMeta           []byte  // raw JSONB from agents.metadata, may be nil
	Parameters          []byte  // raw JSONB, may be nil
	ExecutedAt          time.Time
}

// RecordStandingApprovalExecution inserts an execution record after locking the
// parent standing approval row. The lock enforces user_id and status='active'
// to prevent unauthorized or stale executions.
// Returns a domain error via diagnoseStandingApprovalFailure if no matching row.
func RecordStandingApprovalExecution(ctx context.Context, db DBTX, standingApprovalID string, userID string, parameters []byte) (*StandingApprovalExecution, error) {
	var e StandingApprovalExecution
	var connectorInst sql.NullString
	err := db.QueryRow(ctx, `
		SELECT standing_approval_id, agent_id, user_id, action_type, connector_instance_id
		FROM standing_approvals
		WHERE standing_approval_id = $1 AND user_id = $2 AND status = 'active'
		  AND (expires_at IS NULL OR datetime(expires_at) > datetime('now'))`,
		standingApprovalID, userID,
	).Scan(&e.StandingApprovalID, &e.AgentID, &e.UserID, &e.ActionType, &connectorInst)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, diagnoseStandingApprovalFailure(ctx, db, standingApprovalID, userID)
	}
	if err != nil {
		return nil, err
	}
	if connectorInst.Valid {
		s := connectorInst.String
		e.ConnectorInstanceID = &s
	}

	var executedAt sql.NullString
	err = db.QueryRow(ctx, `
		INSERT INTO standing_approval_executions (standing_approval_id, parameters)
		VALUES ($1, $2)
		RETURNING id, parameters, executed_at`,
		standingApprovalID, parameters,
	).Scan(&e.ExecutionID, &e.Parameters, &executedAt)
	if err != nil {
		return nil, err
	}
	e.ExecutedAt, err = sqliteTimeRequired(executedAt)
	if err != nil {
		return nil, err
	}

	err = db.QueryRow(ctx, `SELECT metadata FROM agents WHERE agent_id = $1`, e.AgentID).Scan(&e.AgentMeta)
	if errors.Is(err, sql.ErrNoRows) {
		e.AgentMeta = nil
	} else if err != nil {
		return nil, err
	}
	return &e, nil
}

// diagnoseStandingApprovalFailure reads the current standing approval row to
// determine why an atomic UPDATE matched zero rows.
func diagnoseStandingApprovalFailure(ctx context.Context, db DBTX, saID, userID string) error {
	sa, err := GetStandingApprovalByIDAndUser(ctx, db, saID, userID)
	if err != nil {
		return err
	}
	if sa == nil {
		return &StandingApprovalError{Code: StandingApprovalErrNotFound}
	}
	if sa.Status == "revoked" {
		return &StandingApprovalError{Code: StandingApprovalErrAlreadyRevoked, Status: sa.Status}
	}
	return &StandingApprovalError{Code: StandingApprovalErrNotActive, Status: sa.Status}
}

// UpdateStandingApprovalParams holds the fields that can be updated on an active standing approval.
type UpdateStandingApprovalParams struct {
	StandingApprovalID     string
	UserID                 string
	Constraints            []byte // raw JSONB
	Name                   *string
	Description            *string
	NameSet                bool
	DescriptionSet         bool
	ExpiresAt              *time.Time // nil means no expiry (until revoked)
	ConnectorInstanceID    *string    // nil = all accounts when ConnectorInstanceIDSet is true
	ConnectorInstanceIDSet bool       // when false, connector_instance_id is left unchanged
}

// UpdateStandingApproval updates the constraints and expires_at of an active
// standing approval belonging to the given user. Returns the updated approval, or a domain error.
func UpdateStandingApproval(ctx context.Context, db DBTX, p UpdateStandingApprovalParams) (*StandingApproval, error) {
	query := `UPDATE standing_approvals
		 SET constraints = $3, expires_at = $4`
	args := []any{p.StandingApprovalID, p.UserID, p.Constraints, NullableTimestampForSQLite(p.ExpiresAt)}
	argIdx := 5
	if p.NameSet {
		query += fmt.Sprintf(`, name = $%d`, argIdx)
		args = append(args, p.Name)
		argIdx++
	}
	if p.DescriptionSet {
		query += fmt.Sprintf(`, description = $%d`, argIdx)
		args = append(args, p.Description)
		argIdx++
	}
	if p.ConnectorInstanceIDSet {
		query += fmt.Sprintf(`, connector_instance_id = $%d`, argIdx)
		args = append(args, p.ConnectorInstanceID)
	}
	query += `
		 WHERE standing_approval_id = $1 AND user_id = $2
		   AND status = 'active'
		 RETURNING ` + standingApprovalColumns

	row := db.QueryRow(ctx, query, args...)
	updated, err := scanStandingApproval(row)
	if err == nil {
		return updated, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	sa, err := GetStandingApprovalByIDAndUser(ctx, db, p.StandingApprovalID, p.UserID)
	if err != nil {
		return nil, err
	}
	if sa == nil {
		return nil, &StandingApprovalError{Code: StandingApprovalErrNotFound}
	}
	if sa.Status == "revoked" {
		return nil, &StandingApprovalError{Code: StandingApprovalErrAlreadyRevoked, Status: sa.Status}
	}
	return nil, &StandingApprovalError{Code: StandingApprovalErrNotActive, Status: sa.Status}
}

// FindActiveStandingApprovalsForAgent returns all active standing approvals for
// the given agent and action type, ordered by most recently created first.
// Exact action_type matches are returned before wildcard ("*") matches.
// If connectorInstanceID is non-empty, only rows with NULL connector_instance_id
// (type-wide) or matching connector_instance_id are included.
// Returns an empty slice if no match is found.
func FindActiveStandingApprovalsForAgent(ctx context.Context, db DBTX, agentID int64, actionType string, connectorInstanceID string) ([]*StandingApproval, error) {
	instanceFilter := `1=1`
	args := []any{agentID, actionType}
	if connectorInstanceID != "" {
		instanceFilter = `(connector_instance_id IS NULL OR connector_instance_id = $3)`
		args = append(args, connectorInstanceID)
	}

	rows, err := db.Query(ctx,
		`SELECT `+standingApprovalColumns+`
		 FROM (
		   SELECT `+standingApprovalColumns+`, 1 AS priority FROM standing_approvals
		   WHERE agent_id = $1 AND action_type = $2 AND status = 'active'
		     AND datetime(starts_at) <= datetime('now') AND (expires_at IS NULL OR datetime(expires_at) > datetime('now'))
		     AND `+instanceFilter+`
		   UNION ALL
		   SELECT `+standingApprovalColumns+`, 2 AS priority FROM standing_approvals
		   WHERE agent_id = $1 AND action_type = '*' AND action_type != $2 AND status = 'active'
		     AND datetime(starts_at) <= datetime('now') AND (expires_at IS NULL OR datetime(expires_at) > datetime('now'))
		     AND `+instanceFilter+`
		 ) combined
		 ORDER BY priority, created_at DESC, standing_approval_id DESC
		 LIMIT 100`,
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var approvals []*StandingApproval
	for rows.Next() {
		sa, err := scanStandingApproval(rows)
		if err != nil {
			return nil, err
		}
		approvals = append(approvals, sa)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return approvals, nil
}

// RecordStandingApprovalExecutionByAgent inserts an execution record after locking
// the standing approval row, scoped by agent_id. This is used by the auto-approval
// logic in POST /approvals/request where authentication is via agent signature
// rather than user session.
//
// The requestID is stored in the execution record and enforced via a unique
// index on (standing_approval_id, request_id) for idempotency. A duplicate
// request_id returns StandingApprovalErrDuplicateRequest.
func RecordStandingApprovalExecutionByAgent(ctx context.Context, db DBTX, standingApprovalID string, agentID int64, requestID string, parameters []byte) (*StandingApprovalExecution, error) {
	var e StandingApprovalExecution
	var connectorInst sql.NullString
	err := db.QueryRow(ctx, `
		SELECT standing_approval_id, agent_id, user_id, action_type, connector_instance_id
		FROM standing_approvals
		WHERE standing_approval_id = $1
		  AND agent_id = $2
		  AND status = 'active'
		  AND datetime(starts_at) <= datetime('now')
		  AND (expires_at IS NULL OR datetime(expires_at) > datetime('now'))`,
		standingApprovalID, agentID,
	).Scan(&e.StandingApprovalID, &e.AgentID, &e.UserID, &e.ActionType, &connectorInst)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, &StandingApprovalError{Code: StandingApprovalErrNotActive}
	}
	if err != nil {
		return nil, err
	}
	if connectorInst.Valid {
		s := connectorInst.String
		e.ConnectorInstanceID = &s
	}

	var executedAt sql.NullString
	err = db.QueryRow(ctx, `
		INSERT INTO standing_approval_executions (standing_approval_id, parameters, request_id)
		VALUES ($1, $2, $3)
		RETURNING id, parameters, executed_at`,
		standingApprovalID, parameters, requestID,
	).Scan(&e.ExecutionID, &e.Parameters, &executedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, &StandingApprovalError{Code: StandingApprovalErrDuplicateRequest}
		}
		return nil, err
	}
	e.ExecutedAt, err = sqliteTimeRequired(executedAt)
	if err != nil {
		return nil, err
	}

	err = db.QueryRow(ctx, `SELECT metadata FROM agents WHERE agent_id = $1`, e.AgentID).Scan(&e.AgentMeta)
	if errors.Is(err, sql.ErrNoRows) {
		e.AgentMeta = nil
	} else if err != nil {
		return nil, err
	}
	return &e, nil
}
