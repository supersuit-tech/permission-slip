package db_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip-web/db"
	"github.com/supersuit-tech/permission-slip-web/db/testhelper"
)

func TestAgentsSchema(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	testhelper.RequireColumns(t, tx, "agents", []string{
		"agent_id", "public_key", "approver_id", "status",
		"metadata", "confirmation_code", "verification_attempts",
		"registration_ttl", "expires_at", "registered_at", "deactivated_at",
		"last_active_at", "created_at",
	})
}

func TestAgentIndexes(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	testhelper.RequireIndex(t, tx, "agents", "idx_agents_approver_status")
	testhelper.RequireIndex(t, tx, "agents", "idx_agents_approver_created")
}

func TestAgentCascadeDeleteOnProfileDelete(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)

	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	_ = agentID

	testhelper.RequireCascadeDeletes(t, tx,
		"DELETE FROM profiles WHERE id = '"+uid+"'",
		[]string{"agents"},
		"approver_id = '"+uid+"'",
	)
}

func TestAgentStatusCheckConstraint(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)

	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	testhelper.RequireCheckValues(t, tx, "status",
		[]string{"pending", "registered", "deactivated"}, "invalid",
		func(value string, _ int) error {
			_, err := tx.Exec(context.Background(),
				`INSERT INTO agents (public_key, approver_id, status)
				 VALUES ('pk', $1, $2)`,
				uid, value)
			return err
		})
}

func TestAgentRegistrationTTLCheckConstraint(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	ctx := context.Background()
	uid := testhelper.GenerateUID(t)

	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	// Valid TTL values should succeed
	for _, ttl := range []int{60, 3600, 86400} {
		_, err := tx.Exec(ctx,
			`INSERT INTO agents (public_key, approver_id, status, registration_ttl) VALUES ('pk', $1, 'pending', $2)`,
			uid, ttl)
		if err != nil {
			t.Errorf("valid TTL %d was rejected: %v", ttl, err)
		}
	}

	// NULL TTL should also succeed (it's nullable)
	_, err := tx.Exec(ctx,
		`INSERT INTO agents (public_key, approver_id, status, registration_ttl) VALUES ('pk', $1, 'pending', NULL)`,
		uid)
	if err != nil {
		t.Errorf("NULL TTL was rejected: %v", err)
	}

	// TTL below 60 should fail
	err = testhelper.WithSavepoint(t, tx, func() error {
		_, err := tx.Exec(ctx,
			`INSERT INTO agents (public_key, approver_id, status, registration_ttl) VALUES ('pk', $1, 'pending', 59)`,
			uid)
		return err
	})
	if err == nil {
		t.Error("expected CHECK constraint violation for TTL=59, but insert succeeded")
	}

	// TTL above 86400 should fail
	err = testhelper.WithSavepoint(t, tx, func() error {
		_, err := tx.Exec(ctx,
			`INSERT INTO agents (public_key, approver_id, status, registration_ttl) VALUES ('pk', $1, 'pending', 86401)`,
			uid)
		return err
	})
	if err == nil {
		t.Error("expected CHECK constraint violation for TTL=86401, but insert succeeded")
	}
}

