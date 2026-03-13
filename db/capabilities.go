package db

import (
	"context"
	"encoding/json"
	"time"
)

// AgentCapabilities holds all data needed to build the capabilities response
// for an authenticated agent. The caller assembles ConnectorCapability,
// ActionCapability, StandingApprovalCapability, and CapabilityActionConfig
// structs from these rows.
type AgentCapabilities struct {
	Connectors        []CapabilityConnector
	Actions           []CapabilityAction
	StandingApprovals []CapabilityStandingApproval
	ActionConfigs     []CapabilityActionConfig
}

// CapabilityConnector represents an enabled connector with credential readiness.
type CapabilityConnector struct {
	ID               string
	Name             string
	Description      *string
	CredentialsReady bool
}

// CapabilityAction represents an action available through an enabled connector.
type CapabilityAction struct {
	ConnectorID      string
	ActionType       string
	Name             string
	Description      *string
	RiskLevel        *string
	ParametersSchema json.RawMessage // raw JSONB
}

// CapabilityActionConfig represents an active action configuration for an agent.
// Each configuration defines a pre-approved parameter set that the agent can
// reference when requesting approval or executing actions.
type CapabilityActionConfig struct {
	ConfigurationID string
	ConnectorID     string
	ActionType      string
	Name            string
	Description     *string
	Parameters      json.RawMessage // includes wildcards ("*")
}

// CapabilityStandingApproval represents an active, non-expired standing approval.
type CapabilityStandingApproval struct {
	StandingApprovalID  string
	ActionType          string
	Constraints         json.RawMessage // raw JSONB
	MaxExecutions       *int
	ExecutionsRemaining *int
	ExpiresAt           time.Time
}

// GetAgentCapabilities retrieves all data needed for the capabilities endpoint:
// enabled connectors, their actions, credential readiness, and active standing
// approvals. It runs three focused queries within the same connection.
//
// The caller (API handler) is responsible for verifying agent ownership and
// status before calling this function.
func GetAgentCapabilities(ctx context.Context, db DBTX, agentID int64, approverID string) (*AgentCapabilities, error) {
	caps := &AgentCapabilities{}

	// 1. Enabled connectors with credential readiness.
	//
	// For each enabled connector, check if ALL required credential services
	// have a matching row in the credentials table for this user. If a
	// connector has no required credentials, it's considered ready.
	connRows, err := db.Query(ctx, `
		SELECT c.id, c.name, c.description,
		       NOT EXISTS (
		           SELECT 1 FROM connector_required_credentials crc
		           WHERE crc.connector_id = c.id
		             AND NOT EXISTS (
		               SELECT 1 FROM credentials cr
		               WHERE cr.user_id = ac.approver_id
		                 AND cr.service = crc.service
		             )
		       ) AS credentials_ready
		FROM agent_connectors ac
		JOIN connectors c ON c.id = ac.connector_id
		WHERE ac.agent_id = $1 AND ac.approver_id = $2
		ORDER BY c.id`,
		agentID, approverID,
	)
	if err != nil {
		return nil, err
	}
	defer connRows.Close()

	var connectorIDs []string
	for connRows.Next() {
		var cc CapabilityConnector
		if err := connRows.Scan(&cc.ID, &cc.Name, &cc.Description, &cc.CredentialsReady); err != nil {
			return nil, err
		}
		caps.Connectors = append(caps.Connectors, cc)
		connectorIDs = append(connectorIDs, cc.ID)
	}
	if err := connRows.Err(); err != nil {
		return nil, err
	}

	// Short-circuit: no connectors means no actions or standing approvals.
	if len(connectorIDs) == 0 {
		return caps, nil
	}

	// 2. Actions for enabled connectors.
	actionRows, err := db.Query(ctx, `
		SELECT ca.connector_id, ca.action_type, ca.name, ca.description,
		       ca.risk_level, ca.parameters_schema
		FROM connector_actions ca
		WHERE ca.connector_id = ANY($1)
		ORDER BY ca.connector_id, ca.action_type`,
		connectorIDs,
	)
	if err != nil {
		return nil, err
	}
	defer actionRows.Close()

	for actionRows.Next() {
		var a CapabilityAction
		if err := actionRows.Scan(&a.ConnectorID, &a.ActionType, &a.Name, &a.Description, &a.RiskLevel, &a.ParametersSchema); err != nil {
			return nil, err
		}
		caps.Actions = append(caps.Actions, a)
	}
	if err := actionRows.Err(); err != nil {
		return nil, err
	}

	// 3. Active, non-expired standing approvals for this agent.
	saRows, err := db.Query(ctx, `
		SELECT sa.standing_approval_id, sa.action_type, sa.constraints,
		       sa.max_executions,
		       CASE WHEN sa.max_executions IS NOT NULL
		            THEN sa.max_executions - sa.execution_count
		            ELSE NULL
		       END AS executions_remaining,
		       sa.expires_at
		FROM standing_approvals sa
		WHERE sa.agent_id = $1
		  AND sa.user_id = $2
		  AND sa.status = 'active'
		  AND sa.expires_at > now()
		  AND sa.starts_at <= now()
		ORDER BY sa.action_type, sa.expires_at`,
		agentID, approverID,
	)
	if err != nil {
		return nil, err
	}
	defer saRows.Close()

	for saRows.Next() {
		var sa CapabilityStandingApproval
		if err := saRows.Scan(&sa.StandingApprovalID, &sa.ActionType, &sa.Constraints, &sa.MaxExecutions, &sa.ExecutionsRemaining, &sa.ExpiresAt); err != nil {
			return nil, err
		}
		caps.StandingApprovals = append(caps.StandingApprovals, sa)
	}
	if err := saRows.Err(); err != nil {
		return nil, err
	}

	// 4. Active action configurations for this agent.
	//
	// Each configuration defines a pre-approved set of parameters (with
	// optional wildcards). The agent picks from these when requesting
	// approval or executing actions.
	acRows, err := db.Query(ctx, `
		SELECT ac.id, ac.connector_id, ac.action_type, ac.name, ac.description,
		       ac.parameters
		FROM action_configurations ac
		WHERE ac.agent_id = $1
		  AND ac.user_id = $2
		  AND ac.status = 'active'
		  AND ac.connector_id = ANY($3)
		ORDER BY ac.connector_id, ac.action_type, ac.created_at`,
		agentID, approverID, connectorIDs,
	)
	if err != nil {
		return nil, err
	}
	defer acRows.Close()

	for acRows.Next() {
		var ac CapabilityActionConfig
		if err := acRows.Scan(&ac.ConfigurationID, &ac.ConnectorID, &ac.ActionType,
			&ac.Name, &ac.Description, &ac.Parameters); err != nil {
			return nil, err
		}
		caps.ActionConfigs = append(caps.ActionConfigs, ac)
	}
	return caps, acRows.Err()
}
