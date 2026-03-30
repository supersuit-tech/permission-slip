package testhelper

import (
	"context"
	"testing"

	"github.com/supersuit-tech/permission-slip/db"
)

// InsertActionConfig creates an active action configuration with minimal defaults.
// The agent, user, connector, and connector_action must already exist.
func InsertActionConfig(t *testing.T, d db.DBTX, configID string, agentID int64, userID, connectorID, actionType string) {
	t.Helper()
	mustExec(t, d,
		`INSERT INTO action_configurations (id, agent_id, user_id, connector_id, action_type, parameters, name)
		 VALUES ($1, $2, $3, $4, $5, '{}', $1)`,
		configID, agentID, userID, connectorID, actionType)
}

// ActionConfigOpts holds optional fields for InsertActionConfigFull.
type ActionConfigOpts struct {
	Parameters  []byte // raw JSON, defaults to '{}'
	Status      string // defaults to 'active'
	Name        string // defaults to configID
	Description *string
}

// InsertActionConfigFull creates an action configuration with full control over all fields.
func InsertActionConfigFull(t *testing.T, d db.DBTX, configID string, agentID int64, userID, connectorID, actionType string, opts ActionConfigOpts) {
	t.Helper()
	params := opts.Parameters
	if params == nil {
		params = []byte("{}")
	}
	status := opts.Status
	if status == "" {
		status = "active"
	}
	name := opts.Name
	if name == "" {
		name = configID
	}

	_, err := d.Exec(context.Background(),
		`INSERT INTO action_configurations (id, agent_id, user_id, connector_id, action_type, parameters, status, name, description)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		configID, agentID, userID, connectorID, actionType, params, status, name, opts.Description)
	if err != nil {
		t.Fatalf("InsertActionConfigFull: %v", err)
	}
}
