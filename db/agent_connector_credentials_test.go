package db_test

import (
	"testing"

	"github.com/supersuit-tech/permission-slip/db/testhelper"
)

func TestAgentConnectorCredentialsSchema(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	testhelper.RequireColumns(t, tx, "agent_connector_credentials", []string{
		"id", "agent_id", "connector_id", "approver_id", "connector_instance_id",
		"credential_id", "oauth_connection_id", "created_at",
	})
}
