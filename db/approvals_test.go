package db_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip-web/db"
	"github.com/supersuit-tech/permission-slip-web/db/testhelper"
)

func TestApprovalsSchema(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	t.Run("approvals", func(t *testing.T) {
		testhelper.RequireColumns(t, tx, "approvals", []string{
			"approval_id", "agent_id", "approver_id", "action", "context",
			"status", "confirmation_code_hash", "verification_attempts",
			"token_jti", "expires_at", "approved_at", "denied_at",
			"cancelled_at", "created_at",
		})
	})
	t.Run("request_ids", func(t *testing.T) {
		testhelper.RequireColumns(t, tx, "request_ids", []string{"request_id", "agent_id", "approver_id", "created_at"})
	})
	t.Run("consumed_tokens", func(t *testing.T) {
		testhelper.RequireColumns(t, tx, "consumed_tokens", []string{"jti", "consumed_at"})
	})
}

func TestApprovalIndexes(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	indexes := []struct {
		table string
		name  string
	}{
		{"approvals", "idx_approvals_agent_status"},
		{"approvals", "idx_approvals_agent_created"},
		{"approvals", "idx_approvals_expires_at"},
		{"approvals", "idx_approvals_approver_created"},
		{"request_ids", "idx_request_ids_created_at"},
		{"consumed_tokens", "idx_consumed_tokens_consumed_at"},
	}

	for _, idx := range indexes {
		t.Run(idx.name, func(t *testing.T) {
			testhelper.RequireIndex(t, tx, idx.table, idx.name)
		})
	}
}

func TestApprovalCascadeDeleteOnAgentDelete(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)

	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertApproval(t, tx, testhelper.GenerateID(t, "appr_"), agentID, uid)
	testhelper.InsertRequestID(t, tx, testhelper.GenerateID(t, "req_"), agentID, uid)

	testhelper.RequireCascadeDeletes(t, tx,
		fmt.Sprintf("DELETE FROM agents WHERE agent_id = %d", agentID),
		[]string{"approvals", "request_ids"},
		fmt.Sprintf("agent_id = %d", agentID),
	)
}

func TestApprovalCascadeDeleteOnProfileDelete(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)

	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertApproval(t, tx, testhelper.GenerateID(t, "appr_"), agentID, uid)

	testhelper.RequireCascadeDeletes(t, tx,
		"DELETE FROM profiles WHERE id = '"+uid+"'",
		[]string{"approvals"},
		"approver_id = '"+uid+"'",
	)
}

func TestApprovalStatusCheckConstraint(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)

	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	base := testhelper.GenerateID(t, "appr_")
	testhelper.RequireCheckValues(t, tx, "status",
		[]string{"pending", "approved", "denied", "cancelled"}, "invalid",
		func(value string, i int) error {
			_, err := tx.Exec(context.Background(),
				`INSERT INTO approvals (approval_id, agent_id, approver_id, action, context, status, expires_at)
				 VALUES ($1, $2, $3, '{"type":"test"}', '{"description":"test"}', $4, now() + interval '1 hour')`,
				fmt.Sprintf("%s_%d", base, i), agentID, uid, value)
			return err
		})
}

func TestApprovalTokenJTIUnique(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)

	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	jti := testhelper.GenerateID(t, "jti_")
	apprID1 := testhelper.GenerateID(t, "appr_")
	apprID2 := testhelper.GenerateID(t, "appr_")

	testhelper.RequireUniqueViolation(t, tx, "token_jti",
		func() error {
			testhelper.InsertApprovalWithJTI(t, tx, apprID1, agentID, uid, jti)
			return nil
		},
		func() error {
			_, err := tx.Exec(context.Background(),
				`INSERT INTO approvals (approval_id, agent_id, approver_id, action, context, status, token_jti, expires_at)
				 VALUES ($1, $2, $3, '{"type":"test"}', '{"description":"test"}', 'approved', $4, now() + interval '1 hour')`,
				apprID2, agentID, uid, jti)
			return err
		})
}