func TestGetAgentsByApprover(t *testing.T) {
	t.Parallel()

	t.Run("Empty", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)
		uid := testhelper.GenerateUID(t)
		testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

		page, err := db.GetAgentsByApprover(context.Background(), tx, uid, 50, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(page.Agents) != 0 {
			t.Errorf("expected 0 agents, got %d", len(page.Agents))
		}
		if page.HasMore {
			t.Error("expected has_more=false")
		}
	})

	t.Run("DefaultLimit", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)
		uid := testhelper.GenerateUID(t)
		testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

		for i := 0; i < 3; i++ {
			testhelper.InsertAgentWithStatus(t, tx, uid, "pending")
		}

		// limit=0 should default to 50
		page, err := db.GetAgentsByApprover(context.Background(), tx, uid, 0, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(page.Agents) != 3 {
			t.Errorf("expected 3 agents, got %d", len(page.Agents))
		}
		if page.HasMore {
			t.Error("expected has_more=false")
		}
	})

	t.Run("LimitClampedTo100", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)
		uid := testhelper.GenerateUID(t)
		testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

		testhelper.InsertAgentWithStatus(t, tx, uid, "pending")

		// limit=200 should be clamped to 100
		page, err := db.GetAgentsByApprover(context.Background(), tx, uid, 200, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(page.Agents) != 1 {
			t.Errorf("expected 1 agent, got %d", len(page.Agents))
		}
	})

	t.Run("HasMore", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)
		uid := testhelper.GenerateUID(t)
		testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

		for i := 0; i < 3; i++ {
			testhelper.InsertAgentWithCreatedAt(t, tx, uid, time.Date(2026, 1, 1+i, 0, 0, 0, 0, time.UTC))
		}

		ctx := context.Background()
		page, err := db.GetAgentsByApprover(ctx, tx, uid, 2, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(page.Agents) != 2 {
			t.Fatalf("expected 2 agents, got %d", len(page.Agents))
		}
		if !page.HasMore {
			t.Error("expected has_more=true")
		}
	})

	t.Run("CursorFiltering", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)
		uid := testhelper.GenerateUID(t)
		testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

		// Insert 5 agents with distinct timestamps
		for i := 0; i < 5; i++ {
			testhelper.InsertAgentWithCreatedAt(t, tx, uid, time.Date(2026, 1, 1+i, 0, 0, 0, 0, time.UTC))
		}

		// First page
		ctx := context.Background()
		page1, err := db.GetAgentsByApprover(ctx, tx, uid, 2, nil)
		if err != nil {
			t.Fatalf("page 1: %v", err)
		}
		if len(page1.Agents) != 2 {
			t.Fatalf("page 1: expected 2, got %d", len(page1.Agents))
		}
		if !page1.HasMore {
			t.Error("page 1: expected has_more=true")
		}

		// Second page using cursor from first
		last1 := page1.Agents[len(page1.Agents)-1]
		cursor := &db.AgentCursor{CreatedAt: last1.CreatedAt, AgentID: last1.AgentID}
		page2, err := db.GetAgentsByApprover(ctx, tx, uid, 2, cursor)
		if err != nil {
			t.Fatalf("page 2: %v", err)
		}
		if len(page2.Agents) != 2 {
			t.Fatalf("page 2: expected 2, got %d", len(page2.Agents))
		}
		if !page2.HasMore {
			t.Error("page 2: expected has_more=true")
		}

		// Third page — 1 remaining
		last2 := page2.Agents[len(page2.Agents)-1]
		cursor = &db.AgentCursor{CreatedAt: last2.CreatedAt, AgentID: last2.AgentID}
		page3, err := db.GetAgentsByApprover(ctx, tx, uid, 2, cursor)
		if err != nil {
			t.Fatalf("page 3: %v", err)
		}
		if len(page3.Agents) != 1 {
			t.Fatalf("page 3: expected 1, got %d", len(page3.Agents))
		}
		if page3.HasMore {
			t.Error("page 3: expected has_more=false")
		}

		// Verify no duplicates across all pages
		seen := map[int64]bool{}
		for _, a := range page1.Agents {
			seen[a.AgentID] = true
		}
		for _, a := range page2.Agents {
			if seen[a.AgentID] {
				t.Errorf("duplicate agent %d", a.AgentID)
			}
			seen[a.AgentID] = true
		}
		for _, a := range page3.Agents {
			if seen[a.AgentID] {
				t.Errorf("duplicate agent %d", a.AgentID)
			}
			seen[a.AgentID] = true
		}
		if len(seen) != 5 {
			t.Errorf("expected 5 unique agents, got %d", len(seen))
		}
	})

	t.Run("DuplicateTimestamps", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)
		uid := testhelper.GenerateUID(t)
		testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

		// Insert 4 agents all sharing the same created_at timestamp.
		sameTime := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
		for i := 0; i < 4; i++ {
			testhelper.InsertAgentWithCreatedAt(t, tx, uid, sameTime)
		}

		// Page through all 4 with limit=2
		ctx := context.Background()
		page1, err := db.GetAgentsByApprover(ctx, tx, uid, 2, nil)
		if err != nil {
			t.Fatalf("page 1: %v", err)
		}
		if len(page1.Agents) != 2 {
			t.Fatalf("page 1: expected 2, got %d", len(page1.Agents))
		}
		if !page1.HasMore {
			t.Error("page 1: expected has_more=true")
		}

		last1 := page1.Agents[len(page1.Agents)-1]
		cursor := &db.AgentCursor{CreatedAt: last1.CreatedAt, AgentID: last1.AgentID}
		page2, err := db.GetAgentsByApprover(ctx, tx, uid, 2, cursor)
		if err != nil {
			t.Fatalf("page 2: %v", err)
		}
		if len(page2.Agents) != 2 {
			t.Fatalf("page 2: expected 2, got %d", len(page2.Agents))
		}
		if page2.HasMore {
			t.Error("page 2: expected has_more=false")
		}

		// Ensure all 4 unique agents returned with no duplicates
		seen := map[int64]bool{}
		for _, a := range page1.Agents {
			seen[a.AgentID] = true
		}
		for _, a := range page2.Agents {
			if seen[a.AgentID] {
				t.Errorf("duplicate agent %d across pages with same timestamp", a.AgentID)
			}
			seen[a.AgentID] = true
		}
		if len(seen) != 4 {
			t.Errorf("expected 4 unique agents, got %d", len(seen))
		}
	})

	t.Run("ExcludesExpiredPending", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)
		uid := testhelper.GenerateUID(t)
		testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
		ctx := context.Background()

		// Create a registered agent (should always appear).
		testhelper.InsertAgentWithStatus(t, tx, uid, "registered")

		// Create a pending agent with a real TTL via InsertPendingAgent.
		pending, err := db.InsertPendingAgent(ctx, tx, uid, "pk_active", "AA1-BB2", 300, nil)
		if err != nil {
			t.Fatalf("InsertPendingAgent: %v", err)
		}

		// Create another pending agent and backdate its expires_at to simulate expiration.
		expired, err := db.InsertPendingAgent(ctx, tx, uid, "pk_expired", "CC3-DD4", 300, nil)
		if err != nil {
			t.Fatalf("InsertPendingAgent expired: %v", err)
		}
		testhelper.MustExec(t, tx,
			`UPDATE agents SET expires_at = now() - interval '1 hour' WHERE agent_id = $1`, expired.AgentID)

		// Create a pending agent without expires_at (NULL) — should still appear.
		pendingNoTTL := testhelper.InsertAgentWithStatus(t, tx, uid, "pending")

		page, err := db.GetAgentsByApprover(ctx, tx, uid, 50, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Should see 3 agents: registered, active pending, pending-no-TTL.
		// The expired pending agent should be excluded.
		if len(page.Agents) != 3 {
			t.Fatalf("expected 3 agents (expired pending excluded), got %d", len(page.Agents))
		}

		ids := map[int64]bool{}
		for _, a := range page.Agents {
			ids[a.AgentID] = true
		}
		if ids[expired.AgentID] {
			t.Error("expired pending agent should not appear in results")
		}
		if !ids[pending.AgentID] {
			t.Error("active pending agent should appear in results")
		}
		if !ids[pendingNoTTL] {
			t.Error("pending agent without TTL should appear in results")
		}
	})

	t.Run("OrderDescending", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)
		uid := testhelper.GenerateUID(t)
		testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

		for i := 0; i < 3; i++ {
			testhelper.InsertAgentWithCreatedAt(t, tx, uid, time.Date(2026, 1, 1+i, 0, 0, 0, 0, time.UTC))
		}

		page, err := db.GetAgentsByApprover(context.Background(), tx, uid, 10, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		for i := 1; i < len(page.Agents); i++ {
			if !page.Agents[i].CreatedAt.Before(page.Agents[i-1].CreatedAt) {
				t.Errorf("agents not in descending order: %v >= %v", page.Agents[i].CreatedAt, page.Agents[i-1].CreatedAt)
			}
		}
	})
}

