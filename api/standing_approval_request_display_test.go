package api

import (
	"context"
	"testing"

	"github.com/supersuit-tech/permission-slip/db/testhelper"
)

func TestResolveStandingApprovalRequestDisplay_FromActionType(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	ctx := context.Background()

	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	connID := "protonmail"
	testhelper.InsertConnector(t, tx, connID)
	actionType := "protonmail.read_email"
	testhelper.InsertConnectorAction(t, tx, connID, actionType, "Read email")
	testhelper.InsertAgentConnector(t, tx, agentID, uid, connID)

	display := resolveStandingApprovalRequestDisplay(ctx, tx, agentID, uid, actionType)
	if display.ConnectorName == "" {
		t.Error("expected connector name")
	}
	if display.ConnectorInstanceDisplay != "" {
		t.Errorf("expected empty instance display without explicit selector, got %q", display.ConnectorInstanceDisplay)
	}
}
