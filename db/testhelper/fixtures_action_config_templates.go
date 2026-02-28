package testhelper

import (
	"testing"

	"github.com/supersuit-tech/permission-slip-web/db"
)

// InsertActionConfigTemplate creates an action configuration template with minimal defaults.
// The connector and connector_action must already exist.
func InsertActionConfigTemplate(t *testing.T, d db.DBTX, templateID, connectorID, actionType, name string) {
	t.Helper()
	mustExec(t, d,
		`INSERT INTO action_config_templates (id, connector_id, action_type, name, parameters)
		 VALUES ($1, $2, $3, $4, '{}')`,
		templateID, connectorID, actionType, name)
}

// ActionConfigTemplateOpts holds optional fields for InsertActionConfigTemplateFull.
type ActionConfigTemplateOpts struct {
	Description *string
	Parameters  []byte // raw JSON, defaults to '{}'
}

// InsertActionConfigTemplateFull creates an action configuration template with full control over all fields.
func InsertActionConfigTemplateFull(t *testing.T, d db.DBTX, templateID, connectorID, actionType, name string, opts ActionConfigTemplateOpts) {
	t.Helper()
	params := opts.Parameters
	if params == nil {
		params = []byte("{}")
	}
	mustExec(t, d,
		`INSERT INTO action_config_templates (id, connector_id, action_type, name, description, parameters)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		templateID, connectorID, actionType, name, opts.Description, params)
}
