package db

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// StandingApproval represents a row from the standing_approvals table.
type StandingApproval struct {
	StandingApprovalID          string
	AgentID                     int64
	UserID                      string
	ActionType                  string
	ActionVersion               string
	Constraints                 []byte // raw JSONB
	SourceActionConfigurationID *string
	Status                      string
	MaxExecutions               *int
	ExecutionCount              int
	StartsAt                    time.Time
	ExpiresAt                   *time.Time // nil means no expiry (until revoked)
	CreatedAt                   time.Time
	RevokedAt                   *time.Time
	ExpiredAt                   *time.Time
	ExhaustedAt                 *time.Time
}

// standingApprovalColumns is the canonical column list for SELECT on the standing_approvals table.
// Keep in sync with scanStandingApproval.
const standingApprovalColumns = `standing_approval_id, agent_id, user_id, action_type, action_version,
	constraints, source_action_configuration_id, status, max_executions, execution_count,
	starts_at, expires_at, created_at, revoked_at, expired_at, exhausted_at`

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
func scanStandingApproval(row pgx.Row) (*StandingApproval, error) {
	var sa StandingApproval
	err := row.Scan(
		&sa.StandingApprovalID, &sa.AgentID, &sa.UserID, &sa.ActionType, &sa.ActionVersion,
		&sa.Constraints, &sa.SourceActionConfigurationID, &sa.Status, &sa.MaxExecutions, &sa.ExecutionCount,
		&sa.StartsAt, &sa.ExpiresAt, &sa.CreatedAt, &sa.RevokedAt, &sa.ExpiredAt, &sa.ExhaustedAt,
	)
	if err != nil {
		return nil, err
	}
	return &sa, nil
}

