package db_test

import (
	"context"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/db"
	"github.com/supersuit-tech/permission-slip-web/db/testhelper"
)

func TestPurgeExpiredAuditEvents(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("PurgesOldEventsForFreePlan", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)

		uid := testhelper.GenerateUID(t)
		agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
		testhelper.InsertSubscription(t, tx, uid, db.PlanFree) // 7-day retention

		// Insert an event dated 10 days ago — should be purged.
		testhelper.MustExec(t, tx,
			`INSERT INTO audit_events (user_id, agent_id, event_type, outcome, source_id, source_type, agent_meta, created_at)
			 VALUES ($1, $2, 'approval.approved', 'approved', 'test_old', 'approval', '{}', now() - interval '10 days')`,
			uid, agentID)

		// Insert a recent event — should NOT be purged.
		testhelper.InsertAuditEvent(t, tx, uid, agentID, "approval.denied", "denied", testhelper.GenerateID(t, "appr_"))

		deleted, err := db.PurgeExpiredAuditEvents(ctx, tx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if deleted != 1 {
			t.Errorf("expected 1 deleted row, got %d", deleted)
		}

		// Verify the recent event still exists.
		page, err := db.ListAuditEvents(ctx, tx, uid, 20, nil, nil)
		if err != nil {
			t.Fatalf("list error: %v", err)
		}
		if len(page.Events) != 1 {
			t.Errorf("expected 1 remaining event, got %d", len(page.Events))
		}
	})

	t.Run("PreservesPaidPlanEvents", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)

		uid := testhelper.GenerateUID(t)
		agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
		testhelper.InsertSubscription(t, tx, uid, db.PlanPayAsYouGo) // 90-day retention

		// Insert an event dated 30 days ago — should NOT be purged (within 90 days).
		testhelper.MustExec(t, tx,
			`INSERT INTO audit_events (user_id, agent_id, event_type, outcome, source_id, source_type, agent_meta, created_at)
			 VALUES ($1, $2, 'approval.approved', 'approved', 'test_30d', 'approval', '{}', now() - interval '30 days')`,
			uid, agentID)

		deleted, err := db.PurgeExpiredAuditEvents(ctx, tx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if deleted != 0 {
			t.Errorf("expected 0 deleted rows, got %d", deleted)
		}
	})

	t.Run("PurgesOldPaidPlanEvents", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)

		uid := testhelper.GenerateUID(t)
		agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
		testhelper.InsertSubscription(t, tx, uid, db.PlanPayAsYouGo)

		// Insert event dated 100 days ago — should be purged (>90 days).
		testhelper.MustExec(t, tx,
			`INSERT INTO audit_events (user_id, agent_id, event_type, outcome, source_id, source_type, agent_meta, created_at)
			 VALUES ($1, $2, 'approval.approved', 'approved', 'test_100d', 'approval', '{}', now() - interval '100 days')`,
			uid, agentID)

		deleted, err := db.PurgeExpiredAuditEvents(ctx, tx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if deleted != 1 {
			t.Errorf("expected 1 deleted row, got %d", deleted)
		}
	})
}

func TestDeleteAccount(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("DeletesCascadingData", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)

		uid := testhelper.GenerateUID(t)
		agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
		testhelper.InsertSubscription(t, tx, uid, db.PlanFree)
		testhelper.InsertAuditEvent(t, tx, uid, agentID, "approval.approved", "approved", testhelper.GenerateID(t, "appr_"))

		err := db.DeleteAccount(ctx, tx, uid, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify profile is gone.
		profile, err := db.GetProfileByUserID(ctx, tx, uid)
		if err != nil {
			t.Fatalf("profile lookup error: %v", err)
		}
		if profile != nil {
			t.Error("expected profile to be deleted")
		}

		// Verify audit events are gone.
		page, err := db.ListAuditEvents(ctx, tx, uid, 20, nil, nil)
		if err != nil {
			t.Fatalf("audit events error: %v", err)
		}
		if len(page.Events) != 0 {
			t.Errorf("expected 0 events after deletion, got %d", len(page.Events))
		}

		// Verify subscription is gone.
		sub, err := db.GetSubscriptionByUserID(ctx, tx, uid)
		if err != nil {
			t.Fatalf("subscription lookup error: %v", err)
		}
		if sub != nil {
			t.Error("expected subscription to be deleted")
		}
	})

	t.Run("NotFoundReturnsError", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)

		fakeUID := testhelper.GenerateUID(t)
		err := db.DeleteAccount(ctx, tx, fakeUID, nil)
		if err == nil {
			t.Error("expected error for non-existent user")
		}
	})

	t.Run("CallsVaultDeleteForCredentials", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)

		uid := testhelper.GenerateUID(t)
		testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
		testhelper.InsertSubscription(t, tx, uid, db.PlanFree)

		// Insert a credential row with a fake vault_secret_id.
		fakeVaultID := testhelper.GenerateUID(t)
		credID := testhelper.GenerateID(t, "cred_")
		testhelper.MustExec(t, tx,
			`INSERT INTO credentials (id, user_id, service, vault_secret_id)
			 VALUES ($1, $2, 'test_service', $3)`,
			credID, uid, fakeVaultID)

		var deletedSecrets []string
		vaultDeleteFn := func(_ context.Context, _ db.DBTX, secretID string) error {
			deletedSecrets = append(deletedSecrets, secretID)
			return nil
		}

		err := db.DeleteAccount(ctx, tx, uid, vaultDeleteFn)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(deletedSecrets) != 1 || deletedSecrets[0] != fakeVaultID {
			t.Errorf("expected vault delete for %s, got %v", fakeVaultID, deletedSecrets)
		}
	})
}
