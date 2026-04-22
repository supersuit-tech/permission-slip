package db

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// AgentConnectorInstance is a row in agent_connectors (one instance of a connector type for an agent).
// DisplayName is derived from the bound credential: the static API key label, or for OAuth
// connections the identifier in extra_data (display_name / email, optionally suffixed with
// " @ <team_name>" for Slack). See connectorInstanceDisplayNameSQL for the exact priority.
type AgentConnectorInstance struct {
	ConnectorInstanceID string
	AgentID             int64
	ConnectorID         string
	ApproverID          string
	DisplayName         string
	IsDefault           bool
	EnabledAt           time.Time
}

// connectorInstanceDisplayNameSQL is the shared SQL expression that derives a human-readable
// display name for a connector instance. It must stay in sync with the Go helper
// api.displayNameFromExtraData: priority is cr.label → "<display_name> @ <team_name>" →
// "<email> @ <team_name>" → display_name → email → team_name → "".
//
// Callers must alias LEFT JOINs on credentials as cr and oauth_connections as oc.
const connectorInstanceDisplayNameSQL = `COALESCE(
    cr.label,
    CASE
        WHEN NULLIF(oc.extra_data->>'display_name', '') IS NOT NULL
             AND NULLIF(oc.extra_data->>'team_name', '') IS NOT NULL
            THEN (oc.extra_data->>'display_name') || ' @ ' || (oc.extra_data->>'team_name')
        WHEN NULLIF(oc.extra_data->>'email', '') IS NOT NULL
             AND NULLIF(oc.extra_data->>'team_name', '') IS NOT NULL
            THEN (oc.extra_data->>'email') || ' @ ' || (oc.extra_data->>'team_name')
        WHEN NULLIF(oc.extra_data->>'display_name', '') IS NOT NULL
            THEN oc.extra_data->>'display_name'
        WHEN NULLIF(oc.extra_data->>'email', '') IS NOT NULL
            THEN oc.extra_data->>'email'
        WHEN NULLIF(oc.extra_data->>'team_name', '') IS NOT NULL
            THEN oc.extra_data->>'team_name'
        ELSE ''
    END,
    ''
)`

// AgentConnectorInstanceError is a domain error for connector instance operations.
type AgentConnectorInstanceError struct {
	Code                 AgentConnectorInstanceErrCode
	AmbiguousInstanceIDs []string // set when Code == AgentConnectorInstanceErrAmbiguousDisplay
}

func (e *AgentConnectorInstanceError) Error() string { return string(e.Code) }

// AgentConnectorInstanceErrCode enumerates connector-instance-specific error codes.
type AgentConnectorInstanceErrCode string

const (
	AgentConnectorInstanceErrNotFound            AgentConnectorInstanceErrCode = "connector_instance_not_found"
	AgentConnectorInstanceErrCannotDeleteDefault AgentConnectorInstanceErrCode = "cannot_delete_default_instance"
	AgentConnectorInstanceErrAmbiguousDisplay    AgentConnectorInstanceErrCode = "connector_instance_ambiguous_display"
)

// agentConnectorInstanceSelect is the standard SELECT for an agent_connectors row with display name
// from the assigned credential (static API key label or OAuth extra_data identifier).
const agentConnectorInstanceSelect = `
	SELECT ac.connector_instance_id, ac.agent_id, ac.connector_id, ac.approver_id,
	       ` + connectorInstanceDisplayNameSQL + ` AS display_name,
	       ac.is_default, ac.enabled_at
	FROM agent_connectors ac
	LEFT JOIN agent_connector_credentials acc
	       ON acc.agent_id = ac.agent_id
	      AND acc.connector_id = ac.connector_id
	      AND acc.approver_id = ac.approver_id
	      AND acc.connector_instance_id = ac.connector_instance_id
	LEFT JOIN credentials cr ON cr.id = acc.credential_id
	LEFT JOIN oauth_connections oc ON oc.id = acc.oauth_connection_id`

// CreateAgentConnectorInstanceParams holds parameters for creating a new instance (non-default row).
type CreateAgentConnectorInstanceParams struct {
	AgentID     int64
	ApproverID  string
	ConnectorID string
}