func TestRequestIdCascadeDeleteOnAgentDelete(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)

	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertRequestID(t, tx, testhelper.GenerateID(t, "req_"), agentID, uid)

	testhelper.RequireCascadeDeletes(t, tx,
		fmt.Sprintf("DELETE FROM agents WHERE agent_id = %d", agentID),
		[]string{"request_ids"},
		fmt.Sprintf("agent_id = %d", agentID),
	)
}

func TestConsumedTokensInsert(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	tok := testhelper.GenerateID(t, "tok_")
	testhelper.RequireUniqueViolation(t, tx, "consumed_tokens PK",
		func() error {
			_, err := tx.Exec(context.Background(),
				`INSERT INTO consumed_tokens (jti) VALUES ($1)`, tok)
			return err
		},
		func() error {
			_, err := tx.Exec(context.Background(),
				`INSERT INTO consumed_tokens (jti) VALUES ($1)`, tok)
			return err
		})
}

func TestListApprovalsByApproverPaginated(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	// Create 5 pending approvals with distinct created_at times (future expires_at)
	ids := make([]string, 5)
	for i := 0; i < 5; i++ {
		ids[i] = testhelper.GenerateID(t, "appr_")
		testhelper.InsertApprovalWithCreatedAt(t, tx, ids[i], agentID, uid,
			time.Date(2026, 6, 1+i, 0, 0, 0, 0, time.UTC))
	}

	ctx := context.Background()

	// Page 1: limit=2 from the start (newest first: ids[4], ids[3])
	page1, err := db.ListApprovalsByApproverPaginated(ctx, tx, uid, "pending", 2, nil)
	if err != nil {
		t.Fatalf("page 1: %v", err)
	}
	if len(page1.Approvals) != 2 {
		t.Fatalf("page 1: expected 2 approvals, got %d", len(page1.Approvals))
	}
	if !page1.HasMore {
		t.Error("page 1: expected has_more=true")
	}
	if page1.Approvals[0].ApprovalID != ids[4] {
		t.Errorf("page 1[0]: expected %s, got %s", ids[4], page1.Approvals[0].ApprovalID)
	}

	// Page 2: cursor from last item of page 1
	last := page1.Approvals[len(page1.Approvals)-1]
	cursor := &db.ApprovalCursor{CreatedAt: last.CreatedAt, ApprovalID: last.ApprovalID}
	page2, err := db.ListApprovalsByApproverPaginated(ctx, tx, uid, "pending", 2, cursor)
	if err != nil {
		t.Fatalf("page 2: %v", err)
	}
	if len(page2.Approvals) != 2 {
		t.Fatalf("page 2: expected 2 approvals, got %d", len(page2.Approvals))
	}
	if !page2.HasMore {
		t.Error("page 2: expected has_more=true")
	}

	// Page 3: should have 1 remaining
	last = page2.Approvals[len(page2.Approvals)-1]
	cursor = &db.ApprovalCursor{CreatedAt: last.CreatedAt, ApprovalID: last.ApprovalID}
	page3, err := db.ListApprovalsByApproverPaginated(ctx, tx, uid, "pending", 2, cursor)
	if err != nil {
		t.Fatalf("page 3: %v", err)
	}
	if len(page3.Approvals) != 1 {
		t.Fatalf("page 3: expected 1 approval, got %d", len(page3.Approvals))
	}
	if page3.HasMore {
		t.Error("page 3: expected has_more=false")
	}

	// Collect all IDs and verify no duplicates
	seen := map[string]bool{}
	for _, p := range []*db.ApprovalPage{page1, page2, page3} {
		for _, a := range p.Approvals {
			if seen[a.ApprovalID] {
				t.Errorf("duplicate approval_id %s across pages", a.ApprovalID)
			}
			seen[a.ApprovalID] = true
		}
	}
	if len(seen) != 5 {
		t.Errorf("expected 5 unique approvals, got %d", len(seen))
	}
}

