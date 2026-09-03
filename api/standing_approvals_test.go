package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip/db"
	"github.com/supersuit-tech/permission-slip/db/testhelper"
)

// standingApprovalTestSetupConnector inserts connector + action rows for standing approval API tests.
func standingApprovalTestSetupConnector(t *testing.T, tx db.DBTX, actionType string) {
	t.Helper()
	safe := strings.ReplaceAll(strings.ReplaceAll(actionType, ".", "_"), "*", "wildcard")
	connectorID := "tconn_" + safe
	if len(connectorID) > 200 {
		connectorID = connectorID[:200]
	}
	testhelper.InsertConnector(t, tx, connectorID)
	testhelper.InsertConnectorAction(t, tx, connectorID, actionType, actionType)
}

func decodeStandingApprovalList(t *testing.T, body []byte) standingApprovalListResponse {
	t.Helper()
	var resp standingApprovalListResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("failed to unmarshal standing approval list response: %v", err)
	}
	return resp
}

// ── GET /standing-approvals ───────────────────────────────────────────────────

func TestListStandingApprovals_Empty(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, "/standing-approvals", uid)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := decodeStandingApprovalList(t, w.Body.Bytes())
	if len(resp.Data) != 0 {
		t.Errorf("expected 0 standing approvals, got %d", len(resp.Data))
	}
}

func TestListStandingApprovals_ReturnsActiveByDefault(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	testhelper.InsertStandingApproval(t, tx, testhelper.GenerateID(t, "sa_"), agentID, uid)
	testhelper.InsertStandingApproval(t, tx, testhelper.GenerateID(t, "sa_"), agentID, uid)
	testhelper.InsertStandingApprovalWithStatus(t, tx, testhelper.GenerateID(t, "sa_"), agentID, uid, "revoked")

	deps := &Deps{DB: tx, JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, "/standing-approvals", uid)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := decodeStandingApprovalList(t, w.Body.Bytes())
	if len(resp.Data) != 2 {
		t.Fatalf("expected 2 active standing approvals, got %d", len(resp.Data))
	}
	for _, sa := range resp.Data {
		if sa.Status != "active" {
			t.Errorf("expected status 'active', got %q", sa.Status)
		}
	}
}

func TestListStandingApprovals_StatusFilterAll(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	testhelper.InsertStandingApproval(t, tx, testhelper.GenerateID(t, "sa_"), agentID, uid)
	testhelper.InsertStandingApprovalWithStatus(t, tx, testhelper.GenerateID(t, "sa_"), agentID, uid, "revoked")
	testhelper.InsertStandingApprovalWithStatus(t, tx, testhelper.GenerateID(t, "sa_"), agentID, uid, "expired")

	deps := &Deps{DB: tx, JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, "/standing-approvals?status=all", uid)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := decodeStandingApprovalList(t, w.Body.Bytes())
	if len(resp.Data) != 3 {
		t.Errorf("expected 3 standing approvals with status=all, got %d", len(resp.Data))
	}
}

func TestListStandingApprovals_InvalidStatusFilter(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, "/standing-approvals?status=invalid", uid)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestListStandingApprovals_DoesNotReturnOtherUsersApprovals(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	uid1 := testhelper.GenerateUID(t)
	agent1 := testhelper.InsertUserWithAgent(t, tx, uid1, "u1_"+uid1[:6])
	testhelper.InsertStandingApproval(t, tx, testhelper.GenerateID(t, "sa_"), agent1, uid1)

	uid2 := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid2, "u2_"+uid2[:6])

	deps := &Deps{DB: tx, JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, "/standing-approvals", uid2)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := decodeStandingApprovalList(t, w.Body.Bytes())
	if len(resp.Data) != 0 {
		t.Errorf("expected 0 standing approvals for other user, got %d", len(resp.Data))
	}
}

