package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// AgentConnector represents a connector enabled for an agent,
// enriched with the connector summary fields.
type AgentConnector struct {
	ConnectorSummary
	EnabledAt time.Time
}

// AgentConnectorRow represents the raw agent_connectors junction table row.
type AgentConnectorRow struct {
	AgentID     int64
	ConnectorID string
	EnabledAt   time.Time
}

// AgentConnectorError represents a domain-specific error from agent connector operations.
type AgentConnectorError struct {
	Code AgentConnectorErrCode
}

func (e *AgentConnectorError) Error() string { return string(e.Code) }

// AgentConnectorErrCode enumerates agent-connector-specific error codes.
type AgentConnectorErrCode string

const (
	AgentConnectorErrAgentNotFound       AgentConnectorErrCode = "agent_not_found"
	AgentConnectorErrConnectorNotFound   AgentConnectorErrCode = "connector_not_found"
	AgentConnectorErrConnectorNotEnabled AgentConnectorErrCode = "connector_not_enabled"
)

// ListAgentConnectors returns enabled connectors for an agent, scoped to the approver.
// Each entry includes the connector summary (name, actions, required credentials) and
// the enabled_at timestamp.
// AgentConnectorEnabled returns true if the given connector is enabled for the agent.
func AgentConnectorEnabled(ctx context.Context, db DBTX, agentID int64, approverID, connectorID string) (bool, error) {
	var ok bool
	err := db.QueryRow(ctx,
		`SELECT EXISTS (
			SELECT 1 FROM agent_connectors
			WHERE agent_id = $1 AND approver_id = $2 AND connector_id = $3
		)`,
		agentID, approverID, connectorID,
	).Scan(&ok)
	return ok, err
}

