package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip-web/db/testhelper"
)

func decodeAgentList(t *testing.T, body []byte) agentListResponse {
	t.Helper()
	var resp agentListResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("failed to unmarshal agent list response: %v", err)
	}
	return resp
}

func TestListAgents_Empty(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, "/agents", uid)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := decodeAgentList(t, w.Body.Bytes())
	if len(resp.Data) != 0 {
		t.Errorf("expected 0 agents, got %d", len(resp.Data))
	}
	if resp.HasMore {
		t.Error("expected has_more=false for empty list")
	}
}

func TestListAgents_ReturnsOwnAgents(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	// Create a second agent for the same user
	testhelper.InsertAgentWithStatus(t, tx, uid, "registered")

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, "/agents", uid)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := decodeAgentList(t, w.Body.Bytes())
	if len(resp.Data) != 2 {
		t.Fatalf("expected 2 agents, got %d", len(resp.Data))
	}
	if resp.HasMore {
		t.Error("expected has_more=false when all agents fit on one page")
	}
}

func TestListAgents_DoesNotReturnOtherUsersAgents(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	uid1 := testhelper.GenerateUID(t)
	agent1 := testhelper.InsertUserWithAgent(t, tx, uid1, "u1_"+uid1[:6])

	uid2 := testhelper.GenerateUID(t)
	testhelper.InsertUserWithAgent(t, tx, uid2, "u2_"+uid2[:6])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, "/agents", uid1)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := decodeAgentList(t, w.Body.Bytes())
	if len(resp.Data) != 1 {
		t.Fatalf("expected 1 agent, got %d", len(resp.Data))
	}
	if resp.Data[0].AgentID != agent1 {
		t.Errorf("expected agent %d, got %d", agent1, resp.Data[0].AgentID)
	}
}

