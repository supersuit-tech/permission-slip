package db_test

import (
	"context"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip/db"
	"github.com/supersuit-tech/permission-slip/db/testhelper"
)

func TestComputeActionFingerprintStable(t *testing.T) {
	t.Parallel()
	action := []byte(`{"type":"email.send","parameters":{"to":"alice@example.com"}}`)
	a := db.ComputeActionFingerprint(42, "approver-1", action)
	b := db.ComputeActionFingerprint(42, "approver-1", action)
	if a != b {
		t.Fatalf("expected stable fingerprint, got %q vs %q", a, b)
	}
	if a == db.ComputeActionFingerprint(43, "approver-1", action) {
		t.Fatal("expected different fingerprint for different agent")
	}
}

func TestFindRecentDeniedApproval(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	agentID := testhelper.InsertAgent(t, tx, uid)

	action := []byte(`{"type":"email.send","parameters":{"to":"alice@example.com"}}`)
	fingerprint := db.ComputeActionFingerprint(agentID, uid, action)
	approvalID := testhelper.GenerateID(t, "appr_")

	_, err := tx.Exec(context.Background(),
		`INSERT INTO approvals (approval_id, agent_id, approver_id, action, context, status, action_fingerprint, expires_at, denied_at, denial_reason)
		 VALUES ($1, $2, $3, $4, '{"description":"test"}', 'denied', $5, strftime('%Y-%m-%dT%H:%M:%fZ', 'now', '+1 hour'), strftime('%Y-%m-%dT%H:%M:%fZ', 'now'), 'No')`,
		approvalID, agentID, uid, action, fingerprint,
	)
	if err != nil {
		t.Fatalf("insert denied approval: %v", err)
	}

	found, err := db.FindRecentDeniedApproval(context.Background(), tx, agentID, uid, fingerprint, time.Now().Add(-time.Hour))
	if err != nil {
		t.Fatalf("FindRecentDeniedApproval: %v", err)
	}
	if found == nil || found.ApprovalID != approvalID {
		t.Fatalf("expected recent denial %q, got %+v", approvalID, found)
	}

	outsideWindow, err := db.FindRecentDeniedApproval(context.Background(), tx, agentID, uid, fingerprint, time.Now().Add(time.Minute))
	if err != nil {
		t.Fatalf("FindRecentDeniedApproval outside window: %v", err)
	}
	if outsideWindow != nil {
		t.Fatalf("expected nil outside cooldown window, got %+v", outsideWindow)
	}
}