// CreateAgentConnectorInstance inserts a new agent connector instance.
// The first instance for an (agent, connector) pair is the default (enforced by DB trigger);
// additional instances are non-default.
func CreateAgentConnectorInstance(ctx context.Context, db DBTX, p CreateAgentConnectorInstanceParams) (*AgentConnectorInstance, error) {
	var agentOK bool
	if err := db.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM agents WHERE agent_id = $1 AND approver_id = $2)`,
		p.AgentID, p.ApproverID,
	).Scan(&agentOK); err != nil {
		return nil, err
	}
	if !agentOK {
		return nil, &AgentConnectorError{Code: AgentConnectorErrAgentNotFound}
	}

	var connOK bool
	if err := db.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM connectors WHERE id = $1)`,
		p.ConnectorID,
	).Scan(&connOK); err != nil {
		return nil, err
	}
	if !connOK {
		return nil, &AgentConnectorError{Code: AgentConnectorErrConnectorNotFound}
	}

	var enabled bool
	if err := db.QueryRow(ctx,
		`SELECT EXISTS (
			SELECT 1 FROM agent_connectors
			WHERE agent_id = $1 AND approver_id = $2 AND connector_id = $3
		)`,
		p.AgentID, p.ApproverID, p.ConnectorID,
	).Scan(&enabled); err != nil {
		return nil, err
	}
	if !enabled {
		return nil, &AgentConnectorError{Code: AgentConnectorErrConnectorNotEnabled}
	}

	var newID string
	err := db.QueryRow(ctx, `
		INSERT INTO agent_connectors (agent_id, approver_id, connector_id, is_default)
		VALUES ($1, $2, $3, false)
		RETURNING connector_instance_id::text`,
		p.AgentID, p.ApproverID, p.ConnectorID,
	).Scan(&newID)
	if err != nil {
		return nil, err
	}
	return GetAgentConnectorInstance(ctx, db, p.AgentID, p.ApproverID, p.ConnectorID, newID)
}

func scanAgentConnectorInstance(row pgx.Row) (*AgentConnectorInstance, error) {
	var inst AgentConnectorInstance
	err := row.Scan(
		&inst.ConnectorInstanceID, &inst.AgentID, &inst.ConnectorID, &inst.ApproverID,
		&inst.DisplayName, &inst.IsDefault, &inst.EnabledAt,
	)
	if err != nil {
		return nil, err
	}
	return &inst, nil
}