func TestListAgents_Unauthenticated(t *testing.T) {
	t.Parallel()
	deps := &Deps{SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := httptest.NewRequest(http.MethodGet, "/agents", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestListAgents_Pagination(t *testing.T) {
	t.Parallel()

	t.Run("LimitParam", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)
		uid := testhelper.GenerateUID(t)
		testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

		// Create 3 agents with distinct created_at times
		for i := 0; i < 3; i++ {
			testhelper.InsertAgentWithCreatedAt(t, tx, uid, time.Date(2026, 1, 1+i, 0, 0, 0, 0, time.UTC))
		}

		deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
		router := NewRouter(deps)

		r := authenticatedRequest(t, http.MethodGet, "/agents?limit=2", uid)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		resp := decodeAgentList(t, w.Body.Bytes())
		if len(resp.Data) != 2 {
			t.Fatalf("expected 2 agents, got %d", len(resp.Data))
		}
		if !resp.HasMore {
			t.Error("expected has_more=true when more agents exist")
		}
		if resp.NextCursor == nil {
			t.Fatal("expected next_cursor to be set when has_more=true")
		}
	})

	t.Run("CursorPagination", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)
		uid := testhelper.GenerateUID(t)
		testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

		// Create 5 agents with distinct created_at times
		agentIDs := make([]int64, 5)
		for i := 0; i < 5; i++ {
			agentIDs[i] = testhelper.InsertAgentWithCreatedAt(t, tx, uid, time.Date(2026, 1, 1+i, 0, 0, 0, 0, time.UTC))
		}

		deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
		router := NewRouter(deps)

		// First page: limit=2
		r := authenticatedRequest(t, http.MethodGet, "/agents?limit=2", uid)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)

		if w.Code != http.StatusOK {
			t.Fatalf("page 1: expected 200, got %d: %s", w.Code, w.Body.String())
		}

		page1 := decodeAgentList(t, w.Body.Bytes())
		if len(page1.Data) != 2 {
			t.Fatalf("page 1: expected 2 agents, got %d", len(page1.Data))
		}
		if !page1.HasMore {
			t.Error("page 1: expected has_more=true")
		}
		if page1.NextCursor == nil {
			t.Fatal("page 1: expected next_cursor")
		}

		// Second page: use cursor from first page
		r = authenticatedRequest(t, http.MethodGet, fmt.Sprintf("/agents?limit=2&after=%s", *page1.NextCursor), uid)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, r)

		if w.Code != http.StatusOK {
			t.Fatalf("page 2: expected 200, got %d: %s", w.Code, w.Body.String())
		}

		page2 := decodeAgentList(t, w.Body.Bytes())
		if len(page2.Data) != 2 {
			t.Fatalf("page 2: expected 2 agents, got %d", len(page2.Data))
		}
		if !page2.HasMore {
			t.Error("page 2: expected has_more=true")
		}

		// Third page: should have 1 remaining
		r = authenticatedRequest(t, http.MethodGet, fmt.Sprintf("/agents?limit=2&after=%s", *page2.NextCursor), uid)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, r)

		if w.Code != http.StatusOK {
			t.Fatalf("page 3: expected 200, got %d: %s", w.Code, w.Body.String())
		}

		page3 := decodeAgentList(t, w.Body.Bytes())
		if len(page3.Data) != 1 {
			t.Fatalf("page 3: expected 1 agent, got %d", len(page3.Data))
		}
		if page3.HasMore {
			t.Error("page 3: expected has_more=false")
		}
		if page3.NextCursor != nil {
			t.Error("page 3: expected next_cursor to be nil")
		}

		// Ensure no duplicates across pages
		seen := map[int64]bool{}
		for _, a := range page1.Data {
			seen[a.AgentID] = true
		}
		for _, a := range page2.Data {
			if seen[a.AgentID] {
				t.Errorf("duplicate agent_id %d across pages", a.AgentID)
			}
			seen[a.AgentID] = true
		}
		for _, a := range page3.Data {
			if seen[a.AgentID] {
				t.Errorf("duplicate agent_id %d across pages", a.AgentID)
			}
			seen[a.AgentID] = true
		}
		if len(seen) != 5 {
			t.Errorf("expected 5 unique agents across all pages, got %d", len(seen))
		}
	})

	t.Run("InvalidLimit", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)
		uid := testhelper.GenerateUID(t)
		testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

		deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
		router := NewRouter(deps)

		for _, limit := range []string{"0", "-1", "abc", "101"} {
			t.Run("limit="+limit, func(t *testing.T) {
				r := authenticatedRequest(t, http.MethodGet, "/agents?limit="+limit, uid)
				w := httptest.NewRecorder()
				router.ServeHTTP(w, r)

				if w.Code != http.StatusBadRequest {
					t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
				}
			})
		}
	})

	t.Run("InvalidCursor", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)
		uid := testhelper.GenerateUID(t)
		testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

		deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
		router := NewRouter(deps)

		for _, cursor := range []string{
			"not-a-cursor",
			"bad-timestamp,123",
			"2026-01-01T00:00:00Z,",
			"2026-01-01T00:00:00Z,abc",
		} {
			t.Run("cursor="+cursor, func(t *testing.T) {
				r := authenticatedRequest(t, http.MethodGet, "/agents?after="+cursor, uid)
				w := httptest.NewRecorder()
				router.ServeHTTP(w, r)

				if w.Code != http.StatusBadRequest {
					t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
				}
			})
		}
	})

	t.Run("DefaultLimit", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)
		uid := testhelper.GenerateUID(t)
		testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

		// Create 3 agents — should all be returned with default limit=50
		for i := 0; i < 3; i++ {
			testhelper.InsertAgentWithStatus(t, tx, uid, "pending")
		}

		deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
		router := NewRouter(deps)

		r := authenticatedRequest(t, http.MethodGet, "/agents", uid)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		resp := decodeAgentList(t, w.Body.Bytes())
		if len(resp.Data) != 3 {
			t.Fatalf("expected 3 agents, got %d", len(resp.Data))
		}
		if resp.HasMore {
			t.Error("expected has_more=false")
		}
	})
}

