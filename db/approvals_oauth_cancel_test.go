package db_test

import (
	"context"
	"testing"

	"github.com/supersuit-tech/permission-slip/db"
	"github.com/supersuit-tech/permission-slip/db/testhelper"
)

func TestCancelPendingApprovalsForOAuthConnection(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	connID := "testgoogle"
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertConnectorAction(t, tx, connID, "testgoogle.send_email", "Send")

	agentID := testhelper.InsertAgentWithStatus(t, tx, uid, "registered")
	testhelper.InsertAgentConnector(t, tx, agentID, uid, connID)

	oauthID := testhelper.GenerateID(t, "oconn_")
	testhelper.InsertOAuthConnectionFull(t, tx, oauthID, uid, "google", testhelper.OAuthConnectionOpts{
		Status: "needs_reauth",
		Scopes: []string{},
	})
	conn, err := db.GetOAuthConnectionByProvider(context.Background(), tx, uid, "google")
	if err != nil {
		t.Fatalf("get oauth: %v", err)
	}

	bindingID := testhelper.GenerateID(t, "accr_")
	if _, err := db.UpsertAgentConnectorCredential(context.Background(), tx, db.UpsertAgentConnectorCredentialParams{
		ID: bindingID, AgentID: agentID, ConnectorID: connID,
		ApproverID: uid, OAuthConnectionID: &conn.ID,
	}); err != nil {
		t.Fatalf("bind: %v", err)
	}

	apprID := testhelper.GenerateID(t, "appr_")
	testhelper.InsertApprovalWithStatus(t, tx, apprID, agentID, uid, "pending")
	if _, err := tx.Exec(context.Background(),
		`UPDATE approvals SET action = $1 WHERE approval_id = $2`,
		`{"type":"testgoogle.send_email","parameters":{}}`, apprID); err != nil {
		t.Fatalf("update action: %v", err)
	}

	n, err := db.CancelPendingApprovalsForOAuthConnection(context.Background(), tx, conn.ID)
	if err != nil {
		t.Fatalf("cancel: %v", err)
	}
	if n != 1 {
		t.Fatalf("expected 1 cancelled, got %d", n)
	}

	stored, err := db.GetApprovalByIDAndApprover(context.Background(), tx, apprID, uid)
	if err != nil || stored == nil {
		t.Fatalf("get approval: %v", err)
	}
	if stored.Status != "cancelled" {
		t.Errorf("expected cancelled, got %q", stored.Status)
	}
}
