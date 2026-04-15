package db

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// Agent status values. These correspond to the CHECK constraint on agents.status.
const (
	AgentStatusPending     = "pending"
	AgentStatusRegistered  = "registered"
	AgentStatusDeactivated = "deactivated"
)

const (
	// maxVerificationAttempts is the maximum number of failed confirmation code
	// attempts before the pending registration is locked out.
	maxVerificationAttempts = 5
)

// Agent represents a row from the agents table.
type Agent struct {
	AgentID              int64
	PublicKey            string
	ApproverID           string
	Status               string
	Metadata             []byte // raw JSONB
	ConfirmationCode     *string // plaintext code for dashboard display (pending agents only)
	VerificationAttempts int
	RegistrationTTL      *int
	ExpiresAt            *time.Time
	RegisteredAt         *time.Time
	DeactivatedAt        *time.Time
	LastActiveAt         *time.Time
	CreatedAt            time.Time
}

// agentColumns is the canonical column list for SELECT/RETURNING on the agents table.
// Keep in sync with scanAgent.
const agentColumns = `agent_id, public_key, approver_id, status, metadata,
	confirmation_code, verification_attempts,
	registration_ttl, expires_at, registered_at, deactivated_at, last_active_at, created_at`

// scanAgent scans a single row into an Agent. The row must select agentColumns.
func scanAgent(row pgx.Row) (*Agent, error) {
	var a Agent
	err := row.Scan(
		&a.AgentID, &a.PublicKey, &a.ApproverID, &a.Status, &a.Metadata,
		&a.ConfirmationCode, &a.VerificationAttempts,
		&a.RegistrationTTL, &a.ExpiresAt, &a.RegisteredAt, &a.DeactivatedAt, &a.LastActiveAt, &a.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// AgentCursor identifies the position of the last item on a page,
// using both created_at and agent_id as a compound key to avoid
// skipping rows when multiple agents share the same created_at.
type AgentCursor struct {
	CreatedAt time.Time
	AgentID   int64
}

// AgentListItem is an Agent enriched with computed stats for list responses.
type AgentListItem struct {
	Agent
	RequestCount30d int
}

// AgentPage holds a page of agents plus a flag indicating whether more exist.
type AgentPage struct {
	Agents  []AgentListItem
	HasMore bool
}

// agentListColumns extends agentColumns with computed stats for the list query.
// request_count_30d includes both one-off approval requests and standing
// approval executions to accurately reflect total agent activity.
//
// Design note: this queries the source-of-truth tables (approvals,
// standing_approval_executions) rather than audit_events, because audit event
// insertion is best-effort and could undercount if inserts fail silently.
const agentListColumns = agentColumns + `,
	(SELECT COUNT(*)
	 FROM approvals
	 WHERE approvals.agent_id = agents.agent_id
	   AND approvals.created_at > now() - interval '30 days')
	+
	(SELECT COUNT(*)
	 FROM standing_approval_executions sae
	 JOIN standing_approvals sa ON sa.standing_approval_id = sae.standing_approval_id
	 WHERE sa.agent_id = agents.agent_id
	   AND sae.executed_at > now() - interval '30 days') AS request_count_30d`

// scanAgentListItem scans a row selected with agentListColumns into an AgentListItem.
func scanAgentListItem(row pgx.Row) (*AgentListItem, error) {
	var item AgentListItem
	err := row.Scan(
		&item.AgentID, &item.PublicKey, &item.ApproverID, &item.Status, &item.Metadata,
		&item.ConfirmationCode, &item.VerificationAttempts,
		&item.RegistrationTTL, &item.ExpiresAt, &item.RegisteredAt,
		&item.DeactivatedAt, &item.LastActiveAt, &item.CreatedAt,
		&item.RequestCount30d,
	)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

// GetAgentsByApprover returns agents owned by the given approver with
// cursor-based pagination, ordered by creation time descending (newest first),
// with agent_id as a tiebreaker. Pass a nil cursor to start from the beginning.
// Limit is clamped to [1, 100] with a default of 50 when <= 0.
//
// Expired pending agents (status='pending' with expires_at <= now()) are
// excluded from results since they can never complete registration.
//
// Each returned AgentListItem includes a RequestCount30d computed from
// the approvals table (number of approval requests in the last 30 days).
func GetAgentsByApprover(ctx context.Context, db DBTX, approverID string, limit int, cursor *AgentCursor) (*AgentPage, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	// Fetch one extra row to determine has_more.
	fetchLimit := limit + 1

	var rows pgx.Rows
	var err error
	// Exclude expired pending agents: they can never complete registration
	// and would otherwise waste pagination slots.
	expiredFilter := `AND NOT (status = 'pending' AND expires_at IS NOT NULL AND expires_at <= now())`

	if cursor != nil {
		rows, err = db.Query(ctx,
			`SELECT `+agentListColumns+`
			 FROM agents
			 WHERE approver_id = $1 AND (created_at, agent_id) < ($2, $3)
			 `+expiredFilter+`
			 ORDER BY created_at DESC, agent_id DESC
			 LIMIT $4`,
			approverID, cursor.CreatedAt, cursor.AgentID, fetchLimit,
		)
	} else {
		rows, err = db.Query(ctx,
			`SELECT `+agentListColumns+`
			 FROM agents
			 WHERE approver_id = $1
			 `+expiredFilter+`
			 ORDER BY created_at DESC, agent_id DESC
			 LIMIT $2`,
			approverID, fetchLimit,
		)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var agents []AgentListItem
	for rows.Next() {
		item, err := scanAgentListItem(rows)
		if err != nil {
			return nil, err
		}
		agents = append(agents, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	hasMore := len(agents) > limit
	if hasMore {
		agents = agents[:limit]
	}

	return &AgentPage{Agents: agents, HasMore: hasMore}, nil
}

// GetAgentByID returns a single agent by ID, scoped to the given approver.
// Returns nil if the agent doesn't exist or doesn't belong to the approver.
func GetAgentByID(ctx context.Context, db DBTX, agentID int64, approverID string) (*Agent, error) {
	row := db.QueryRow(ctx,
		`SELECT `+agentColumns+`
		 FROM agents
		 WHERE agent_id = $1 AND approver_id = $2`,
		agentID, approverID,
	)
	a, err := scanAgent(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return a, nil
}

// UpdateAgentMetadata shallow-merges the supplied JSON into the agent's existing
// metadata (existing keys are preserved unless overridden). The agent must belong
// to the given approver. Returns the updated agent, or nil if no match was found.
func UpdateAgentMetadata(ctx context.Context, db DBTX, agentID int64, approverID string, metadata []byte) (*Agent, error) {
	row := db.QueryRow(ctx,
		`UPDATE agents
		 SET metadata = COALESCE(metadata, '{}'::jsonb) || $3
		 WHERE agent_id = $1 AND approver_id = $2
		 RETURNING `+agentColumns,
		agentID, approverID, metadata,
	)
	a, err := scanAgent(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return a, nil
}

// AgentBelongsToUser checks whether the given agent_id is owned by the given user.
func AgentBelongsToUser(ctx context.Context, db DBTX, agentID int64, userID string) (bool, error) {
	var exists bool
	err := db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM agents WHERE agent_id = $1 AND approver_id = $2)`,
		agentID, userID,
	).Scan(&exists)
	return exists, err
}

// RegisterAgent atomically transitions an agent from 'pending' to 'registered'
// and sets registered_at. The agent must belong to the given approver and be in
// 'pending' status. Returns the updated agent, or nil if no matching pending
// agent was found.
func RegisterAgent(ctx context.Context, db DBTX, agentID int64, approverID string) (*Agent, error) {
	row := db.QueryRow(ctx,
		`UPDATE agents
		 SET status = 'registered', registered_at = now(), confirmation_code = NULL
		 WHERE agent_id = $1 AND approver_id = $2 AND status = 'pending'
		 RETURNING `+agentColumns,
		agentID, approverID,
	)
	a, err := scanAgent(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return a, nil
}

// DeactivateAgent sets an agent's status to 'deactivated' and revokes any
// active standing approvals for the agent in a single atomic operation.
// It returns the updated agent, or nil if no matching active agent was found
// (i.e. the agent doesn't exist, doesn't belong to the approver, or is already deactivated).
func DeactivateAgent(ctx context.Context, db DBTX, agentID int64, approverID string) (*Agent, error) {
	row := db.QueryRow(ctx,
		`WITH deactivated AS (
		     UPDATE agents
		     SET status = 'deactivated', deactivated_at = now()
		     WHERE agent_id = $1 AND approver_id = $2 AND status != 'deactivated'
		     RETURNING `+agentColumns+`
		 ), revoke_standing AS (
		     UPDATE standing_approvals
		     SET status = 'revoked', revoked_at = now()
		     WHERE agent_id = (SELECT agent_id FROM deactivated) AND status = 'active'
		 )
		 SELECT `+agentColumns+`
		 FROM deactivated`,
		agentID, approverID,
	)
	a, err := scanAgent(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return a, nil
}

// InsertPendingAgent creates a new agent in 'pending' status, linked to the user
// via approverID (the invite's user_id). The server auto-assigns the agent_id
// (bigserial). confirmationCode is the plaintext code shown to the user on the
// dashboard. registrationTTL controls how long (in seconds) the agent has to
// verify; expires_at is computed server-side as now() + TTL. metadata may be nil.
func InsertPendingAgent(ctx context.Context, db DBTX, approverID, publicKey, confirmationCode string, registrationTTL int, metadata []byte) (*Agent, error) {
	row := db.QueryRow(ctx,
		`INSERT INTO agents (public_key, approver_id, status, metadata, confirmation_code, registration_ttl, expires_at)
		 VALUES ($1, $2, 'pending', $3, $4, $5, now() + make_interval(secs => $6))
		 RETURNING `+agentColumns,
		publicKey, approverID, metadata, confirmationCode, registrationTTL, float64(registrationTTL),
	)
	return scanAgent(row)
}

// Sentinel errors for VerifyAgentConfirmationCode.
var (
	ErrAgentNotPending       = errors.New("agent is not in pending status")
	ErrRegistrationExpired   = errors.New("registration has expired")
	ErrVerificationLocked    = errors.New("too many failed verification attempts")
	ErrInvalidConfirmation   = errors.New("incorrect confirmation code")
)

// VerifyAgentConfirmationCode atomically increments verification_attempts on
// the pending agent, then compares the submitted code against the stored
// plaintext using constant-time comparison. On match, it transitions the agent
// to status='registered'. On mismatch, it returns ErrInvalidConfirmation with
// the attempt already counted.
//
// submittedCode must be normalized (uppercase, no hyphens) before calling.
//
// The atomic UPDATE ensures that concurrent requests cannot bypass the lockout
// threshold (no TOCTOU race).
func VerifyAgentConfirmationCode(ctx context.Context, db DBTX, agentID int64, submittedCode string) (*Agent, error) {
	// Atomically increment attempts and return the agent, but only if it's
	// still eligible (pending, not expired, under the attempt limit).
	row := db.QueryRow(ctx,
		`UPDATE agents
		 SET verification_attempts = verification_attempts + 1
		 WHERE agent_id = $1
		   AND status = 'pending'
		   AND (expires_at IS NULL OR expires_at > now())
		   AND verification_attempts < $2
		 RETURNING `+agentColumns,
		agentID, maxVerificationAttempts,
	)
	a, err := scanAgent(row)
	if errors.Is(err, pgx.ErrNoRows) {
		// The UPDATE matched nothing — determine why for a specific error.
		return diagnosePendingAgent(ctx, db, agentID)
	}
	if err != nil {
		return nil, fmt.Errorf("increment attempts: %w", err)
	}

	// Constant-time comparison of plaintext codes.
	// Normalize the stored code the same way (strip hyphens, uppercase) for
	// a fair comparison. Note: the normalization itself is not constant-time,
	// but operates on fixed-length values (the stored code is always
	// shared.ConfirmationCodeLength chars plus a hyphen at a fixed position),
	// so there's nothing to leak. The 5-attempt lockout makes timing attacks
	// infeasible regardless.
	storedCode := ""
	if a.ConfirmationCode != nil {
		storedCode = strings.ToUpper(strings.ReplaceAll(*a.ConfirmationCode, "-", ""))
	}
	if subtle.ConstantTimeCompare([]byte(storedCode), []byte(submittedCode)) != 1 {
		return a, ErrInvalidConfirmation
	}

	// Success — transition to registered and clear the confirmation code.
	updateRow := db.QueryRow(ctx,
		`UPDATE agents
		 SET status = 'registered', registered_at = now(), confirmation_code = NULL
		 WHERE agent_id = $1 AND status = 'pending'
		 RETURNING `+agentColumns,
		agentID,
	)
	registered, err := scanAgent(updateRow)
	if errors.Is(err, pgx.ErrNoRows) {
		// Another concurrent request already registered this agent.
		// Re-fetch the current state so the caller can distinguish
		// "already registered" from other non-pending states.
		current, lookupErr := scanAgent(db.QueryRow(ctx,
			`SELECT `+agentColumns+` FROM agents WHERE agent_id = $1`, agentID))
		if lookupErr == nil && current != nil {
			return current, ErrAgentNotPending
		}
		return a, ErrAgentNotPending
	}
	if err != nil {
		return nil, fmt.Errorf("register agent: %w", err)
	}
	return registered, nil
}

// diagnosePendingAgent does a non-locking read to determine why the atomic
// attempt increment failed, returning the appropriate sentinel error.
func diagnosePendingAgent(ctx context.Context, db DBTX, agentID int64) (*Agent, error) {
	row := db.QueryRow(ctx,
		`SELECT `+agentColumns+`
		 FROM agents
		 WHERE agent_id = $1`,
		agentID,
	)
	a, err := scanAgent(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil // not found
	}
	if err != nil {
		return nil, fmt.Errorf("lookup agent: %w", err)
	}
	if a.Status != AgentStatusPending {
		return a, ErrAgentNotPending
	}
	if a.ExpiresAt != nil && time.Now().After(*a.ExpiresAt) {
		return a, ErrRegistrationExpired
	}
	if a.VerificationAttempts >= maxVerificationAttempts {
		return a, ErrVerificationLocked
	}
	// Defensive fallback — shouldn't reach here.
	return a, ErrVerificationLocked
}

// GetAgentByIDUnscoped returns an agent by ID without approver scoping.
// Used by the verify endpoint where the agent authenticates via signature, not session.
// Returns nil if the agent doesn't exist.
func GetAgentByIDUnscoped(ctx context.Context, db DBTX, agentID int64) (*Agent, error) {
	row := db.QueryRow(ctx,
		`SELECT `+agentColumns+`
		 FROM agents
		 WHERE agent_id = $1`,
		agentID,
	)
	a, err := scanAgent(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return a, nil
}

// CountRegisteredAgentsByUser returns the number of agents in 'registered' or
// 'pending' (not yet expired) status for the given user. Both count toward the
// plan limit because pending agents will become registered on verification.
func CountRegisteredAgentsByUser(ctx context.Context, db DBTX, userID string) (int, error) {
	var count int
	err := db.QueryRow(ctx,
		`SELECT COUNT(*) FROM agents
		 WHERE approver_id = $1
		   AND (status = 'registered'
		        OR (status = 'pending' AND (expires_at IS NULL OR expires_at > now())))`,
		userID,
	).Scan(&count)
	return count, err
}

// TouchAgentLastActive updates the agent's last_active_at timestamp to now().
// This is a best-effort operation — callers should not fail the request if it errors.
func TouchAgentLastActive(ctx context.Context, db DBTX, agentID int64) error {
	_, err := db.Exec(ctx,
		`UPDATE agents SET last_active_at = now() WHERE agent_id = $1`,
		agentID,
	)
	return err
}
