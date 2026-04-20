package db

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

// AgentConnectorCredential represents a row from the agent_connector_credentials
// join table. Each row binds exactly one credential (either a static credential
// or an OAuth connection) to an agent+connector pair.
type AgentConnectorCredential struct {
	ID                string
	AgentID           int64
	ConnectorID       string
	ApproverID        string
	CredentialID      *string
	OAuthConnectionID *string
	CreatedAt         time.Time
}

// GetAgentConnectorCredential returns the credential binding for an
// agent+connector pair, or nil if no binding exists.
func GetAgentConnectorCredential(ctx context.Context, db DBTX, agentID int64, connectorID string) (*AgentConnectorCredential, error) {
	var acc AgentConnectorCredential
	err := db.QueryRow(ctx, `
		SELECT id, agent_id, connector_id, approver_id,
		       credential_id, oauth_connection_id, created_at
		FROM agent_connector_credentials
		WHERE agent_id = $1 AND connector_id = $2`,
		agentID, connectorID,
	).Scan(&acc.ID, &acc.AgentID, &acc.ConnectorID, &acc.ApproverID,
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
// for an agent+connector pair. The unique index on (agent_id, connector_id, connector_instance_id)
// ensures at most one binding per instance; omitted instance resolves to the default via trigger.
func UpsertAgentConnectorCredential(ctx context.Context, db DBTX, p UpsertAgentConnectorCredentialParams) (*AgentConnectorCredential, error) {
	var acc AgentConnectorCredential
	err := db.QueryRow(ctx, `
		INSERT INTO agent_connector_credentials
		    (id, agent_id, connector_id, approver_id, credential_id, oauth_connection_id)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (agent_id, connector_id, connector_instance_id) DO UPDATE
		    SET credential_id = EXCLUDED.credential_id,
		        oauth_connection_id = EXCLUDED.oauth_connection_id,
		        approver_id = EXCLUDED.approver_id
		RETURNING id, agent_id, connector_id, approver_id,
		          credential_id, oauth_connection_id, created_at`,
		p.ID, p.AgentID, p.ConnectorID, p.ApproverID,
		p.CredentialID, p.OAuthConnectionID,
	).Scan(&acc.ID, &acc.AgentID, &acc.ConnectorID, &acc.ApproverID,
		&acc.CredentialID, &acc.OAuthConnectionID, &acc.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &acc, nil
}

// DeleteAgentConnectorCredential removes the credential binding for an
// agent+connector pair. Returns true if a row was deleted, false if no
// binding existed.
func DeleteAgentConnectorCredential(ctx context.Context, db DBTX, agentID int64, approverID string, connectorID string) (bool, error) {
	tag, err := db.Exec(ctx, `
		DELETE FROM agent_connector_credentials
		WHERE agent_id = $1 AND approver_id = $2 AND connector_id = $3`,
		agentID, approverID, connectorID,
	)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}
