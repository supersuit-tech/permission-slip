package testhelper

import (
	"testing"

	"github.com/supersuit-tech/permission-slip/db"
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
	ParametersSchema []byte  // raw JSON
	OperationType    *string // read, write, edit, delete — defaults to write when nil
}

// InsertConnectorActionFull creates an action with full details for the given connector.
func InsertConnectorActionFull(t *testing.T, d db.DBTX, connectorID, actionType, name string, opts ConnectorActionOpts) {
	t.Helper()
	op := "write"
	if opts.OperationType != nil && *opts.OperationType != "" {
		op = *opts.OperationType
	}
	mustExec(t, d,
		`INSERT INTO connector_actions (connector_id, action_type, operation_type, name, description, risk_level, parameters_schema)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		connectorID, actionType, op, name, opts.Description, opts.RiskLevel, opts.ParametersSchema)
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

// InsertConnectorWithStaticCredential registers a minimal connector (one ping action)
// and a required static credential row for POST /v1/credentials validation tests.
// When fieldsJSON is non-nil, it is stored as credential_fields (JSONB); otherwise
// the column defaults to [].
func InsertConnectorWithStaticCredential(t *testing.T, d db.DBTX, connectorID, service, authType string, fieldsJSON []byte) {
	t.Helper()
	InsertConnector(t, d, connectorID)
	InsertConnectorAction(t, d, connectorID, connectorID+".ping", "Ping")
	if len(fieldsJSON) == 0 {
		mustExec(t, d,
			`INSERT INTO connector_required_credentials (connector_id, service, auth_type) VALUES ($1, $2, $3)`,
			connectorID, service, authType)
		return
	}
	mustExec(t, d,
		`INSERT INTO connector_required_credentials (connector_id, service, auth_type, credential_fields) VALUES ($1, $2, $3, $4::jsonb)`,
		connectorID, service, authType, fieldsJSON)
}