func TestListAgents_LastActiveAt(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	lastActive := time.Date(2026, 2, 19, 8, 45, 0, 0, time.UTC)
	testhelper.SetAgentLastActiveAt(t, tx, agentID, uid, lastActive)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, "/agents", uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := decodeAgentList(t, w.Body.Bytes())
	if len(resp.Data) != 1 {
		t.Fatalf("expected 1 agent, got %d", len(resp.Data))
	}
	if resp.Data[0].LastActiveAt == nil {
		t.Fatal("expected last_active_at to be set, got nil")
	}
	if !resp.Data[0].LastActiveAt.Equal(lastActive) {
		t.Errorf("expected last_active_at=%v, got %v", lastActive, *resp.Data[0].LastActiveAt)
	}
}

// ── GET /agents/{agent_id} ──────────────────────────────────────────────────

func decodeAgentResponse(t *testing.T, body []byte) agentResponse {
	t.Helper()
	var resp agentResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("failed to unmarshal agent response: %v", err)
	}
	return resp
}

func TestGetAgent_Found(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, fmt.Sprintf("/agents/%d", agentID), uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	resp := decodeAgentResponse(t, w.Body.Bytes())
	if resp.AgentID != agentID {
		t.Errorf("expected agent_id %d, got %d", agentID, resp.AgentID)
	}
}

func TestGetAgent_NotFound(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, "/agents/999999", uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetAgent_WrongOwner(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid1 := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid1, "u1_"+uid1[:6])

	uid2 := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid2, "u2_"+uid2[:6])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, fmt.Sprintf("/agents/%d", agentID), uid2)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for wrong owner, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetAgent_InvalidID(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	for _, id := range []string{"abc", "0", "-1"} {
		t.Run("id="+id, func(t *testing.T) {
			r := authenticatedRequest(t, http.MethodGet, "/agents/"+id, uid)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, r)

			if w.Code != http.StatusBadRequest {
				t.Errorf("expected 400 for id=%s, got %d: %s", id, w.Code, w.Body.String())
			}
		})
	}
}

// ── PATCH /agents/{agent_id} ────────────────────────────────────────────────

func TestUpdateAgent_ValidMetadata(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedJSONRequest(t, http.MethodPatch, fmt.Sprintf("/agents/%d", agentID), uid,
		`{"metadata":{"name":"Updated Agent"}}`)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	resp := decodeAgentResponse(t, w.Body.Bytes())
	if resp.AgentID != agentID {
		t.Errorf("expected agent_id %d, got %d", agentID, resp.AgentID)
	}
	meta, ok := resp.Metadata.(map[string]any)
	if !ok {
		t.Fatalf("expected metadata to be object, got %T", resp.Metadata)
	}
	if meta["name"] != "Updated Agent" {
		t.Errorf("expected name 'Updated Agent', got %v", meta["name"])
	}
}