func TestListStandingApprovals_Unauthenticated(t *testing.T) {
	t.Parallel()
	deps := &Deps{JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	r := httptest.NewRequest(http.MethodGet, "/standing-approvals", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestListStandingApprovals_ResponseShape(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	saID := testhelper.GenerateID(t, "sa_")
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertStandingApproval(t, tx, saID, agentID, uid)

	deps := &Deps{DB: tx, JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, "/standing-approvals", uid)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := decodeStandingApprovalList(t, w.Body.Bytes())
	if len(resp.Data) != 1 {
		t.Fatalf("expected 1 standing approval, got %d", len(resp.Data))
	}

	sa := resp.Data[0]
	if sa.StandingApprovalID != saID {
		t.Errorf("expected standing_approval_id %q, got %q", saID, sa.StandingApprovalID)
	}
	if sa.AgentID != agentID {
		t.Errorf("expected agent_id %d, got %d", agentID, sa.AgentID)
	}
	if sa.Status != "active" {
		t.Errorf("expected status 'active', got %q", sa.Status)
	}
	if sa.ActionType != "test.action" {
		t.Errorf("expected action_type 'test.action', got %q", sa.ActionType)
	}
	if sa.StartsAt.IsZero() {
		t.Error("expected starts_at to be set")
	}
	if sa.ExpiresAt.IsZero() {
		t.Error("expected expires_at to be set")
	}
	if sa.CreatedAt.IsZero() {
		t.Error("expected created_at to be set")
	}
}

func TestListStandingApprovals_NullConstraintsSerializedAsEmptyObject(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	saID := testhelper.GenerateID(t, "sa_")
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertStandingApprovalFull(t, tx, saID, agentID, uid, testhelper.StandingApprovalOpts{
		ActionType:  "test.action",
		Constraints: nil,
	})

	deps := &Deps{DB: tx, JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, "/standing-approvals?status=all", uid)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(w.Body.Bytes(), &raw); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	data, ok := raw["data"]
	if !ok {
		t.Fatal("expected data key in response")
	}
	var items []map[string]json.RawMessage
	if err := json.Unmarshal(data, &items); err != nil {
		t.Fatalf("failed to unmarshal data: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 standing approval, got %d", len(items))
	}
	constraints, ok := items[0]["constraints"]
	if !ok {
		t.Fatal("expected constraints key in standing approval")
	}
	if string(constraints) == "null" {
		t.Fatal("expected null DB constraints to serialize as {}, got null")
	}
	if string(constraints) != "{}" {
		t.Fatalf("expected constraints {}, got %s", string(constraints))
	}
}

// ── Pagination ──────────────────────────────────────────────────────────────

func TestListStandingApprovals_Pagination(t *testing.T) {
	t.Parallel()

	t.Run("LimitParam", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)
		uid := testhelper.GenerateUID(t)
		agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

		// Create 3 standing approvals with distinct created_at times
		for i := 0; i < 3; i++ {
			saID := testhelper.GenerateID(t, "sa_")
			testhelper.InsertStandingApprovalWithCreatedAt(t, tx, saID, agentID, uid, time.Date(2026, 1, 1+i, 0, 0, 0, 0, time.UTC))
		}

		deps := &Deps{DB: tx, JWTSigningSecret: testJWTSecret}
		router := NewRouter(deps)

		r := authenticatedRequest(t, http.MethodGet, "/standing-approvals?limit=2", uid)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		resp := decodeStandingApprovalList(t, w.Body.Bytes())
		if len(resp.Data) != 2 {
			t.Fatalf("expected 2 standing approvals, got %d", len(resp.Data))
		}
		if !resp.HasMore {
			t.Error("expected has_more=true when more approvals exist")
		}
		if resp.NextCursor == nil {
			t.Fatal("expected next_cursor to be set when has_more=true")
		}
	})

	t.Run("CursorPagination", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)
		uid := testhelper.GenerateUID(t)
		agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

		// Create 5 standing approvals with distinct created_at times
		saIDs := make([]string, 5)
		for i := 0; i < 5; i++ {
			saIDs[i] = testhelper.GenerateID(t, "sa_")
			testhelper.InsertStandingApprovalWithCreatedAt(t, tx, saIDs[i], agentID, uid, time.Date(2026, 1, 1+i, 0, 0, 0, 0, time.UTC))
		}

		deps := &Deps{DB: tx, JWTSigningSecret: testJWTSecret}
		router := NewRouter(deps)

		// First page: limit=2
		r := authenticatedRequest(t, http.MethodGet, "/standing-approvals?limit=2", uid)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)

		if w.Code != http.StatusOK {
			t.Fatalf("page 1: expected 200, got %d: %s", w.Code, w.Body.String())
		}

		page1 := decodeStandingApprovalList(t, w.Body.Bytes())
		if len(page1.Data) != 2 {
			t.Fatalf("page 1: expected 2 approvals, got %d", len(page1.Data))
		}
		if !page1.HasMore {
			t.Error("page 1: expected has_more=true")
		}
		if page1.NextCursor == nil {
			t.Fatal("page 1: expected next_cursor")
		}

		// Second page: use cursor from first page
		r = authenticatedRequest(t, http.MethodGet, fmt.Sprintf("/standing-approvals?limit=2&after=%s", *page1.NextCursor), uid)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, r)

		if w.Code != http.StatusOK {
			t.Fatalf("page 2: expected 200, got %d: %s", w.Code, w.Body.String())
		}

		page2 := decodeStandingApprovalList(t, w.Body.Bytes())
		if len(page2.Data) != 2 {
			t.Fatalf("page 2: expected 2 approvals, got %d", len(page2.Data))
		}
		if !page2.HasMore {
			t.Error("page 2: expected has_more=true")
		}

		// Third page: should have 1 remaining
		r = authenticatedRequest(t, http.MethodGet, fmt.Sprintf("/standing-approvals?limit=2&after=%s", *page2.NextCursor), uid)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, r)

		if w.Code != http.StatusOK {
			t.Fatalf("page 3: expected 200, got %d: %s", w.Code, w.Body.String())
		}

		page3 := decodeStandingApprovalList(t, w.Body.Bytes())
		if len(page3.Data) != 1 {
			t.Fatalf("page 3: expected 1 approval, got %d", len(page3.Data))
		}
		if page3.HasMore {
			t.Error("page 3: expected has_more=false")
		}
		if page3.NextCursor != nil {
			t.Error("page 3: expected next_cursor to be nil")
		}

		// Ensure no duplicates across pages
		seen := map[string]bool{}
		for _, sa := range page1.Data {
			seen[sa.StandingApprovalID] = true
		}
		for _, sa := range page2.Data {
			if seen[sa.StandingApprovalID] {
				t.Errorf("duplicate standing_approval_id %s across pages", sa.StandingApprovalID)
			}
			seen[sa.StandingApprovalID] = true
		}
		for _, sa := range page3.Data {
			if seen[sa.StandingApprovalID] {
				t.Errorf("duplicate standing_approval_id %s across pages", sa.StandingApprovalID)
			}
			seen[sa.StandingApprovalID] = true
		}
		if len(seen) != 5 {
			t.Errorf("expected 5 unique approvals across all pages, got %d", len(seen))
		}
	})

	t.Run("InvalidLimit", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)
		uid := testhelper.GenerateUID(t)
		testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

		deps := &Deps{DB: tx, JWTSigningSecret: testJWTSecret}
		router := NewRouter(deps)

		for _, limit := range []string{"0", "-1", "abc", "101"} {
			t.Run("limit="+limit, func(t *testing.T) {
				r := authenticatedRequest(t, http.MethodGet, "/standing-approvals?limit="+limit, uid)
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

		deps := &Deps{DB: tx, JWTSigningSecret: testJWTSecret}
		router := NewRouter(deps)

		for _, cursor := range []string{
			"not-a-cursor",
			"bad-timestamp,sa_123",
			"2026-01-01T00:00:00Z,",
		} {
			t.Run("cursor="+cursor, func(t *testing.T) {
				r := authenticatedRequest(t, http.MethodGet, "/standing-approvals?after="+cursor, uid)
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
		agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

		// Create 3 approvals — should all be returned with default limit=50
		for i := 0; i < 3; i++ {
			testhelper.InsertStandingApproval(t, tx, testhelper.GenerateID(t, "sa_"), agentID, uid)
		}

		deps := &Deps{DB: tx, JWTSigningSecret: testJWTSecret}
		router := NewRouter(deps)

		r := authenticatedRequest(t, http.MethodGet, "/standing-approvals", uid)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		resp := decodeStandingApprovalList(t, w.Body.Bytes())
		if len(resp.Data) != 3 {
			t.Fatalf("expected 3 approvals, got %d", len(resp.Data))
		}
		if resp.HasMore {
			t.Error("expected has_more=false")
		}
	})

	t.Run("EmptyList", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)
		uid := testhelper.GenerateUID(t)
		testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

		deps := &Deps{DB: tx, JWTSigningSecret: testJWTSecret}
		router := NewRouter(deps)

		r := authenticatedRequest(t, http.MethodGet, "/standing-approvals?limit=10", uid)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		resp := decodeStandingApprovalList(t, w.Body.Bytes())
		if len(resp.Data) != 0 {
			t.Errorf("expected 0 approvals, got %d", len(resp.Data))
		}
		if resp.HasMore {
			t.Error("expected has_more=false for empty list")
		}
		if resp.NextCursor != nil {
			t.Error("expected next_cursor to be nil for empty list")
		}
	})
}

// ── POST /standing-approvals/create ──────────────────────────────────────────

func TestCreateStandingApproval_Success(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	standingApprovalTestSetupConnector(t, tx, "email.send")

	deps := &Deps{DB: tx, JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	expiresAt := time.Now().Add(30 * 24 * time.Hour).UTC().Format(time.RFC3339)
	body := fmt.Sprintf(`{
		"agent_id": %d,
		"action_type": "email.send",
		"action_version": "1",
		"constraints": {"recipient_pattern": "*@company.com"},
		"expires_at": "%s"
	}`, agentID, expiresAt)

	r := authenticatedJSONRequest(t, http.MethodPost, "/standing-approvals/create", uid, body)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp standingApprovalResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp.AgentID != agentID {
		t.Errorf("expected agent_id %d, got %d", agentID, resp.AgentID)
	}
	if resp.ActionType != "email.send" {
		t.Errorf("expected action_type 'email.send', got %q", resp.ActionType)
	}
	if resp.Status != "active" {
		t.Errorf("expected status 'active', got %q", resp.Status)
	}
	if resp.Constraints == nil {
		t.Error("expected constraints to be non-nil")
	}
}

func TestCreateStandingApproval_MissingAgentID(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	expiresAt := time.Now().Add(30 * 24 * time.Hour).UTC().Format(time.RFC3339)
	body := `{"action_type": "email.send", "expires_at": "` + expiresAt + `"}`
	r := authenticatedJSONRequest(t, http.MethodPost, "/standing-approvals/create", uid, body)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateStandingApproval_MissingActionType(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	standingApprovalTestSetupConnector(t, tx, "email.send")

	deps := &Deps{DB: tx, JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	expiresAt := time.Now().Add(30 * 24 * time.Hour).UTC().Format(time.RFC3339)
	body := fmt.Sprintf(`{"agent_id": %d, "expires_at": "%s"}`, agentID, expiresAt)
	r := authenticatedJSONRequest(t, http.MethodPost, "/standing-approvals/create", uid, body)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateStandingApproval_AgentNotFound(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	standingApprovalTestSetupConnector(t, tx, "email.send")

	deps := &Deps{DB: tx, JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	expiresAt := time.Now().Add(30 * 24 * time.Hour).UTC().Format(time.RFC3339)
	body := fmt.Sprintf(`{"agent_id": 99999999, "action_type": "email.send", "constraints": {"to": "test@example.com"}, "expires_at": "%s"}`, expiresAt)
	r := authenticatedJSONRequest(t, http.MethodPost, "/standing-approvals/create", uid, body)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateStandingApproval_OtherUsersAgent(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	uid1 := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid1, "u1_"+uid1[:6])

	uid2 := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid2, "u2_"+uid2[:6])

	deps := &Deps{DB: tx, JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	standingApprovalTestSetupConnector(t, tx, "email.send")
	expiresAt := time.Now().Add(30 * 24 * time.Hour).UTC().Format(time.RFC3339)
	body := fmt.Sprintf(`{"agent_id": %d, "action_type": "email.send", "constraints": {"to": "test@example.com"}, "expires_at": "%s"}`, agentID, expiresAt)
	r := authenticatedJSONRequest(t, http.MethodPost, "/standing-approvals/create", uid2, body)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateStandingApproval_LongDurationAllowed(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	standingApprovalTestSetupConnector(t, tx, "email.send")

	deps := &Deps{DB: tx, JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	expiresAt := time.Now().Add(365 * 24 * time.Hour).UTC().Format(time.RFC3339)
	body := fmt.Sprintf(`{"agent_id": %d, "action_type": "email.send", "constraints": {"to": "a@b.com"}, "expires_at": "%s"}`, agentID, expiresAt)
	r := authenticatedJSONRequest(t, http.MethodPost, "/standing-approvals/create", uid, body)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateStandingApproval_ActionTypeTooLong(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	standingApprovalTestSetupConnector(t, tx, "email.send")

	deps := &Deps{DB: tx, JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	longActionType := strings.Repeat("a", 129)
	expiresAt := time.Now().Add(30 * 24 * time.Hour).UTC().Format(time.RFC3339)
	body := fmt.Sprintf(`{"agent_id": %d, "action_type": "%s", "constraints": {"to": "a@b.com"}, "expires_at": "%s"}`, agentID, longActionType, expiresAt)
	r := authenticatedJSONRequest(t, http.MethodPost, "/standing-approvals/create", uid, body)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateStandingApproval_ActionVersionTooLong(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	standingApprovalTestSetupConnector(t, tx, "email.send")

	deps := &Deps{DB: tx, JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	longVersion := strings.Repeat("1", 11)
	expiresAt := time.Now().Add(30 * 24 * time.Hour).UTC().Format(time.RFC3339)
	body := fmt.Sprintf(`{"agent_id": %d, "action_type": "email.send", "action_version": "%s", "constraints": {"to": "a@b.com"}, "expires_at": "%s"}`, agentID, longVersion, expiresAt)
	r := authenticatedJSONRequest(t, http.MethodPost, "/standing-approvals/create", uid, body)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateStandingApproval_ActionVersionInvalidFormat(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	standingApprovalTestSetupConnector(t, tx, "email.send")

	deps := &Deps{DB: tx, JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	expiresAt := time.Now().Add(30 * 24 * time.Hour).UTC().Format(time.RFC3339)
	body := fmt.Sprintf(`{"agent_id": %d, "action_type": "email.send", "action_version": "abc", "constraints": {"to": "a@b.com"}, "expires_at": "%s"}`, agentID, expiresAt)
	r := authenticatedJSONRequest(t, http.MethodPost, "/standing-approvals/create", uid, body)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateStandingApproval_ConstraintsNonObject(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	standingApprovalTestSetupConnector(t, tx, "email.send")

	deps := &Deps{DB: tx, JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	expiresAt := time.Now().Add(30 * 24 * time.Hour).UTC().Format(time.RFC3339)

	// Array is not a valid constraints value.
	body := fmt.Sprintf(`{"agent_id": %d, "action_type": "email.send", "constraints": [1,2,3], "expires_at": "%s"}`, agentID, expiresAt)
	r := authenticatedJSONRequest(t, http.MethodPost, "/standing-approvals/create", uid, body)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for array constraints, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateStandingApproval_ConstraintsNull(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	standingApprovalTestSetupConnector(t, tx, "email.send")

	deps := &Deps{DB: tx, JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	expiresAt := time.Now().Add(30 * 24 * time.Hour).UTC().Format(time.RFC3339)

	// Null constraints are rejected.
	body := fmt.Sprintf(`{"agent_id": %d, "action_type": "email.send", "constraints": null, "expires_at": "%s"}`, agentID, expiresAt)
	r := authenticatedJSONRequest(t, http.MethodPost, "/standing-approvals/create", uid, body)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for null constraints, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateStandingApproval_ConstraintsTooLarge(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	standingApprovalTestSetupConnector(t, tx, "email.send")

	deps := &Deps{DB: tx, JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	// Generate a constraints JSON object larger than 16 KB.
	bigValue := strings.Repeat("x", 16*1024+1)
	expiresAt := time.Now().Add(30 * 24 * time.Hour).UTC().Format(time.RFC3339)
	body := fmt.Sprintf(`{"agent_id": %d, "action_type": "email.send", "constraints": {"k":"%s"}, "expires_at": "%s"}`, agentID, bigValue, expiresAt)
	r := authenticatedJSONRequest(t, http.MethodPost, "/standing-approvals/create", uid, body)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateStandingApproval_ConstraintsOmitted(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	standingApprovalTestSetupConnector(t, tx, "email.send")

	deps := &Deps{DB: tx, JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	expiresAt := time.Now().Add(30 * 24 * time.Hour).UTC().Format(time.RFC3339)

	// Omitting constraints entirely should be rejected.
	body := fmt.Sprintf(`{"agent_id": %d, "action_type": "email.send", "expires_at": "%s"}`, agentID, expiresAt)
	r := authenticatedJSONRequest(t, http.MethodPost, "/standing-approvals/create", uid, body)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for omitted constraints, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateStandingApproval_ConstraintsEmptyObject(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	standingApprovalTestSetupConnector(t, tx, "email.send")

	deps := &Deps{DB: tx, JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	expiresAt := time.Now().Add(30 * 24 * time.Hour).UTC().Format(time.RFC3339)

	body := fmt.Sprintf(`{"agent_id": %d, "action_type": "email.send", "constraints": {}, "expires_at": "%s"}`, agentID, expiresAt)
	r := authenticatedJSONRequest(t, http.MethodPost, "/standing-approvals/create", uid, body)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for unconfirmed empty constraints, got %d: %s", w.Code, w.Body.String())
	}

	confirmed := fmt.Sprintf(`{"agent_id": %d, "action_type": "email.send", "constraints": {}, "expires_at": "%s", "confirm_unrestricted": true}`, agentID, expiresAt)
	r = authenticatedJSONRequest(t, http.MethodPost, "/standing-approvals/create", uid, confirmed)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 for confirmed empty constraints, got %d: %s", w.Code, w.Body.String())
	}
	var resp standingApprovalResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !resp.Unrestricted {
		t.Fatal("expected unrestricted=true for confirmed empty constraints")
	}
}

func TestCreateStandingApproval_ParameterlessActionNullOrEmptyConstraints(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	actionType := "google.list_calendars"
	testhelper.InsertConnector(t, tx, "google")
	testhelper.InsertConnectorAction(t, tx, "google", actionType, actionType)
	deps := &Deps{DB: tx, JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	body := fmt.Sprintf(`{"agent_id": %d, "action_type": %q, "constraints": null, "expires_at": null}`, agentID, actionType)
	r := authenticatedJSONRequest(t, http.MethodPost, "/standing-approvals/create", uid, body)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for null constraints, got %d: %s", w.Code, w.Body.String())
	}

	unconfirmed := fmt.Sprintf(`{"agent_id": %d, "action_type": %q, "constraints": {}, "expires_at": null}`, agentID, actionType)
	r = authenticatedJSONRequest(t, http.MethodPost, "/standing-approvals/create", uid, unconfirmed)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for unconfirmed empty constraints, got %d: %s", w.Code, w.Body.String())
	}

	confirmed := fmt.Sprintf(`{"agent_id": %d, "action_type": %q, "constraints": {}, "expires_at": null, "confirm_unrestricted": true}`, agentID, actionType)
	r = authenticatedJSONRequest(t, http.MethodPost, "/standing-approvals/create", uid, confirmed)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 for confirmed parameterless constraints, got %d: %s", w.Code, w.Body.String())
	}
	var resp standingApprovalResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !resp.Unrestricted {
		t.Fatal("expected unrestricted=true for a parameterless action")
	}
}

func TestCreateStandingApproval_ConstraintsAllWildcard(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	standingApprovalTestSetupConnector(t, tx, "email.send")

	deps := &Deps{DB: tx, JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	expiresAt := time.Now().Add(30 * 24 * time.Hour).UTC().Format(time.RFC3339)

	// All-wildcard constraints require explicit confirmation.
	body := fmt.Sprintf(`{"agent_id": %d, "action_type": "email.send", "constraints": {"to": "*", "subject": "*"}, "expires_at": "%s"}`, agentID, expiresAt)
	r := authenticatedJSONRequest(t, http.MethodPost, "/standing-approvals/create", uid, body)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for unconfirmed unrestricted constraints, got %d: %s", w.Code, w.Body.String())
	}

	confirmed := fmt.Sprintf(`{"agent_id": %d, "action_type": "email.send", "constraints": {"to": "*", "subject": "*"}, "expires_at": "%s", "confirm_unrestricted": true}`, agentID, expiresAt)
	r = authenticatedJSONRequest(t, http.MethodPost, "/standing-approvals/create", uid, confirmed)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 for confirmed unrestricted constraints, got %d: %s", w.Code, w.Body.String())
	}
	var resp standingApprovalResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !resp.Unrestricted {
		t.Fatal("expected unrestricted=true on the created standing approval")
	}
}

func TestCreateStandingApproval_SemanticWildcardPatternRequiresConfirm(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	standingApprovalTestSetupConnector(t, tx, "email.send")

	deps := &Deps{DB: tx, JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	expiresAt := time.Now().Add(30 * 24 * time.Hour).UTC().Format(time.RFC3339)
	body := fmt.Sprintf(`{"agent_id": %d, "action_type": "email.send", "constraints": {"to": {"$pattern": "**"}, "subject": {"$pattern": "*"}}, "expires_at": "%s"}`, agentID, expiresAt)
	r := authenticatedJSONRequest(t, http.MethodPost, "/standing-approvals/create", uid, body)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	confirmed := fmt.Sprintf(`{"agent_id": %d, "action_type": "email.send", "constraints": {"to": {"$pattern": "**"}, "subject": {"$pattern": "*"}}, "expires_at": "%s", "confirm_unrestricted": true}`, agentID, expiresAt)
	r = authenticatedJSONRequest(t, http.MethodPost, "/standing-approvals/create", uid, confirmed)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 for confirmed all-star $pattern constraints, got %d: %s", w.Code, w.Body.String())
	}
	var resp standingApprovalResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !resp.Unrestricted {
		t.Fatal("expected unrestricted=true for all-star $pattern constraints")
	}
}

func TestCreateStandingApproval_ConstraintsMixedWildcardAndFixed(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	standingApprovalTestSetupConnector(t, tx, "email.send")

	deps := &Deps{DB: tx, JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	expiresAt := time.Now().Add(30 * 24 * time.Hour).UTC().Format(time.RFC3339)

	// Mix of wildcard and fixed — should succeed because at least one is non-wildcard.
	body := fmt.Sprintf(`{"agent_id": %d, "action_type": "email.send", "constraints": {"to": "user@example.com", "subject": "*"}, "expires_at": "%s"}`, agentID, expiresAt)
	r := authenticatedJSONRequest(t, http.MethodPost, "/standing-approvals/create", uid, body)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 for mixed constraints, got %d: %s", w.Code, w.Body.String())
	}
	var resp standingApprovalResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Unrestricted {
		t.Fatal("expected unrestricted=false when a non-wildcard constraint is present")
	}
}

func TestCreateStandingApproval_ConstraintsAllNull(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	standingApprovalTestSetupConnector(t, tx, "email.send")

	deps := &Deps{DB: tx, JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	expiresAt := time.Now().Add(30 * 24 * time.Hour).UTC().Format(time.RFC3339)

	// All null values should be treated as wildcards and rejected.
	body := fmt.Sprintf(`{"agent_id": %d, "action_type": "email.send", "constraints": {"repo": null, "title": null}, "expires_at": "%s"}`, agentID, expiresAt)
	r := authenticatedJSONRequest(t, http.MethodPost, "/standing-approvals/create", uid, body)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for all-null constraints, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateStandingApproval_ConstraintsNullAndWildcard(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	standingApprovalTestSetupConnector(t, tx, "email.send")

	deps := &Deps{DB: tx, JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	expiresAt := time.Now().Add(30 * 24 * time.Hour).UTC().Format(time.RFC3339)

	// Mix of null and wildcard — all are effectively unconstrained, should be rejected.
	body := fmt.Sprintf(`{"agent_id": %d, "action_type": "email.send", "constraints": {"repo": null, "title": "*"}, "expires_at": "%s"}`, agentID, expiresAt)
	r := authenticatedJSONRequest(t, http.MethodPost, "/standing-approvals/create", uid, body)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for null+wildcard constraints, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateStandingApproval_ConstraintsNullWithFixed(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	standingApprovalTestSetupConnector(t, tx, "email.send")

	deps := &Deps{DB: tx, JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	expiresAt := time.Now().Add(30 * 24 * time.Hour).UTC().Format(time.RFC3339)

	// Null mixed with a fixed value — null values are rejected outright.
	body := fmt.Sprintf(`{"agent_id": %d, "action_type": "email.send", "constraints": {"repo": null, "title": "my-title"}, "expires_at": "%s"}`, agentID, expiresAt)
	r := authenticatedJSONRequest(t, http.MethodPost, "/standing-approvals/create", uid, body)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for null+fixed constraints, got %d: %s", w.Code, w.Body.String())
	}
}

// standingApprovalTestConnectorOnly inserts connector + action rows without an action_configuration.
func standingApprovalTestConnectorOnly(t *testing.T, tx db.DBTX, actionType string) {
	t.Helper()
	connectorIDPtr := connectorIDFromActionType(actionType)
	if connectorIDPtr == nil {
		t.Fatalf("action type %q has no connector prefix", actionType)
	}
	connectorID := *connectorIDPtr
	testhelper.InsertConnector(t, tx, connectorID)
	testhelper.InsertConnectorAction(t, tx, connectorID, actionType, actionType)
}

func TestCreateStandingApproval_Unauthenticated(t *testing.T) {
	t.Parallel()
	deps := &Deps{JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	r := httptest.NewRequest(http.MethodPost, "/standing-approvals/create", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

// ── POST /standing-approvals/{id}/revoke ─────────────────────────────────────

func TestRevokeStandingApproval_Success(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	saID := testhelper.GenerateID(t, "sa_")
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertStandingApproval(t, tx, saID, agentID, uid)

	deps := &Deps{DB: tx, JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodPost, "/standing-approvals/"+saID+"/revoke", uid)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp revokeStandingApprovalResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp.StandingApprovalID != saID {
		t.Errorf("expected standing_approval_id %q, got %q", saID, resp.StandingApprovalID)
	}
	if resp.Status != "revoked" {
		t.Errorf("expected status 'revoked', got %q", resp.Status)
	}
	if resp.RevokedAt.IsZero() {
		t.Error("expected revoked_at to be set")
	}

	// Verify it no longer appears in active list
	r2 := authenticatedRequest(t, http.MethodGet, "/standing-approvals", uid)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, r2)

	list := decodeStandingApprovalList(t, w2.Body.Bytes())
	for _, sa := range list.Data {
		if sa.StandingApprovalID == saID {
			t.Error("revoked standing approval should not appear in active list")
		}
	}
}

func TestRevokeStandingApproval_NotFound(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodPost, "/standing-approvals/sa_nonexistent/revoke", uid)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRevokeStandingApproval_AlreadyRevoked(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	saID := testhelper.GenerateID(t, "sa_")
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertStandingApprovalWithStatus(t, tx, saID, agentID, uid, "revoked")

	deps := &Deps{DB: tx, JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodPost, "/standing-approvals/"+saID+"/revoke", uid)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", w.Code, w.Body.String())
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to unmarshal error response: %v", err)
	}
	if errResp.Error.Code != ErrApprovalAlreadyResolved {
		t.Errorf("expected error code %q, got %q", ErrApprovalAlreadyResolved, errResp.Error.Code)
	}
}

func TestRevokeStandingApproval_Expired(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	saID := testhelper.GenerateID(t, "sa_")
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertStandingApprovalWithStatus(t, tx, saID, agentID, uid, "expired")

	deps := &Deps{DB: tx, JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodPost, "/standing-approvals/"+saID+"/revoke", uid)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusGone {
		t.Fatalf("expected 410, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRevokeStandingApproval_OtherUsersApproval(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	uid1 := testhelper.GenerateUID(t)
	saID := testhelper.GenerateID(t, "sa_")
	agentID := testhelper.InsertUserWithAgent(t, tx, uid1, "u1_"+uid1[:6])
	testhelper.InsertStandingApproval(t, tx, saID, agentID, uid1)

	uid2 := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid2, "u2_"+uid2[:6])

	deps := &Deps{DB: tx, JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodPost, "/standing-approvals/"+saID+"/revoke", uid2)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRevokeStandingApproval_Unauthenticated(t *testing.T) {
	t.Parallel()
	deps := &Deps{JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	r := httptest.NewRequest(http.MethodPost, "/standing-approvals/sa_xyz/revoke", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

// TestRevokeStandingApproval_ConcurrentRevokes verifies that the TOCTOU-safe
// atomic UPDATE in RevokeStandingApproval works correctly when two requests
// race: exactly one gets 200 (success) and the other gets 409 (already revoked).
func TestRevokeStandingApproval_ConcurrentRevokes(t *testing.T) {
	// Not parallel: InsertStandingApproval commits a fixture connector on the
	// shared test pool; other tests (e.g. GET /connectors) assume no stray rows.
	// Uses the pool (not a rolled-back test transaction) so revoke handlers see committed state.
	pool := testhelper.SetupPool(t)
	uid := testhelper.GenerateUID(t)
	saID := testhelper.GenerateID(t, "sa_")
	agentID := testhelper.InsertUserWithAgent(t, pool, uid, "u_"+uid[:8])
	testhelper.InsertStandingApproval(t, pool, saID, agentID, uid)

	t.Cleanup(func() {
		ctx := context.Background()
		pool.Exec(ctx, "DELETE FROM standing_approvals WHERE standing_approval_id = $1", saID)
		pool.Exec(ctx, "DELETE FROM agents WHERE agent_id = $1", agentID)
		pool.Exec(ctx, "DELETE FROM profiles WHERE id = $1", uid)
	})

	deps := &Deps{DB: pool, JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	// Create both requests on the main goroutine (authenticatedRequest uses t).
	req1 := authenticatedRequest(t, http.MethodPost, "/standing-approvals/"+saID+"/revoke", uid)
	req2 := authenticatedRequest(t, http.MethodPost, "/standing-approvals/"+saID+"/revoke", uid)

	results := make(chan int, 2)
	for _, req := range []*http.Request{req1, req2} {
		go func(r *http.Request) {
			w := httptest.NewRecorder()
			router.ServeHTTP(w, r)
			results <- w.Code
		}(req)
	}

	codes := []int{<-results, <-results}
	sort.Ints(codes)

	// One should succeed (200), the other should get conflict (409).
	if codes[0] != http.StatusOK || codes[1] != http.StatusConflict {
		t.Errorf("expected [200, 409], got %v", codes)
	}
}

func TestCreateStandingApproval_BareStringPatternAutoWrapped(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	standingApprovalTestSetupConnector(t, tx, "github.create_issue")

	deps := &Deps{DB: tx, JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	// Create a standing approval with bare string "[Tracking]*" (not wrapped in $pattern).
	// The backend should auto-normalize this to {"$pattern": "[Tracking]*"}.
	expiresAt := time.Now().Add(30 * 24 * time.Hour).UTC().Format(time.RFC3339)
	body := fmt.Sprintf(`{
		"agent_id": %d,
		"action_type": "github.create_issue",
		"action_version": "1",
		"constraints": {"owner":"testuser","repo":"testrepo","title":"[Tracking]*","body":"*"},
		"expires_at": "%s"
	}`, agentID, expiresAt)

	r := authenticatedJSONRequest(t, http.MethodPost, "/standing-approvals/create", uid, body)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp standingApprovalResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	// Verify the title constraint was auto-wrapped as a pattern.
	constraintsMap, ok := resp.Constraints.(map[string]any)
	if !ok {
		t.Fatalf("expected constraints to be a map, got %T", resp.Constraints)
	}
	titleConstraint, ok := constraintsMap["title"]
	if !ok {
		t.Fatal("expected title constraint to be present")
	}
	// Should be {"$pattern": "[Tracking]*"}, not a bare string.
	patternObj, ok := titleConstraint.(map[string]any)
	if !ok {
		t.Fatalf("expected title constraint to be a pattern object, got %T: %v", titleConstraint, titleConstraint)
	}
	patternValue, ok := patternObj["$pattern"].(string)
	if !ok || patternValue != "[Tracking]*" {
		t.Errorf("expected title pattern to be \"[Tracking]*\", got %v", patternObj["$pattern"])
	}

	// Verify the body constraint remains a bare wildcard "*".
	bodyConstraint, ok := constraintsMap["body"].(string)
	if !ok || bodyConstraint != "*" {
		t.Errorf("expected body constraint to remain \"*\", got %v", constraintsMap["body"])
	}
}

func TestCreateStandingApproval_AutoCreateUnderivableConnector(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	expiresAt := time.Now().Add(30 * 24 * time.Hour).UTC().Format(time.RFC3339)
	body := fmt.Sprintf(`{
		"agent_id": %d,
		"action_type": "malformed_action_type",
		"constraints": {"to": "user@example.com"},
		"expires_at": "%s"
	}`, agentID, expiresAt)
	r := authenticatedJSONRequest(t, http.MethodPost, "/standing-approvals/create", uid, body)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}
