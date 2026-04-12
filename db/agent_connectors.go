package db

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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
	AgentConnectorErrAgentNotFound     AgentConnectorErrCode = "agent_not_found"
	AgentConnectorErrConnectorNotFound AgentConnectorErrCode = "connector_not_found"
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
		       COALESCE(array_agg(DISTINCT ca.action_type ORDER BY ca.action_type) FILTER (WHERE ca.action_type IS NOT NULL), '{}'),
		       COALESCE(array_agg(DISTINCT crc.service ORDER BY crc.service) FILTER (WHERE crc.service IS NOT NULL), '{}'),
		       ac.enabled_at
		FROM agent_connectors ac
		JOIN connectors c ON c.id = ac.connector_id
		LEFT JOIN connector_actions ca ON ca.connector_id = c.id
		LEFT JOIN connector_required_credentials crc ON crc.connector_id = c.id
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
		if err := rows.Scan(&ac.ID, &ac.Name, &ac.Description, &ac.Actions, &ac.RequiredCredentials, &ac.EnabledAt); err != nil {
			return nil, err
		}
		result = append(result, ac)
	}
	return result, rows.Err()
}

// EnableAgentConnector idempotently enables a connector for an agent.
// The INSERT is guarded by an agent ownership check: if the agent does not
// belong to the approver, no row is inserted and AgentConnectorErrAgentNotFound
// is returned. If the connector does not exist, the FK violation is mapped to
// AgentConnectorErrConnectorNotFound.
func EnableAgentConnector(ctx context.Context, db DBTX, agentID int64, approverID string, connectorID string) (*AgentConnectorRow, error) {
	var row AgentConnectorRow
	err := db.QueryRow(ctx, `
		WITH agent_check AS (
			SELECT 1 FROM agents WHERE agent_id = $1 AND approver_id = $2
		), inserted AS (
			INSERT INTO agent_connectors (agent_id, approver_id, connector_id)
			SELECT $1, $2, $3
			WHERE EXISTS (SELECT 1 FROM agent_check)
			ON CONFLICT (agent_id, connector_id) DO NOTHING
			RETURNING agent_id, connector_id, enabled_at
		)
		SELECT agent_id, connector_id, enabled_at FROM inserted
		UNION ALL
		SELECT ac.agent_id, ac.connector_id, ac.enabled_at
		FROM agent_connectors ac
		WHERE ac.agent_id = $1 AND ac.connector_id = $3
		  AND NOT EXISTS (SELECT 1 FROM inserted)
		  AND EXISTS (SELECT 1 FROM agent_check)`,
		agentID, approverID, connectorID,
	).Scan(&row.AgentID, &row.ConnectorID, &row.EnabledAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, &AgentConnectorError{Code: AgentConnectorErrAgentNotFound}
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == PgCodeForeignKeyViolation {
			return nil, &AgentConnectorError{Code: AgentConnectorErrConnectorNotFound}
		}
		return nil, err
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
		WITH deleted AS (
			DELETE FROM agent_connectors
			WHERE agent_id = $1 AND approver_id = $2 AND connector_id = $3
			RETURNING agent_id, connector_id
		), revoked AS (
			UPDATE standing_approvals
			SET status = 'revoked', revoked_at = now()
			WHERE agent_id = $1
			  AND user_id = $2
			  AND status = 'active'
			  AND action_type IN (
			      SELECT action_type FROM connector_actions
			      WHERE connector_id = $3
			  )
			  AND EXISTS (SELECT 1 FROM deleted)
			RETURNING 1
		)
		SELECT
			(SELECT agent_id FROM deleted),
			(SELECT connector_id FROM deleted),
			now(),
			(SELECT count(*) FROM revoked)`,
		agentID, approverID, connectorID,
	)

	var agentIDResult *int64
	var connectorIDResult *string
	var result DisableAgentConnectorResult
	err := row.Scan(&agentIDResult, &connectorIDResult, &result.DisabledAt, &result.RevokedStandingApprovals)
	if err != nil {
		return nil, err
	}
	// If the DELETE matched nothing, agent_id/connector_id will be NULL.
	if agentIDResult == nil {
		return nil, nil
	}

	result.AgentID = *agentIDResult
	result.ConnectorID = *connectorIDResult
	return &result, nil
}