func TestUpdateAgent_ShallowMerge(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	// Set initial metadata
	r := authenticatedJSONRequest(t, http.MethodPatch, fmt.Sprintf("/agents/%d", agentID), uid,
		`{"metadata":{"name":"A","version":"1.0"}}`)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("initial update: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Update name only — version should be preserved
	r = authenticatedJSONRequest(t, http.MethodPatch, fmt.Sprintf("/agents/%d", agentID), uid,
		`{"metadata":{"name":"B"}}`)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("merge update: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := decodeAgentResponse(t, w.Body.Bytes())
	meta, ok := resp.Metadata.(map[string]any)
	if !ok {
		t.Fatalf("expected metadata to be object, got %T", resp.Metadata)
	}
	if meta["name"] != "B" {
		t.Errorf("expected name 'B', got %v", meta["name"])
	}
	if meta["version"] != "1.0" {
		t.Errorf("expected version '1.0' preserved, got %v", meta["version"])
	}
}

func TestUpdateAgent_NotFound(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedJSONRequest(t, http.MethodPatch, "/agents/999999", uid,
		`{"metadata":{"name":"x"}}`)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdateAgent_WrongOwner(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid1 := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid1, "u1_"+uid1[:6])

	uid2 := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid2, "u2_"+uid2[:6])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedJSONRequest(t, http.MethodPatch, fmt.Sprintf("/agents/%d", agentID), uid2,
		`{"metadata":{"name":"hijack"}}`)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for wrong owner, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdateAgent_InvalidID(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	for _, id := range []string{"abc", "0", "-1"} {
		t.Run("id="+id, func(t *testing.T) {
			r := authenticatedJSONRequest(t, http.MethodPatch, "/agents/"+id, uid,
				`{"metadata":{"name":"x"}}`)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, r)

			if w.Code != http.StatusBadRequest {
				t.Errorf("expected 400 for id=%s, got %d: %s", id, w.Code, w.Body.String())
			}
		})
	}
}

func TestUpdateAgent_MissingMetadata(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedJSONRequest(t, http.MethodPatch, fmt.Sprintf("/agents/%d", agentID), uid, `{}`)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing metadata, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdateAgent_NonObjectMetadata(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	for _, tc := range []struct {
		name string
		body string
	}{
		{"array", `{"metadata":[1,2,3]}`},
		{"string", `{"metadata":"hello"}`},
		{"number", `{"metadata":42}`},
		{"boolean", `{"metadata":true}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := authenticatedJSONRequest(t, http.MethodPatch, fmt.Sprintf("/agents/%d", agentID), uid, tc.body)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, r)

			if w.Code != http.StatusBadRequest {
				t.Errorf("expected 400 for %s metadata, got %d: %s", tc.name, w.Code, w.Body.String())
			}
		})
	}
}

func TestListAgents_RequestCount30d(t *testing.T) {
	t.Parallel()

	t.Run("CountsRecentApprovals", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)
		uid := testhelper.GenerateUID(t)
		agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

		// Create 3 approvals within the last 30 days
		for i := 0; i < 3; i++ {
			aid := testhelper.GenerateID(t, "appr_")
			testhelper.InsertApproval(t, tx, aid, agentID, uid)
		}

		deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
		router := NewRouter(deps)

		r := authenticatedRequest(t, http.MethodGet, "/agents", uid)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		resp := decodeAgentList(t, w.Body.Bytes())
		if len(resp.Data) != 1 {
			t.Fatalf("expected 1 agent, got %d", len(resp.Data))
		}
		if resp.Data[0].RequestCount30d == nil {
			t.Fatal("expected request_count_30d to be set, got nil")
		}
		if *resp.Data[0].RequestCount30d != 3 {
			t.Errorf("expected request_count_30d=3, got %d", *resp.Data[0].RequestCount30d)
		}
	})

	t.Run("ExcludesOldApprovals", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)
		uid := testhelper.GenerateUID(t)
		agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

		// Create 1 recent approval (now)
		testhelper.InsertApproval(t, tx, testhelper.GenerateID(t, "appr_"), agentID, uid)

		// Create 2 old approvals (60 days ago)
		old := time.Now().Add(-60 * 24 * time.Hour)
		for i := 0; i < 2; i++ {
			testhelper.InsertApprovalWithCreatedAt(t, tx, testhelper.GenerateID(t, "appr_"), agentID, uid, old.Add(time.Duration(i)*time.Second))
		}

		deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
		router := NewRouter(deps)

		r := authenticatedRequest(t, http.MethodGet, "/agents", uid)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		resp := decodeAgentList(t, w.Body.Bytes())
		if len(resp.Data) != 1 {
			t.Fatalf("expected 1 agent, got %d", len(resp.Data))
		}
		if resp.Data[0].RequestCount30d == nil {
			t.Fatal("expected request_count_30d to be set, got nil")
		}
		if *resp.Data[0].RequestCount30d != 1 {
			t.Errorf("expected request_count_30d=1 (only recent), got %d", *resp.Data[0].RequestCount30d)
		}
	})

	t.Run("ZeroWhenNoApprovals", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)
		uid := testhelper.GenerateUID(t)
		testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

		deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
		router := NewRouter(deps)

		r := authenticatedRequest(t, http.MethodGet, "/agents", uid)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		resp := decodeAgentList(t, w.Body.Bytes())
		if len(resp.Data) != 1 {
			t.Fatalf("expected 1 agent, got %d", len(resp.Data))
		}
		if resp.Data[0].RequestCount30d == nil {
			t.Fatal("expected request_count_30d to be set, got nil")
		}
		if *resp.Data[0].RequestCount30d != 0 {
			t.Errorf("expected request_count_30d=0, got %d", *resp.Data[0].RequestCount30d)
		}
	})
}

