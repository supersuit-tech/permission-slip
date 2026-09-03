package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// AgentWebhookConfig holds wake webhook settings for an agent.
type AgentWebhookConfig struct {
	AgentID             int64
	WebhookURL          *string
	WebhookTokenVaultID *string
	WebhookProvider     string
}

// GetAgentWebhookConfig returns webhook URL, provider, and vault token ID for the agent.
func GetAgentWebhookConfig(ctx context.Context, db DBTX, agentID int64) (*AgentWebhookConfig, error) {
	var url, vaultID, provider sql.NullString
	err := db.QueryRow(ctx,
		`SELECT webhook_url, webhook_token_vault_id, webhook_provider FROM agents WHERE agent_id = $1`,
		agentID,
	).Scan(&url, &vaultID, &provider)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	cfg := &AgentWebhookConfig{AgentID: agentID}
	if url.Valid && url.String != "" {
		s := url.String
		cfg.WebhookURL = &s
	}
	if vaultID.Valid && vaultID.String != "" {
		s := vaultID.String
		cfg.WebhookTokenVaultID = &s
	}
	if provider.Valid && provider.String != "" {
		cfg.WebhookProvider = provider.String
	} else {
		cfg.WebhookProvider = "openclaw"
	}
	return cfg, nil
}

// SetAgentWebhook stores webhook URL, provider, and vault token ID for an agent.
func SetAgentWebhook(ctx context.Context, db DBTX, agentID int64, webhookURL, tokenVaultID, provider string) error {
	if provider == "" {
		provider = "openclaw"
	}
	_, err := db.Exec(ctx,
		`UPDATE agents SET webhook_url = $2, webhook_token_vault_id = $3, webhook_provider = $4 WHERE agent_id = $1`,
		agentID, webhookURL, tokenVaultID, provider,
	)
	return err
}

// ClearAgentWebhook removes webhook configuration from an agent.
// Returns the previous token vault ID so the caller can delete the vault secret.
func ClearAgentWebhook(ctx context.Context, db DBTX, agentID int64) (*string, error) {
	cfg, err := GetAgentWebhookConfig(ctx, db, agentID)
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		return nil, fmt.Errorf("agent not found")
	}
	prevVaultID := cfg.WebhookTokenVaultID
	_, err = db.Exec(ctx,
		`UPDATE agents SET webhook_url = NULL, webhook_token_vault_id = NULL, webhook_provider = 'openclaw' WHERE agent_id = $1`,
		agentID,
	)
	if err != nil {
		return nil, err
	}
	return prevVaultID, nil
}

// WebhookURLSharedByOtherAgent reports whether another agent belonging to the
// same approver has the same webhook URL (trailing slashes normalized).
func WebhookURLSharedByOtherAgent(ctx context.Context, db DBTX, agentID int64, webhookURL string) (bool, error) {
	var exists bool
	err := db.QueryRow(ctx,
		`SELECT EXISTS(
			SELECT 1 FROM agents other
			INNER JOIN agents self ON self.approver_id = other.approver_id
			WHERE self.agent_id = $1
			  AND other.agent_id != $1
			  AND other.webhook_url IS NOT NULL
			  AND other.webhook_url != ''
			  AND rtrim(other.webhook_url, '/') = rtrim($2, '/')
		)`,
		agentID, webhookURL,
	).Scan(&exists)
	return exists, err
}

// AgentHasWebhook reports whether the agent has a webhook URL configured.
func AgentHasWebhook(ctx context.Context, db DBTX, agentID int64) (bool, error) {
	var exists bool
	err := db.QueryRow(ctx,
		`SELECT EXISTS(
			SELECT 1 FROM agents
			WHERE agent_id = $1 AND webhook_url IS NOT NULL AND webhook_url != ''
		)`,
		agentID,
	).Scan(&exists)
	return exists, err
}
