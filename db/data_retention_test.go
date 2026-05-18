package db_test

import (
	"context"
	"testing"

	"github.com/supersuit-tech/permission-slip/db"
	"github.com/supersuit-tech/permission-slip/db/testhelper"
)

func TestPurgeExpiredAuditEvents(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	cleanStaleAuditEvents := func(t *testing.T, tx db.DBTX) {
		t.Helper()
		testhelper.MustExec(t, tx, `DELETE FROM audit_events`)
	}

	t.Run("PurgesEventsOlderThanRetention", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)
		cleanStaleAuditEvents(t, tx)

		uid := testhelper.GenerateUID(t)
		agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

		testhelper.MustExec(t, tx,
			`INSERT INTO audit_events (user_id, agent_id, event_type, outcome, source_id, source_type, agent_meta, created_at)
			 VALUES ($1, $2, 'approval.approved', 'approved', 'test_old', 'approval', '{}', strftime('%Y-%m-%dT%H:%M:%fZ', 'now', '-10 days'))`,
			uid, agentID)

		testhelper.InsertAuditEvent(t, tx, uid, agentID, "approval.denied", "denied", testhelper.GenerateID(t, "appr_"))

		deleted, err := db.PurgeExpiredAuditEvents(ctx, tx, 7)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if deleted != 1 {
			t.Errorf("expected 1 deleted row, got %d", deleted)
		}

		page, err := db.ListAuditEvents(ctx, tx, uid, 20, nil, nil, 0)
		if err != nil {
			t.Fatalf("list error: %v", err)
		}
		if len(page.Events) != 1 {
			t.Errorf("expected 1 remaining event, got %d", len(page.Events))
		}
	})

	t.Run("PreservesEventsWithinRetention", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)
		cleanStaleAuditEvents(t, tx)

		uid := testhelper.GenerateUID(t)
		agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

		testhelper.MustExec(t, tx,
			`INSERT INTO audit_events (user_id, agent_id, event_type, outcome, source_id, source_type, agent_meta, created_at)
			 VALUES ($1, $2, 'approval.approved', 'approved', 'test_30d', 'approval', '{}', strftime('%Y-%m-%dT%H:%M:%fZ', 'now', '-30 days'))`,
			uid, agentID)

		deleted, err := db.PurgeExpiredAuditEvents(ctx, tx, 90)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if deleted != 0 {
			t.Errorf("expected 0 deleted rows, got %d", deleted)
		}
	})

	t.Run("PurgesVeryOldEventsFor90DayWindow", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)
		cleanStaleAuditEvents(t, tx)

		uid := testhelper.GenerateUID(t)
		agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

		testhelper.MustExec(t, tx,
			`INSERT INTO audit_events (user_id, agent_id, event_type, outcome, source_id, source_type, agent_meta, created_at)
			 VALUES ($1, $2, 'approval.approved', 'approved', 'test_100d', 'approval', '{}', strftime('%Y-%m-%dT%H:%M:%fZ', 'now', '-100 days'))`,
			uid, agentID)

		deleted, err := db.PurgeExpiredAuditEvents(ctx, tx, 90)
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
		testhelper.InsertAuditEvent(t, tx, uid, agentID, "approval.approved", "approved", testhelper.GenerateID(t, "appr_"))

		err := db.DeleteAccount(ctx, tx, uid, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		profile, err := db.GetProfileByUserID(ctx, tx, uid)
		if err != nil {
			t.Fatalf("profile lookup error: %v", err)
		}
		if profile != nil {
			t.Error("expected profile to be deleted")
		}

		page, err := db.ListAuditEvents(ctx, tx, uid, 20, nil, nil, 0)
		if err != nil {
			t.Fatalf("audit events error: %v", err)
		}
		if len(page.Events) != 0 {
			t.Errorf("expected 0 events after deletion, got %d", len(page.Events))
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

func TestPurgeExpiredAuditEvents_PgCronJob(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	testhelper.RequirePgCronJob(t, tx, "purge_expired_audit_events")
}

func TestAuditRetentionDaysFromEnv(t *testing.T) {
	t.Setenv("AUDIT_RETENTION_DAYS", "45")
	if got := db.AuditRetentionDaysFromEnv(); got != 45 {
		t.Errorf("got %d, want 45", got)
	}
	t.Setenv("AUDIT_RETENTION_DAYS", "")
	if got := db.AuditRetentionDaysFromEnv(); got != 90 {
		t.Errorf("default got %d, want 90", got)
	}
}
