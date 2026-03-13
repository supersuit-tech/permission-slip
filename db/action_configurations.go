package db

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// ActionConfiguration represents a row from the action_configurations table.
type ActionConfiguration struct {
	ID          string
	AgentID     int64
	UserID      string
	ConnectorID string
	ActionType  string
	Parameters  []byte // raw JSONB
	Status      string
	Name        string
	Description *string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// actionConfigColumns is the canonical column list for SELECT on the action_configurations table.
// Keep in sync with scanActionConfig.
const actionConfigColumns = `id, agent_id, user_id, connector_id, action_type,
	parameters, status, name, description, created_at, updated_at`

// scanActionConfig scans a single row into an ActionConfiguration.
// The row must select actionConfigColumns.
func scanActionConfig(row pgx.Row) (*ActionConfiguration, error) {
	var ac ActionConfiguration
	err := row.Scan(
		&ac.ID, &ac.AgentID, &ac.UserID, &ac.ConnectorID, &ac.ActionType,
		&ac.Parameters, &ac.Status, &ac.Name, &ac.Description,
		&ac.CreatedAt, &ac.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &ac, nil
}

// ActionConfigError represents a domain-specific error from action configuration operations.
type ActionConfigError struct {
	Code    ActionConfigErrCode
	Message string
}

func (e *ActionConfigError) Error() string { return e.Message }

// ActionConfigErrCode enumerates action configuration-specific error codes.
type ActionConfigErrCode int

const (
	ActionConfigErrNotFound      ActionConfigErrCode = iota
	ActionConfigErrAgentNotFound                     // agent does not exist or does not belong to user
	ActionConfigErrInvalidRef                        // FK violation (connector, action, or credential)
)

// MaxActionConfigListSize is the maximum number of action configurations returned by list queries.
const MaxActionConfigListSize = 200

// CreateActionConfigParams holds the parameters for inserting a new action configuration.
type CreateActionConfigParams struct {
	ID          string
	AgentID     int64
	UserID      string
	ConnectorID string
	ActionType  string
	Parameters  []byte // raw JSONB
	Name        string
	Description *string
}

// CreateActionConfig inserts a new action configuration with status 'active'.
// The INSERT is guarded by an agent ownership check: if the agent does not
// belong to the user, no row is inserted and ActionConfigErrAgentNotFound
// is returned. FK violations (connector, action_type) are mapped
// to ActionConfigErrInvalidRef.
func CreateActionConfig(ctx context.Context, db DBTX, p CreateActionConfigParams) (*ActionConfiguration, error) {
	row := db.QueryRow(ctx,
		`WITH agent_check AS (
			SELECT 1 FROM agents WHERE agent_id = $2 AND approver_id = $3
		)
		INSERT INTO action_configurations
		   (id, agent_id, user_id, connector_id, action_type, parameters, name, description)
		 SELECT $1, $2, $3, $4, $5, $6, $7, $8
		 WHERE EXISTS (SELECT 1 FROM agent_check)
		 RETURNING `+actionConfigColumns,
		p.ID, p.AgentID, p.UserID, p.ConnectorID, p.ActionType,
		p.Parameters, p.Name, p.Description,
	)
	ac, err := scanActionConfig(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, &ActionConfigError{Code: ActionConfigErrAgentNotFound, Message: "agent not found or not owned by user"}
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == PgCodeForeignKeyViolation {
			return nil, &ActionConfigError{Code: ActionConfigErrInvalidRef, Message: "invalid connector, action type, or credential reference"}
		}
		return nil, err
	}
	return ac, nil
}

// GetActionConfigByID returns the action configuration with the given ID
// belonging to the given user, or nil if not found.
func GetActionConfigByID(ctx context.Context, db DBTX, configID, userID string) (*ActionConfiguration, error) {
	row := db.QueryRow(ctx,
		`SELECT `+actionConfigColumns+`
		 FROM action_configurations
		 WHERE id = $1 AND user_id = $2`,
		configID, userID,
	)
	ac, err := scanActionConfig(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return ac, nil
}

// ListActionConfigsByAgent returns action configurations for the given agent,
// scoped to the user. Results are ordered by created_at descending (newest first)
// and capped at MaxActionConfigListSize.
func ListActionConfigsByAgent(ctx context.Context, db DBTX, agentID int64, userID string) ([]ActionConfiguration, error) {
	rows, err := db.Query(ctx,
		`SELECT `+actionConfigColumns+`
		 FROM action_configurations
		 WHERE agent_id = $1 AND user_id = $2
		 ORDER BY created_at DESC
		 LIMIT $3`,
		agentID, userID, MaxActionConfigListSize,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var configs []ActionConfiguration
	for rows.Next() {
		ac, err := scanActionConfig(rows)
		if err != nil {
			return nil, err
		}
		configs = append(configs, *ac)
	}
	return configs, rows.Err()
}

// UpdateActionConfigParams holds the mutable fields for updating an action configuration.
type UpdateActionConfigParams struct {
	ID          string
	UserID      string
	Parameters  []byte // raw JSONB
	Status      *string
	Name        *string
	Description *string
}

// UpdateActionConfig updates the mutable fields of an action configuration.
// Only non-nil fields in the params struct are updated. Returns the updated
// configuration, or a domain error if not found.
func UpdateActionConfig(ctx context.Context, db DBTX, p UpdateActionConfigParams) (*ActionConfiguration, error) {
	row := db.QueryRow(ctx,
		`UPDATE action_configurations
		 SET parameters    = COALESCE($3, parameters),
		     status        = COALESCE($4, status),
		     name          = COALESCE($5, name),
		     description   = COALESCE($6, description),
		     updated_at    = now()
		 WHERE id = $1 AND user_id = $2
		 RETURNING `+actionConfigColumns,
		p.ID, p.UserID, p.Parameters, p.Status, p.Name, p.Description,
	)
	ac, err := scanActionConfig(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, &ActionConfigError{Code: ActionConfigErrNotFound, Message: "action configuration not found"}
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == PgCodeForeignKeyViolation {
			return nil, &ActionConfigError{Code: ActionConfigErrInvalidRef, Message: "invalid credential reference"}
		}
		return nil, err
	}
	return ac, nil
}

// GetActiveActionConfigForAgent returns an active action configuration by ID,
// validated against the given agent. This is used by agent-facing endpoints
// (signature auth) where the agent references a configuration_id. The lookup
// joins through the agent's approver to verify ownership without requiring the
// user_id directly. Returns nil if the config is not found, not active, or
// does not belong to the agent.
func GetActiveActionConfigForAgent(ctx context.Context, db DBTX, configID string, agentID int64) (*ActionConfiguration, error) {
	row := db.QueryRow(ctx,
		`SELECT ac.id, ac.agent_id, ac.user_id, ac.connector_id, ac.action_type,
		        ac.parameters, ac.status, ac.name, ac.description,
		        ac.created_at, ac.updated_at
		 FROM action_configurations ac
		 JOIN agents a ON a.agent_id = ac.agent_id AND a.approver_id = ac.user_id
		 WHERE ac.id = $1 AND ac.agent_id = $2 AND ac.status = 'active'
		   AND a.status = 'registered'`,
		configID, agentID,
	)
	ac, err := scanActionConfig(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return ac, nil
}

// DeleteActionConfig deletes an action configuration by ID, scoped to the given user.
// Returns the deleted configuration, or nil if not found.
func DeleteActionConfig(ctx context.Context, db DBTX, configID, userID string) (*ActionConfiguration, error) {
	row := db.QueryRow(ctx,
		`DELETE FROM action_configurations
		 WHERE id = $1 AND user_id = $2
		 RETURNING `+actionConfigColumns,
		configID, userID,
	)
	ac, err := scanActionConfig(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return ac, nil
}

// WildcardActionType is the reserved action_type value that means
// "all actions on this connector". A wildcard config covers every current
// and future action — the agent can choose any action and any parameter
// values at execution time. Only one wildcard config is allowed per
// agent + connector pair (enforced by idx_action_config_wildcard_unique).
//
// SECURITY: Wildcard configs skip action-type and parameter validation at
// execution time, but standing approvals still gate actual execution
// per-action-type.
const WildcardActionType = "*"

// ConnectorActionExists checks whether a (connector_id, action_type) pair
// exists in the connector_actions table. Used to validate non-wildcard
// action types now that the composite FK has been dropped.
func ConnectorActionExists(ctx context.Context, db DBTX, connectorID, actionType string) (bool, error) {
	var exists bool
	err := db.QueryRow(ctx,
		`SELECT EXISTS(
			SELECT 1 FROM connector_actions
			WHERE connector_id = $1 AND action_type = $2
		)`,
		connectorID, actionType,
	).Scan(&exists)
	return exists, err
}