func TestListApprovalsByApproverPaginated_DuplicateTimestamps(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	// Create 4 approvals all sharing the same created_at timestamp.
	// The compound cursor (created_at, approval_id) must use approval_id
	// as a tiebreaker to avoid skipping or duplicating rows.
	sameTime := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	for i := 0; i < 4; i++ {
		testhelper.InsertApprovalWithCreatedAt(t, tx, testhelper.GenerateID(t, "appr_"), agentID, uid, sameTime)
	}

	ctx := context.Background()

	// Page 1: limit=2
	page1, err := db.ListApprovalsByApproverPaginated(ctx, tx, uid, "pending", 2, nil)
	if err != nil {
		t.Fatalf("page 1: %v", err)
	}
	if len(page1.Approvals) != 2 {
		t.Fatalf("page 1: expected 2 approvals, got %d", len(page1.Approvals))
	}
	if !page1.HasMore {
		t.Error("page 1: expected has_more=true")
	}

	// Page 2: cursor from last item of page 1
	last := page1.Approvals[len(page1.Approvals)-1]
	cursor := &db.ApprovalCursor{CreatedAt: last.CreatedAt, ApprovalID: last.ApprovalID}
	page2, err := db.ListApprovalsByApproverPaginated(ctx, tx, uid, "pending", 2, cursor)
	if err != nil {
		t.Fatalf("page 2: %v", err)
	}
	if len(page2.Approvals) != 2 {
		t.Fatalf("page 2: expected 2 approvals, got %d", len(page2.Approvals))
	}
	if page2.HasMore {
		t.Error("page 2: expected has_more=false")
	}

	// All 4 unique approvals returned with no duplicates
	seen := map[string]bool{}
	for _, p := range []*db.ApprovalPage{page1, page2} {
		for _, a := range p.Approvals {
			if seen[a.ApprovalID] {
				t.Errorf("duplicate approval_id %s across pages", a.ApprovalID)
			}
			seen[a.ApprovalID] = true
		}
	}
	if len(seen) != 4 {
		t.Errorf("expected 4 unique approvals, got %d", len(seen))
	}
}

func TestListApprovalsByApproverPaginated_StatusFilters(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	// Create approvals with various statuses
	testhelper.InsertApproval(t, tx, testhelper.GenerateID(t, "appr_"), agentID, uid) // pending
	testhelper.InsertApprovalWithStatus(t, tx, testhelper.GenerateID(t, "appr_"), agentID, uid, "approved")
	testhelper.InsertApprovalWithStatus(t, tx, testhelper.GenerateID(t, "appr_"), agentID, uid, "denied")
	testhelper.InsertApprovalWithStatus(t, tx, testhelper.GenerateID(t, "appr_"), agentID, uid, "cancelled")

	ctx := context.Background()

	tests := []struct {
		filter   string
		expected int
	}{
		{"pending", 1},
		{"approved", 1},
		{"denied", 1},
		{"cancelled", 1},
		{"all", 4},
	}

	for _, tt := range tests {
		page, err := db.ListApprovalsByApproverPaginated(ctx, tx, uid, tt.filter, 50, nil)
		if err != nil {
			t.Fatalf("filter=%q: %v", tt.filter, err)
		}
		if len(page.Approvals) != tt.expected {
			t.Errorf("filter=%q: expected %d approvals, got %d", tt.filter, tt.expected, len(page.Approvals))
		}
	}
}

func TestApprovalVerificationAttemptsCheckConstraint(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	apprID := testhelper.GenerateID(t, "appr_")

	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertApproval(t, tx, apprID, agentID, uid)

	// Setting verification_attempts to a negative value should fail
	err := testhelper.WithSavepoint(t, tx, func() error {
		_, err := tx.Exec(context.Background(),
			`UPDATE approvals SET verification_attempts = -1 WHERE approval_id = $1`, apprID)
		return err
	})
	if err == nil {
		t.Error("expected CHECK constraint violation for negative verification_attempts, but update succeeded")
	}
}

func TestPgCronJobsScheduled(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	for _, jobName := range []string{"cleanup_request_ids", "cleanup_consumed_tokens"} {
		t.Run(jobName, func(t *testing.T) {
			testhelper.RequirePgCronJob(t, tx, jobName)
		})
	}
}
