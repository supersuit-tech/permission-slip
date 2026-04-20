package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// AgentConnectorCredential represents a row from the agent_connector_credentials
// join table. Each row binds exactly one credential (either a static credential
// or an OAuth connection) to an agent+connector pair.
type AgentConnectorCredential struct {
	ID                  string
	AgentID             int64
	ConnectorID         string
	ConnectorInstanceID string
	ApproverID          string
	CredentialID        *string
	OAuthConnectionID   *string
	CreatedAt           time.Time
}

// GetAgentConnectorCredential returns the credential binding for the default
// agent+connector instance, or nil if no binding exists.
//
// Deprecated: prefer GetAgentConnectorCredentialByInstance when the instance is known.
func GetAgentConnectorCredential(ctx context.Context, db DBTX, agentID int64, connectorID string) (*AgentConnectorCredential, error) {
	defaultInst, err := GetDefaultAgentConnectorInstanceByAgent(ctx, db, agentID, connectorID)
	if err != nil {
		return nil, err
	}
	if defaultInst == nil {
		return nil, nil
	}
	return GetAgentConnectorCredentialByInstance(ctx, db, agentID, connectorID, defaultInst.ConnectorInstanceID)
}

// GetAgentConnectorCredentialByInstance returns the credential binding for a specific connector instance.
func GetAgentConnectorCredentialByInstance(ctx context.Context, db DBTX, agentID int64, connectorID, connectorInstanceID string) (*AgentConnectorCredential, error) {
	var acc AgentConnectorCredential
	err := db.QueryRow(ctx, `
		SELECT id, agent_id, connector_id, connector_instance_id, approver_id,
		       credential_id, oauth_connection_id, created_at
		FROM agent_connector_credentials
		WHERE agent_id = $1 AND connector_id = $2 AND connector_instance_id = $3::uuid`,
		agentID, connectorID, connectorInstanceID,
	).Scan(&acc.ID, &acc.AgentID, &acc.ConnectorID, &acc.ConnectorInstanceID, &acc.ApproverID,
		&acc.CredentialID, &acc.OAuthConnectionID, &acc.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &acc, nil
}

// UpsertAgentConnectorCredentialParams holds the parameters for upserting a
// credential binding. Exactly one of CredentialID or OAuthConnectionID must
// be non-nil.
type UpsertAgentConnectorCredentialParams struct {
	ID                string
	AgentID           int64
	ConnectorID       string
	ApproverID        string
	CredentialID      *string
	OAuthConnectionID *string
}

// UpsertAgentConnectorCredential creates or replaces the credential binding
// for the default agent+connector instance.
func UpsertAgentConnectorCredential(ctx context.Context, db DBTX, p UpsertAgentConnectorCredentialParams) (*AgentConnectorCredential, error) {
	defaultInst, err := GetDefaultAgentConnectorInstance(ctx, db, p.AgentID, p.ApproverID, p.ConnectorID)
	if err != nil {
		return nil, err
	}
	if defaultInst == nil {
		return nil, fmt.Errorf("connector not enabled for agent: no default instance")
	}
	return UpsertAgentConnectorCredentialByInstance(ctx, db, UpsertAgentConnectorCredentialByInstanceParams{
		ID:                  p.ID,
		AgentID:             p.AgentID,
		ConnectorID:         p.ConnectorID,
		ConnectorInstanceID: defaultInst.ConnectorInstanceID,
		ApproverID:          p.ApproverID,
		CredentialID:        p.CredentialID,
		OAuthConnectionID:   p.OAuthConnectionID,
	})
}

// UpsertAgentConnectorCredentialByInstanceParams holds parameters for upserting a credential on a specific instance.
type UpsertAgentConnectorCredentialByInstanceParams struct {
	ID                  string
	AgentID             int64
	ConnectorID         string
	ConnectorInstanceID string
	ApproverID          string
	CredentialID        *string
	OAuthConnectionID   *string
}

// UpsertAgentConnectorCredentialByInstance creates or replaces the credential binding for a specific instance.
func UpsertAgentConnectorCredentialByInstance(ctx context.Context, db DBTX, p UpsertAgentConnectorCredentialByInstanceParams) (*AgentConnectorCredential, error) {
	var acc AgentConnectorCredential
	err := db.QueryRow(ctx, `
		INSERT INTO agent_connector_credentials
		    (id, agent_id, connector_id, connector_instance_id, approver_id, credential_id, oauth_connection_id)
		VALUES ($1, $2, $3, $4::uuid, $5, $6, $7)
		ON CONFLICT (agent_id, connector_id, connector_instance_id) DO UPDATE
		    SET credential_id = EXCLUDED.credential_id,
		        oauth_connection_id = EXCLUDED.oauth_connection_id,
		        approver_id = EXCLUDED.approver_id
		RETURNING id, agent_id, connector_id, connector_instance_id, approver_id,
		          credential_id, oauth_connection_id, created_at`,
		p.ID, p.AgentID, p.ConnectorID, p.ConnectorInstanceID, p.ApproverID,
		p.CredentialID, p.OAuthConnectionID,
	).Scan(&acc.ID, &acc.AgentID, &acc.ConnectorID, &acc.ConnectorInstanceID, &acc.ApproverID,
		&acc.CredentialID, &acc.OAuthConnectionID, &acc.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &acc, nil
}

// DeleteAgentConnectorCredential removes the credential binding for the default instance.
func DeleteAgentConnectorCredential(ctx context.Context, db DBTX, agentID int64, approverID string, connectorID string) (bool, error) {
	defaultInst, err := GetDefaultAgentConnectorInstance(ctx, db, agentID, approverID, connectorID)
	if err != nil {
		return false, err
	}
	if defaultInst == nil {
		return false, nil
	}
	return DeleteAgentConnectorCredentialByInstance(ctx, db, agentID, approverID, connectorID, defaultInst.ConnectorInstanceID)
}

// DeleteAgentConnectorCredentialByInstance removes the credential binding for a specific instance.
func DeleteAgentConnectorCredentialByInstance(ctx context.Context, db DBTX, agentID int64, approverID, connectorID, connectorInstanceID string) (bool, error) {
	tag, err := db.Exec(ctx, `
		DELETE FROM agent_connector_credentials
		WHERE agent_id = $1 AND approver_id = $2 AND connector_id = $3 AND connector_instance_id = $4::uuid`,
		agentID, approverID, connectorID, connectorInstanceID,
	)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// ListAgentConnectorCredentialsForAgentConnector returns all credential bindings for every
// instance of this connector on the agent (used when disabling with delete_credentials).
func ListAgentConnectorCredentialsForAgentConnector(ctx context.Context, db DBTX, agentID int64, approverID, connectorID string) ([]AgentConnectorCredential, error) {
	rows, err := db.Query(ctx, `
		SELECT id, agent_id, connector_id, connector_instance_id, approver_id,
		       credential_id, oauth_connection_id, created_at
		FROM agent_connector_credentials
		WHERE agent_id = $1 AND approver_id = $2 AND connector_id = $3`,
		agentID, approverID, connectorID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []AgentConnectorCredential
	for rows.Next() {
		var acc AgentConnectorCredential
		if err := rows.Scan(&acc.ID, &acc.AgentID, &acc.ConnectorID, &acc.ConnectorInstanceID, &acc.ApproverID,
			&acc.CredentialID, &acc.OAuthConnectionID, &acc.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, acc)
	}
	return out, rows.Err()
}