// CountActiveStandingApprovalsByUser returns the number of standing approvals
// that are currently active for the given user. An approval counts as active
// if its status is 'active' and either has no expiry (expires_at IS NULL) or
// has not yet expired (expires_at > now()).
// This excludes approvals that have technically expired but whose status
// hasn't yet been updated by the cleanup job, so users aren't penalized
// by stale data.
//
// Note: starts_at is intentionally not checked here. Future-dated approvals
// (starts_at > now()) still count toward the plan limit since the user
// created them deliberately — otherwise users could bypass limits by
// scheduling approvals far in the future.
func CountActiveStandingApprovalsByUser(ctx context.Context, db DBTX, userID string) (int, error) {
	var count int
	err := db.QueryRow(ctx,
		`SELECT COUNT(*) FROM standing_approvals
		 WHERE user_id = $1 AND status = 'active' AND (expires_at IS NULL OR expires_at > now())`,
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
	StandingApprovalErrNotFound            = "not_found"
	StandingApprovalErrAlreadyRevoked      = "already_revoked"
	StandingApprovalErrNotActive           = "not_active"
	StandingApprovalErrAgentNotFound       = "agent_not_found"
	StandingApprovalErrDuplicateRequest    = "duplicate_request"
	StandingApprovalErrMaxExecutionsTooLow = "max_executions_too_low"
)

// CreateStandingApprovalParams holds the parameters for creating a standing approval.
type CreateStandingApprovalParams struct {
	StandingApprovalID          string
	AgentID                     int64
	UserID                      string
	ActionType                  string
	ActionVersion               string
	Constraints                 []byte // raw JSONB, may be nil
	SourceActionConfigurationID *string
	MaxExecutions               *int
	StartsAt                    time.Time
	ExpiresAt                   *time.Time // nil means no expiry (until revoked)
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
		   (standing_approval_id, agent_id, user_id, action_type, action_version, constraints, source_action_configuration_id, status, max_executions, starts_at, expires_at)
		 SELECT $1, $2, $3, $4, $5, $6, $7, 'active', $8, $9, $10
		 WHERE EXISTS (SELECT 1 FROM agent_check)
		 RETURNING `+standingApprovalColumns,
		p.StandingApprovalID, p.AgentID, p.UserID, p.ActionType, p.ActionVersion,
		p.Constraints, p.SourceActionConfigurationID, p.MaxExecutions, p.StartsAt, p.ExpiresAt,
	)
	sa, err := scanStandingApproval(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
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
//
// If sourceActionConfigID is non-nil, only rows whose source_action_configuration_id
// equals that value are returned.
func ListStandingApprovalsByUser(ctx context.Context, db DBTX, userID, statusFilter string, sourceActionConfigID *string, limit int, cursor *StandingApprovalCursor) (*StandingApprovalPage, error) {
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

	if sourceActionConfigID != nil {
		p := b.addArg(*sourceActionConfigID)
		where = append(where, "source_action_configuration_id = "+p)
	}

	if cursor != nil {
		tsPlaceholder := b.addArg(cursor.CreatedAt)
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

// CountActiveStandingApprovalsBySourceActionConfigID returns how many active
// standing approvals reference the given action configuration.
func CountActiveStandingApprovalsBySourceActionConfigID(ctx context.Context, db DBTX, userID, sourceConfigID string) (int, error) {
	var n int
	err := db.QueryRow(ctx,
		`SELECT COUNT(*) FROM standing_approvals
		 WHERE user_id = $1 AND source_action_configuration_id = $2 AND status = 'active'`,
		userID, sourceConfigID,
	).Scan(&n)
	return n, err
}

// ListActiveStandingApprovalsBySourceActionConfigID returns active standing
// approvals linked to the given action configuration via
// source_action_configuration_id, scoped to the user.
// ListActiveStandingApprovalsBySourceActionConfigIDs returns active standing
// approvals grouped by source_action_configuration_id for the given config IDs.
func ListActiveStandingApprovalsBySourceActionConfigIDs(ctx context.Context, db DBTX, userID string, configIDs []string) (map[string][]StandingApproval, error) {
	out := make(map[string][]StandingApproval)
	if len(configIDs) == 0 {
		return out, nil
	}
	rows, err := db.Query(ctx,
		`SELECT `+standingApprovalColumns+`
		 FROM standing_approvals
		 WHERE user_id = $1 AND status = 'active'
		   AND source_action_configuration_id = ANY($2::text[])`,
		userID, configIDs,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		sa, err := scanStandingApproval(rows)
		if err != nil {
			return nil, err
		}
		if sa.SourceActionConfigurationID == nil {
			continue
		}
		id := *sa.SourceActionConfigurationID
		out[id] = append(out[id], *sa)
	}
	return out, rows.Err()
}

func ListActiveStandingApprovalsBySourceActionConfigID(ctx context.Context, db DBTX, userID, sourceConfigID string) ([]StandingApproval, error) {
	rows, err := db.Query(ctx,
		`SELECT `+standingApprovalColumns+`
		 FROM standing_approvals
		 WHERE user_id = $1 AND source_action_configuration_id = $2 AND status = 'active'
		 ORDER BY created_at DESC
		 LIMIT $3`,
		userID, sourceConfigID, MaxStandingApprovalListSize,
	)
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
	return approvals, rows.Err()
}

// RevokeActiveStandingApprovalsForSourceActionConfig revokes all active standing
// approvals that reference the given action configuration ID. Returns the
// number of rows updated.
func RevokeActiveStandingApprovalsForSourceActionConfig(ctx context.Context, db DBTX, userID, sourceConfigID string) (int64, error) {
	tag, err := db.Exec(ctx,
		`UPDATE standing_approvals
		 SET status = 'revoked', revoked_at = now()
		 WHERE user_id = $1 AND source_action_configuration_id = $2 AND status = 'active'`,
		userID, sourceConfigID,
	)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
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

	var rows pgx.Rows
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
			agentID, cursor.CreatedAt, cursor.StandingApprovalID, fetchLimit,
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
	if errors.Is(err, pgx.ErrNoRows) {
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
		 SET status = 'revoked', revoked_at = now()
		 WHERE standing_approval_id = $1 AND user_id = $2
		   AND status = 'active'
		 RETURNING `+standingApprovalColumns,
		saID, userID,
	)
	updated, err := scanStandingApproval(row)
	if err == nil {
		return updated, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	// UPDATE matched zero rows — determine why.
	return nil, diagnoseStandingApprovalFailure(ctx, db, saID, userID)
}

// StandingApprovalExecution represents a single recorded execution of a standing approval.
// AgentID, UserID, ActionType, and AgentMeta are derived from related rows
// (not stored on the executions table) and populated via JOIN in queries.
type StandingApprovalExecution struct {
	ExecutionID        int64
	StandingApprovalID string
	AgentID            int64
	UserID             string
	ActionType         string
	AgentMeta          []byte // raw JSONB from agents.metadata, may be nil
	Parameters         []byte // raw JSONB, may be nil
	ExecutedAt         time.Time
	// MaxExecutions and ExecutionCount are populated by
	// RecordStandingApprovalExecutionByAgent only.
	MaxExecutions  *int
	ExecutionCount int
}

// RecordStandingApprovalExecution atomically increments the parent standing
// approval's execution_count and inserts an execution record. Both operations
// run in a single CTE so they succeed or fail together. The UPDATE enforces
// user_id and status='active' to prevent unauthorized or stale executions.
// Returns a domain error via diagnoseStandingApprovalFailure if no matching row.
func RecordStandingApprovalExecution(ctx context.Context, db DBTX, standingApprovalID string, userID string, parameters []byte) (*StandingApprovalExecution, error) {
	var e StandingApprovalExecution

	err := db.QueryRow(ctx,
		`WITH updated AS (
			UPDATE standing_approvals
			SET execution_count = execution_count + 1
			WHERE standing_approval_id = $1 AND user_id = $2 AND status = 'active'
			  AND (expires_at IS NULL OR expires_at > now())
			RETURNING standing_approval_id, agent_id, user_id, action_type
		),
		ins AS (
			INSERT INTO standing_approval_executions (standing_approval_id, parameters)
			SELECT standing_approval_id, $3
			FROM updated
			RETURNING id, standing_approval_id, parameters, executed_at
		)
		SELECT ins.id, ins.standing_approval_id, updated.agent_id, updated.user_id::text,
		       updated.action_type, a.metadata, ins.parameters, ins.executed_at
		FROM ins, updated
		LEFT JOIN agents a ON a.agent_id = updated.agent_id`,
		standingApprovalID, userID, parameters,
	).Scan(&e.ExecutionID, &e.StandingApprovalID, &e.AgentID, &e.UserID, &e.ActionType, &e.AgentMeta, &e.Parameters, &e.ExecutedAt)
	if err == nil {
		return &e, nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, diagnoseStandingApprovalFailure(ctx, db, standingApprovalID, userID)
	}
	return nil, err
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
	StandingApprovalID string
	UserID             string
	Constraints        []byte // raw JSONB
	MaxExecutions      *int
	ExpiresAt          *time.Time // nil means no expiry (until revoked)
}

// UpdateStandingApproval updates the constraints, max_executions, and expires_at of an active
// standing approval belonging to the given user. Returns the updated approval, or a domain error.
//
// The UPDATE atomically guards against setting max_executions below the current
// execution_count — even in the presence of concurrent executions between the
// caller's pre-flight validation and this write.
func UpdateStandingApproval(ctx context.Context, db DBTX, p UpdateStandingApprovalParams) (*StandingApproval, error) {
	row := db.QueryRow(ctx,
		`UPDATE standing_approvals
		 SET constraints = $3, max_executions = $4, expires_at = $5
		 WHERE standing_approval_id = $1 AND user_id = $2
		   AND status = 'active'
		   AND ($4::int IS NULL OR $4::int >= execution_count)
		 RETURNING `+standingApprovalColumns,
		p.StandingApprovalID, p.UserID, p.Constraints, p.MaxExecutions, p.ExpiresAt,
	)
	updated, err := scanStandingApproval(row)
	if err == nil {
		return updated, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	// Diagnose why the UPDATE matched zero rows. Check for the max_executions
	// race before falling back to generic status diagnosis.
	sa, err := GetStandingApprovalByIDAndUser(ctx, db, p.StandingApprovalID, p.UserID)
	if err != nil {
		return nil, err
	}
	if sa == nil {
		return nil, &StandingApprovalError{Code: StandingApprovalErrNotFound}
	}
	if sa.Status == "active" && p.MaxExecutions != nil && *p.MaxExecutions < sa.ExecutionCount {
		return nil, &StandingApprovalError{Code: StandingApprovalErrMaxExecutionsTooLow}
	}
	if sa.Status == "revoked" {
		return nil, &StandingApprovalError{Code: StandingApprovalErrAlreadyRevoked, Status: sa.Status}
	}
	return nil, &StandingApprovalError{Code: StandingApprovalErrNotActive, Status: sa.Status}
}

// FindActiveStandingApprovalsForAgent returns all active standing approvals for
// the given agent and action type, ordered by most recently created first.
// Exact action_type matches are returned before wildcard ("*") matches.
// Returns an empty slice if no match is found.
func FindActiveStandingApprovalsForAgent(ctx context.Context, db DBTX, agentID int64, actionType string) ([]*StandingApproval, error) {
	rows, err := db.Query(ctx,
		`SELECT `+standingApprovalColumns+`
		 FROM (
		   SELECT `+standingApprovalColumns+`, 1 AS priority FROM standing_approvals
		   WHERE agent_id = $1 AND action_type = $2 AND status = 'active'
		     AND starts_at <= now() AND (expires_at IS NULL OR expires_at > now())
		     AND (max_executions IS NULL OR execution_count < max_executions)
		   UNION ALL
		   SELECT `+standingApprovalColumns+`, 2 AS priority FROM standing_approvals
		   WHERE agent_id = $1 AND action_type = '*' AND action_type != $2 AND status = 'active'
		     AND starts_at <= now() AND (expires_at IS NULL OR expires_at > now())
		     AND (max_executions IS NULL OR execution_count < max_executions)
		 ) combined
		 ORDER BY priority, created_at DESC, standing_approval_id DESC
		 LIMIT 100`,
		agentID, actionType,
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

// RecordStandingApprovalExecutionByAgent atomically increments the standing
// approval's execution_count and inserts an execution record, scoped by
// agent_id. This is used by the auto-approval logic in POST /approvals/request
// where authentication is via agent signature rather than user session.
//
// The requestID is stored in the execution record and enforced via a unique
// index on (standing_approval_id, request_id) for idempotency. A duplicate
// request_id returns StandingApprovalErrDuplicateRequest.
func RecordStandingApprovalExecutionByAgent(ctx context.Context, db DBTX, standingApprovalID string, agentID int64, requestID string, parameters []byte) (*StandingApprovalExecution, error) {
	var e StandingApprovalExecution

	err := db.QueryRow(ctx,
		`WITH updated AS (
			UPDATE standing_approvals
			SET execution_count = execution_count + 1
			WHERE standing_approval_id = $1
			  AND agent_id = $2
			  AND status = 'active'
			  AND starts_at <= now()
			  AND (expires_at IS NULL OR expires_at > now())
			  AND (max_executions IS NULL OR execution_count < max_executions)
			RETURNING standing_approval_id, agent_id, user_id, action_type, max_executions, execution_count
		),
		ins AS (
			INSERT INTO standing_approval_executions (standing_approval_id, parameters, request_id)
			SELECT standing_approval_id, $3, $4
			FROM updated
			RETURNING id, standing_approval_id, parameters, executed_at
		)
		SELECT ins.id, ins.standing_approval_id, updated.agent_id, updated.user_id::text,
		       updated.action_type, a.metadata, ins.parameters, ins.executed_at,
		       updated.max_executions, updated.execution_count
		FROM ins, updated
		LEFT JOIN agents a ON a.agent_id = updated.agent_id`,
		standingApprovalID, agentID, parameters, requestID,
	).Scan(&e.ExecutionID, &e.StandingApprovalID, &e.AgentID, &e.UserID, &e.ActionType,
		&e.AgentMeta, &e.Parameters, &e.ExecutedAt,
		&e.MaxExecutions, &e.ExecutionCount)
	if err == nil {
		return &e, nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, &StandingApprovalError{Code: StandingApprovalErrNotActive}
	}
	if isUniqueViolation(err) {
		return nil, &StandingApprovalError{Code: StandingApprovalErrDuplicateRequest}
	}
	return nil, err
}
