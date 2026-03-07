package testhelper

import (
	"testing"

	"github.com/supersuit-tech/permission-slip-web/db"
)

// InsertConnector creates a connector with a name matching the ID.
func InsertConnector(t *testing.T, d db.DBTX, connectorID string) {
	t.Helper()
	mustExec(t, d,
		`INSERT INTO connectors (id, name) VALUES ($1, $1)`,
		connectorID)
}

// InsertConnectorAction creates an action for the given connector.
// The connector must already exist via InsertConnector.
func InsertConnectorAction(t *testing.T, d db.DBTX, connectorID, actionType, name string) {
	t.Helper()
	mustExec(t, d,
		`INSERT INTO connector_actions (connector_id, action_type, name) VALUES ($1, $2, $3)`,
		connectorID, actionType, name)
}

// InsertConnectorActionWithPayment creates an action that requires a payment method.
func InsertConnectorActionWithPayment(t *testing.T, d db.DBTX, connectorID, actionType, name string) {
	t.Helper()
	mustExec(t, d,
		`INSERT INTO connector_actions (connector_id, action_type, name, requires_payment_method) VALUES ($1, $2, $3, true)`,
		connectorID, actionType, name)
}

// ConnectorActionOpts holds optional fields for InsertConnectorActionFull.
type ConnectorActionOpts struct {
	Description      *string
	RiskLevel        *string
	ParametersSchema []byte // raw JSON
}

// InsertConnectorActionFull creates an action with full details for the given connector.
func InsertConnectorActionFull(t *testing.T, d db.DBTX, connectorID, actionType, name string, opts ConnectorActionOpts) {
	t.Helper()
	mustExec(t, d,
		`INSERT INTO connector_actions (connector_id, action_type, name, description, risk_level, parameters_schema)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		connectorID, actionType, name, opts.Description, opts.RiskLevel, opts.ParametersSchema)
}

// InsertConnectorWithDescription creates a connector with a custom description.
func InsertConnectorWithDescription(t *testing.T, d db.DBTX, connectorID, name, description string) {
	t.Helper()
	mustExec(t, d,
		`INSERT INTO connectors (id, name, description) VALUES ($1, $2, $3)`,
		connectorID, name, description)
}

// InsertConnectorRequiredCredential creates a required credential entry for the given connector.
// The connector must already exist via InsertConnector.
func InsertConnectorRequiredCredential(t *testing.T, d db.DBTX, connectorID, service, authType string) {
	t.Helper()
	mustExec(t, d,
		`INSERT INTO connector_required_credentials (connector_id, service, auth_type) VALUES ($1, $2, $3)`,
		connectorID, service, authType)
}