func TestGetAgentByID(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("Found", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)
		uid := testhelper.GenerateUID(t)
		agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

		agent, err := db.GetAgentByID(ctx, tx, agentID, uid)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if agent == nil {
			t.Fatal("expected agent, got nil")
		}
		if agent.AgentID != agentID {
			t.Errorf("expected agent_id %d, got %d", agentID, agent.AgentID)
		}
		if agent.ApproverID != uid {
			t.Errorf("expected approver_id %q, got %q", uid, agent.ApproverID)
		}
	})

	t.Run("NotFound", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)
		uid := testhelper.GenerateUID(t)
		testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

		agent, err := db.GetAgentByID(ctx, tx, 999999, uid)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if agent != nil {
			t.Errorf("expected nil, got %+v", agent)
		}
	})

	t.Run("WrongOwner", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)
		uid1 := testhelper.GenerateUID(t)
		agentID := testhelper.InsertUserWithAgent(t, tx, uid1, "u1_"+uid1[:6])

		uid2 := testhelper.GenerateUID(t)
		testhelper.InsertUser(t, tx, uid2, "u2_"+uid2[:6])

		agent, err := db.GetAgentByID(ctx, tx, agentID, uid2)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if agent != nil {
			t.Errorf("expected nil for wrong owner, got %+v", agent)
		}
	})
}