// ListAgentConnectorInstances returns all instances for an agent+connector scoped to the approver.
func ListAgentConnectorInstances(ctx context.Context, db DBTX, agentID int64, approverID, connectorID string) ([]AgentConnectorInstance, error) {
	rows, err := db.Query(ctx, agentConnectorInstanceSelect+`
		WHERE ac.agent_id = $1 AND ac.approver_id = $2 AND ac.connector_id = $3
		ORDER BY ac.enabled_at ASC, ac.connector_instance_id ASC`,
		agentID, approverID, connectorID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []AgentConnectorInstance
	for rows.Next() {
		inst, err := scanAgentConnectorInstance(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *inst)
	}
	return out, rows.Err()
}

// GetAgentConnectorInstance returns a single instance by ID, scoped to agent and approver.
func GetAgentConnectorInstance(ctx context.Context, db DBTX, agentID int64, approverID, connectorID, connectorInstanceID string) (*AgentConnectorInstance, error) {
	row := db.QueryRow(ctx, agentConnectorInstanceSelect+`
		WHERE ac.agent_id = $1 AND ac.approver_id = $2 AND ac.connector_id = $3 AND ac.connector_instance_id = $4::uuid`,
		agentID, approverID, connectorID, connectorInstanceID,
	)
	inst, err := scanAgentConnectorInstance(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return inst, nil
}

// GetDefaultAgentConnectorInstance returns the default instance for an (agent, connector) pair.
func GetDefaultAgentConnectorInstance(ctx context.Context, db DBTX, agentID int64, approverID, connectorID string) (*AgentConnectorInstance, error) {
	row := db.QueryRow(ctx, agentConnectorInstanceSelect+`
		WHERE ac.agent_id = $1 AND ac.approver_id = $2 AND ac.connector_id = $3 AND ac.is_default`,
		agentID, approverID, connectorID,
	)
	inst, err := scanAgentConnectorInstance(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return inst, nil
}

// GetDefaultAgentConnectorInstanceByAgent returns the default instance using only agent_id and connector_id.
// Each agent has a single approver; this is used when approver_id is not available on the call path.
func GetDefaultAgentConnectorInstanceByAgent(ctx context.Context, db DBTX, agentID int64, connectorID string) (*AgentConnectorInstance, error) {
	row := db.QueryRow(ctx, agentConnectorInstanceSelect+`
		WHERE ac.agent_id = $1 AND ac.connector_id = $2 AND ac.is_default`,
		agentID, connectorID,
	)
	inst, err := scanAgentConnectorInstance(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return inst, nil
}

// SetDefaultAgentConnectorInstance marks the given instance as the default for
// (agent, connector) and clears is_default on sibling instances.
func SetDefaultAgentConnectorInstance(ctx context.Context, db DBTX, agentID int64, approverID, connectorID, connectorInstanceID string) (*AgentConnectorInstance, error) {
	tx, owned, err := BeginOrContinue(ctx, db)
	if err != nil {
		return nil, err
	}
	if owned {
		defer RollbackTx(ctx, tx) //nolint:errcheck // commit path clears
	}

	_, err = tx.Exec(ctx, `
		UPDATE agent_connectors
		SET is_default = false
		WHERE agent_id = $1 AND approver_id = $2 AND connector_id = $3
		  AND is_default
		  AND connector_instance_id <> $4::uuid`,
		agentID, approverID, connectorID, connectorInstanceID,
	)
	if err != nil {
		return nil, err
	}

	tag, err := tx.Exec(ctx, `
		UPDATE agent_connectors
		SET is_default = true
		WHERE agent_id = $1 AND approver_id = $2 AND connector_id = $3 AND connector_instance_id = $4::uuid`,
		agentID, approverID, connectorID, connectorInstanceID,
	)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		if owned {
			_ = RollbackTx(ctx, tx)
		}
		return nil, nil
	}

	if owned {
		if err := CommitTx(ctx, tx); err != nil {
			return nil, err
		}
	}
	return GetAgentConnectorInstance(ctx, db, agentID, approverID, connectorID, connectorInstanceID)
}

// DeleteAgentConnectorInstance removes a non-default instance and revokes instance-scoped standing approvals.
func DeleteAgentConnectorInstance(ctx context.Context, db DBTX, agentID int64, approverID, connectorID, connectorInstanceID string) error {
	tx, owned, err := BeginOrContinue(ctx, db)
	if err != nil {
		return err
	}
	if owned {
		defer RollbackTx(ctx, tx) //nolint:errcheck // commit path clears
	}

	var isDefault bool
	err = tx.QueryRow(ctx, `
		SELECT is_default FROM agent_connectors
		WHERE agent_id = $1 AND approver_id = $2 AND connector_id = $3 AND connector_instance_id = $4::uuid`,
		agentID, approverID, connectorID, connectorInstanceID,
	).Scan(&isDefault)
	if errors.Is(err, pgx.ErrNoRows) {
		return &AgentConnectorInstanceError{Code: AgentConnectorInstanceErrNotFound}
	}
	if err != nil {
		return err
	}
	if isDefault {
		return &AgentConnectorInstanceError{Code: AgentConnectorInstanceErrCannotDeleteDefault}
	}

	_, err = tx.Exec(ctx, `
		UPDATE standing_approvals
		SET status = 'revoked', revoked_at = now()
		WHERE agent_id = $1
		  AND user_id = $2
		  AND status = 'active'
		  AND connector_instance_id = $3::uuid`,
		agentID, approverID, connectorInstanceID,
	)
	if err != nil {
		return err
	}

	tag, err := tx.Exec(ctx, `
		DELETE FROM agent_connectors
		WHERE agent_id = $1 AND approver_id = $2 AND connector_id = $3 AND connector_instance_id = $4::uuid`,
		agentID, approverID, connectorID, connectorInstanceID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return &AgentConnectorInstanceError{Code: AgentConnectorInstanceErrNotFound}
	}

	if owned {
		return CommitTx(ctx, tx)
	}
	return nil
}

// ResolveAgentConnectorInstance finds an instance by UUID or by credential display name (trimmed).
// If multiple instances share the same display string, returns AgentConnectorInstanceError with
// Code AgentConnectorInstanceErrAmbiguousDisplay (caller should surface HTTP 400 with matching UUIDs).
func ResolveAgentConnectorInstance(ctx context.Context, db DBTX, agentID int64, approverID, connectorID, selector string) (*AgentConnectorInstance, error) {
	sel := strings.TrimSpace(selector)
	if sel == "" {
		return nil, nil
	}

	if _, err := uuid.Parse(sel); err == nil {
		return GetAgentConnectorInstance(ctx, db, agentID, approverID, connectorID, sel)
	}

	rows, err := db.Query(ctx, agentConnectorInstanceSelect+`
		WHERE ac.agent_id = $1 AND ac.approver_id = $2 AND ac.connector_id = $3
		  AND `+connectorInstanceDisplayNameSQL+` = $4
		ORDER BY ac.connector_instance_id ASC`,
		agentID, approverID, connectorID, sel,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*AgentConnectorInstance
	for rows.Next() {
		inst, err := scanAgentConnectorInstance(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, inst)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, nil
	}
	if len(list) > 1 {
		ids := make([]string, len(list))
		for i, inst := range list {
			ids[i] = inst.ConnectorInstanceID
		}
		return nil, &AgentConnectorInstanceError{
			Code:                 AgentConnectorInstanceErrAmbiguousDisplay,
			AmbiguousInstanceIDs: ids,
		}
	}
	return list[0], nil
}