func ListAgentConnectors(ctx context.Context, db DBTX, agentID int64, approverID string) ([]AgentConnector, error) {
	rows, err := db.Query(ctx, `
		SELECT c.id, c.name, c.description,
		       (SELECT json_group_array(action_type)
		        FROM (SELECT DISTINCT action_type FROM connector_actions WHERE connector_id = c.id ORDER BY action_type)),
		       (SELECT json_group_array(service)
		        FROM (SELECT DISTINCT service FROM connector_required_credentials WHERE connector_id = c.id ORDER BY service)),
		       ac.enabled_at
		FROM agent_connectors ac
		JOIN connectors c ON c.id = ac.connector_id
		WHERE ac.agent_id = $1 AND ac.approver_id = $2
		GROUP BY c.id, c.name, c.description, ac.enabled_at
		ORDER BY ac.enabled_at DESC`,
		agentID, approverID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []AgentConnector
	for rows.Next() {
		var ac AgentConnector
		var actionsJSON, credsJSON []byte
		var enabledAt sql.NullString
		if err := rows.Scan(&ac.ID, &ac.Name, &ac.Description, &actionsJSON, &credsJSON, &enabledAt); err != nil {
			return nil, err
		}
		var tErr error
		ac.EnabledAt, tErr = sqliteTimeRequired(enabledAt)
		if tErr != nil {
			return nil, tErr
		}
		if len(actionsJSON) > 0 && string(actionsJSON) != "null" {
			if err := json.Unmarshal(actionsJSON, &ac.Actions); err != nil {
				return nil, fmt.Errorf("unmarshal actions for connector %s: %w", ac.ID, err)
			}
		}
		if len(credsJSON) > 0 && string(credsJSON) != "null" {
			if err := json.Unmarshal(credsJSON, &ac.RequiredCredentials); err != nil {
				return nil, fmt.Errorf("unmarshal credentials for connector %s: %w", ac.ID, err)
			}
		}
		result = append(result, ac)
	}
	return result, rows.Err()
}

func readEnabledAgentConnectorRow(ctx context.Context, db DBTX, agentID int64, approverID, connectorID string) (*AgentConnectorRow, error) {
	var row AgentConnectorRow
	var enabledAt sql.NullString
	err := db.QueryRow(ctx, `
		SELECT ac.agent_id, ac.connector_id, ac.enabled_at
		FROM agent_connectors ac
		WHERE ac.agent_id = $1 AND ac.approver_id = $2 AND ac.connector_id = $3
		ORDER BY ac.is_default DESC, ac.enabled_at ASC
		LIMIT 1`,
		agentID, approverID, connectorID,
	).Scan(&row.AgentID, &row.ConnectorID, &enabledAt)
	if err != nil {
		return nil, err
	}
	var err2 error
	row.EnabledAt, err2 = sqliteTimeRequired(enabledAt)
	if err2 != nil {
		return nil, err2
	}
	return &row, nil
}

// EnableAgentConnector idempotently enables a connector for an agent.
// The INSERT is guarded by an agent ownership check: if the agent does not
// belong to the approver, no row is inserted and AgentConnectorErrAgentNotFound
// is returned. If the connector does not exist, the FK violation is mapped to
// AgentConnectorErrConnectorNotFound.
//
// SQLite does not support INSERT/DELETE as nested WITH common table expressions
// (unlike PostgreSQL), so we use INSERT…RETURNING and fall back to a read when
// no row was inserted.
func EnableAgentConnector(ctx context.Context, db DBTX, agentID int64, approverID string, connectorID string) (*AgentConnectorRow, error) {
	instID := uuid.NewString()
	var row AgentConnectorRow
	var enabledAt sql.NullString
	err := db.QueryRow(ctx, `
		INSERT INTO agent_connectors (agent_id, approver_id, connector_id, connector_instance_id, is_default)
		SELECT $1, $2, $3, $4, 1
		WHERE EXISTS (SELECT 1 FROM agents WHERE agent_id = $1 AND approver_id = $2)
		  AND NOT EXISTS (
		      SELECT 1 FROM agent_connectors ac0
		      WHERE ac0.agent_id = $1 AND ac0.approver_id = $2 AND ac0.connector_id = $3
		  )
		RETURNING agent_id, connector_id, enabled_at`,
		agentID, approverID, connectorID, instID,
	).Scan(&row.AgentID, &row.ConnectorID, &enabledAt)
	if err != nil {
		if IsForeignKeyViolation(err) {
			return nil, &AgentConnectorError{Code: AgentConnectorErrConnectorNotFound}
		}
		if IsUniqueViolation(err) {
			return readEnabledAgentConnectorRow(ctx, db, agentID, approverID, connectorID)
		}
		if errors.Is(err, sql.ErrNoRows) {
			already, errExists := AgentConnectorEnabled(ctx, db, agentID, approverID, connectorID)
			if errExists != nil {
				return nil, errExists
			}
			if already {
				return readEnabledAgentConnectorRow(ctx, db, agentID, approverID, connectorID)
			}
			return nil, &AgentConnectorError{Code: AgentConnectorErrAgentNotFound}
		}
		return nil, err
	}
	var parseErr error
	row.EnabledAt, parseErr = sqliteTimeRequired(enabledAt)
	if parseErr != nil {
		return nil, parseErr
	}
	return &row, nil
}

// DisableAgentConnectorResult holds the result of disabling a connector.
type DisableAgentConnectorResult struct {
	AgentID                  int64
	ConnectorID              string
	DisabledAt               time.Time
	RevokedStandingApprovals int64
}

// DisableAgentConnector removes a connector from an agent and atomically revokes
// any active standing approvals whose action_type belongs to the disabled connector.
// Returns nil if the agent-connector association was not found.
func DisableAgentConnector(ctx context.Context, db DBTX, agentID int64, approverID string, connectorID string) (*DisableAgentConnectorResult, error) {
	row := db.QueryRow(ctx, `
		DELETE FROM agent_connectors
		WHERE agent_id = $1 AND approver_id = $2 AND connector_id = $3
		RETURNING agent_id, connector_id, strftime('%Y-%m-%dT%H:%M:%fZ', 'now') AS disabled_at`,
		agentID, approverID, connectorID,
	)

	var result DisableAgentConnectorResult
	var disabledAt sql.NullString
	err := row.Scan(&result.AgentID, &result.ConnectorID, &disabledAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	var parseErr error
	result.DisabledAt, parseErr = sqliteTimeRequired(disabledAt)
	if parseErr != nil {
		return nil, parseErr
	}

	tag, err := db.Exec(ctx, `
		UPDATE standing_approvals
		SET status = 'revoked', revoked_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
		WHERE agent_id = $1
		  AND user_id = $2
		  AND status = 'active'
		  AND action_type IN (
		      SELECT action_type FROM connector_actions
		      WHERE connector_id = $3
		  )`,
		agentID, approverID, connectorID,
	)
	if err != nil {
		return nil, err
	}
	result.RevokedStandingApprovals = RowsAffected(tag)
	return &result, nil
}