func TestUpdateAgentMetadata(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("ShallowMerge", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)
		uid := testhelper.GenerateUID(t)
		agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

		// Set initial metadata
		agent, err := db.UpdateAgentMetadata(ctx, tx, agentID, uid, []byte(`{"name":"Agent A","version":"1.0"}`))
		if err != nil {
			t.Fatalf("initial update: %v", err)
		}
		if agent == nil {
			t.Fatal("initial update returned nil")
		}

		// Merge: override name, preserve version
		agent, err = db.UpdateAgentMetadata(ctx, tx, agentID, uid, []byte(`{"name":"Agent B"}`))
		if err != nil {
			t.Fatalf("merge update: %v", err)
		}
		if agent == nil {
			t.Fatal("merge update returned nil")
		}

		var meta map[string]any
		if err := json.Unmarshal(agent.Metadata, &meta); err != nil {
			t.Fatalf("unmarshal metadata: %v", err)
		}
		if meta["name"] != "Agent B" {
			t.Errorf("expected name 'Agent B', got %v", meta["name"])
		}
		if meta["version"] != "1.0" {
			t.Errorf("expected version '1.0' preserved, got %v", meta["version"])
		}
	})

	t.Run("NotFound", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)
		uid := testhelper.GenerateUID(t)
		testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

		agent, err := db.UpdateAgentMetadata(ctx, tx, 999999, uid, []byte(`{"name":"x"}`))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if agent != nil {
			t.Errorf("expected nil for non-existent agent, got %+v", agent)
		}
	})

	t.Run("WrongOwner", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)
		uid1 := testhelper.GenerateUID(t)
		agentID := testhelper.InsertUserWithAgent(t, tx, uid1, "u1_"+uid1[:6])

		uid2 := testhelper.GenerateUID(t)
		testhelper.InsertUser(t, tx, uid2, "u2_"+uid2[:6])

		agent, err := db.UpdateAgentMetadata(ctx, tx, agentID, uid2, []byte(`{"name":"hijack"}`))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if agent != nil {
			t.Errorf("expected nil for wrong owner, got %+v", agent)
		}
	})

	t.Run("NullExistingMetadata", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)
		uid := testhelper.GenerateUID(t)
		agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

		// Default metadata is NULL — merge should initialize to {} then merge
		agent, err := db.UpdateAgentMetadata(ctx, tx, agentID, uid, []byte(`{"name":"First"}`))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if agent == nil {
			t.Fatal("expected agent, got nil")
		}

		var meta map[string]any
		if err := json.Unmarshal(agent.Metadata, &meta); err != nil {
			t.Fatalf("unmarshal metadata: %v", err)
		}
		if meta["name"] != "First" {
			t.Errorf("expected name 'First', got %v", meta["name"])
		}
	})

	t.Run("NestedObjectReplaced", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)
		uid := testhelper.GenerateUID(t)
		agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

		// Set initial metadata with nested object
		_, err := db.UpdateAgentMetadata(ctx, tx, agentID, uid, []byte(`{"config":{"a":1,"b":2}}`))
		if err != nil {
			t.Fatalf("initial: %v", err)
		}

		// Merge with a different nested object — should replace entirely, not deep-merge
		agent, err := db.UpdateAgentMetadata(ctx, tx, agentID, uid, []byte(`{"config":{"c":3}}`))
		if err != nil {
			t.Fatalf("merge: %v", err)
		}
		if agent == nil {
			t.Fatal("expected agent, got nil")
		}

		var meta map[string]any
		if err := json.Unmarshal(agent.Metadata, &meta); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		config, ok := meta["config"].(map[string]any)
		if !ok {
			t.Fatalf("expected config to be object, got %T", meta["config"])
		}
		if _, exists := config["a"]; exists {
			t.Error("expected nested key 'a' to be gone after replacement")
		}
		if _, exists := config["b"]; exists {
			t.Error("expected nested key 'b' to be gone after replacement")
		}
		if config["c"] != float64(3) {
			t.Errorf("expected config.c=3, got %v", config["c"])
		}
	})
}
