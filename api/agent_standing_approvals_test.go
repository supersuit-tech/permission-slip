package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/db/testhelper"
)

func decodeAgentStandingApprovalList(t *testing.T, body []byte) agentStandingApprovalListResponse {
	t.Helper()
	var resp agentStandingApprovalListResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("failed to unmarshal agent standing approval list response: %v", err)
	}
	return resp
}

// ── GET /agents/{agent_id}/standing-approvals ────────────────────────────────

func TestAgentListStandingApprovals_Success(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	pubKeySSH, privKey, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	agentID := testhelper.InsertAgentWithPublicKey(t, tx, uid, "registered", pubKeySSH)

	saID1 := testhelper.GenerateID(t, "sa_")
	saID2 := testhelper.GenerateID(t, "sa_")
	testhelper.InsertStandingApproval(t, tx, saID1, agentID, uid)
	testhelper.InsertStandingApproval(t, tx, saID2, agentID, uid)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	path := fmt.Sprintf("/agents/%d/standing-approvals", agentID)
	r := signedJSONRequest(t, http.MethodGet, path, "", privKey, agentID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := decodeAgentStandingApprovalList(t, w.Body.Bytes())
	if len(resp.StandingApprovals) != 2 {
		t.Fatalf("expected 2 standing approvals, got %d", len(resp.StandingApprovals))
	}
	for _, sa := range resp.StandingApprovals {
		if sa.Status != "active" {
			t.Errorf("expected status 'active', got %q", sa.Status)
		}
		if sa.AgentID != agentID {
			t.Errorf("expected agent_id %d, got %d", agentID, sa.AgentID)
		}
	}
}

func TestAgentListStandingApprovals_OnlyActiveReturned(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	pubKeySSH, privKey, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	agentID := testhelper.InsertAgentWithPublicKey(t, tx, uid, "registered", pubKeySSH)

	testhelper.InsertStandingApproval(t, tx, testhelper.GenerateID(t, "sa_"), agentID, uid)
	testhelper.InsertStandingApprovalWithStatus(t, tx, testhelper.GenerateID(t, "sa_"), agentID, uid, "revoked")
	testhelper.InsertStandingApprovalWithStatus(t, tx, testhelper.GenerateID(t, "sa_"), agentID, uid, "expired")

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	path := fmt.Sprintf("/agents/%d/standing-approvals", agentID)
	r := signedJSONRequest(t, http.MethodGet, path, "", privKey, agentID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := decodeAgentStandingApprovalList(t, w.Body.Bytes())
	if len(resp.StandingApprovals) != 1 {
		t.Fatalf("expected 1 active standing approval, got %d", len(resp.StandingApprovals))
	}
	if resp.StandingApprovals[0].Status != "active" {
		t.Errorf("expected status 'active', got %q", resp.StandingApprovals[0].Status)
	}
}

func TestAgentListStandingApprovals_Empty(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	pubKeySSH, privKey, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	agentID := testhelper.InsertAgentWithPublicKey(t, tx, uid, "registered", pubKeySSH)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	path := fmt.Sprintf("/agents/%d/standing-approvals", agentID)
	r := signedJSONRequest(t, http.MethodGet, path, "", privKey, agentID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := decodeAgentStandingApprovalList(t, w.Body.Bytes())
	if len(resp.StandingApprovals) != 0 {
		t.Errorf("expected 0 standing approvals, got %d", len(resp.StandingApprovals))
	}
	if resp.HasMore {
		t.Error("expected has_more to be false")
	}
}

func TestAgentListStandingApprovals_AgentCanOnlySeeOwnApprovals(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	// Create user1 with agent1 that has standing approvals.
	uid1 := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid1, "u1_"+uid1[:6])
	pubKey1, _, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	agent1 := testhelper.InsertAgentWithPublicKey(t, tx, uid1, "registered", pubKey1)
	testhelper.InsertStandingApproval(t, tx, testhelper.GenerateID(t, "sa_"), agent1, uid1)

	// Create user2 with agent2 (no standing approvals).
	uid2 := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid2, "u2_"+uid2[:6])
	pubKey2, privKey2, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	agent2 := testhelper.InsertAgentWithPublicKey(t, tx, uid2, "registered", pubKey2)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	// Agent2 should see zero standing approvals (agent1's approvals are not visible).
	path := fmt.Sprintf("/agents/%d/standing-approvals", agent2)
	r := signedJSONRequest(t, http.MethodGet, path, "", privKey2, agent2)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := decodeAgentStandingApprovalList(t, w.Body.Bytes())
	if len(resp.StandingApprovals) != 0 {
		t.Errorf("agent2 should see 0 standing approvals, got %d", len(resp.StandingApprovals))
	}
}

func TestAgentListStandingApprovals_DeactivatedAgentRejected(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	pubKeySSH, privKey, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	agentID := testhelper.InsertAgentWithPublicKey(t, tx, uid, "deactivated", pubKeySSH)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	path := fmt.Sprintf("/agents/%d/standing-approvals", agentID)
	r := signedJSONRequest(t, http.MethodGet, path, "", privKey, agentID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	// requireAgentSignature passes, but handler rejects non-registered agents with 404.
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAgentListStandingApprovals_MissingSigHeader(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	pubKeySSH, _, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	agentID := testhelper.InsertAgentWithPublicKey(t, tx, uid, "registered", pubKeySSH)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	// Request without signature header.
	path := fmt.Sprintf("/agents/%d/standing-approvals", agentID)
	r := httptest.NewRequest(http.MethodGet, path, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAgentListStandingApprovals_AgentIDMismatch(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	pubKeySSH, privKey, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	agentID := testhelper.InsertAgentWithPublicKey(t, tx, uid, "registered", pubKeySSH)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	// Request with correct path agent_id but sign with a different agent_id in header.
	path := fmt.Sprintf("/agents/%d/standing-approvals", agentID)
	r := httptest.NewRequest(http.MethodGet, path, nil)
	SignRequest(privKey, agentID+999, r, nil) // wrong agent_id in header
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to unmarshal error response: %v", err)
	}
	if errResp.Error.Code != ErrAgentIDMismatch {
		t.Errorf("expected error code %q, got %q", ErrAgentIDMismatch, errResp.Error.Code)
	}
}

func TestAgentListStandingApprovals_ResponseShape(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	pubKeySSH, privKey, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	agentID := testhelper.InsertAgentWithPublicKey(t, tx, uid, "registered", pubKeySSH)

	saID := testhelper.GenerateID(t, "sa_")
	testhelper.InsertStandingApproval(t, tx, saID, agentID, uid)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	path := fmt.Sprintf("/agents/%d/standing-approvals", agentID)
	r := signedJSONRequest(t, http.MethodGet, path, "", privKey, agentID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify the response uses "standing_approvals" key (not "data").
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(w.Body.Bytes(), &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := raw["standing_approvals"]; !ok {
		t.Fatal("expected 'standing_approvals' key in response")
	}
	if _, ok := raw["has_more"]; !ok {
		t.Fatal("expected 'has_more' key in response")
	}

	resp := decodeAgentStandingApprovalList(t, w.Body.Bytes())
	if len(resp.StandingApprovals) != 1 {
		t.Fatalf("expected 1 standing approval, got %d", len(resp.StandingApprovals))
	}

	sa := resp.StandingApprovals[0]
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

func TestAgentListStandingApprovals_Pagination(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	pubKeySSH, privKey, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	agentID := testhelper.InsertAgentWithPublicKey(t, tx, uid, "registered", pubKeySSH)

	// Create 3 standing approvals.
	for i := 0; i < 3; i++ {
		testhelper.InsertStandingApproval(t, tx, testhelper.GenerateID(t, "sa_"), agentID, uid)
	}

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	// Request first page with limit=2.
	path := fmt.Sprintf("/agents/%d/standing-approvals?limit=2", agentID)
	r := signedJSONRequest(t, http.MethodGet, path, "", privKey, agentID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	page1 := decodeAgentStandingApprovalList(t, w.Body.Bytes())
	if len(page1.StandingApprovals) != 2 {
		t.Fatalf("expected 2 standing approvals on page 1, got %d", len(page1.StandingApprovals))
	}
	if !page1.HasMore {
		t.Fatal("expected has_more to be true on page 1")
	}
	if page1.NextCursor == nil {
		t.Fatal("expected next_cursor to be set on page 1")
	}

	// Request second page using cursor.
	path2 := fmt.Sprintf("/agents/%d/standing-approvals?limit=2&after=%s", agentID, *page1.NextCursor)
	r2 := signedJSONRequest(t, http.MethodGet, path2, "", privKey, agentID)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, r2)

	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w2.Code, w2.Body.String())
	}

	page2 := decodeAgentStandingApprovalList(t, w2.Body.Bytes())
	if len(page2.StandingApprovals) != 1 {
		t.Fatalf("expected 1 standing approval on page 2, got %d", len(page2.StandingApprovals))
	}
	if page2.HasMore {
		t.Error("expected has_more to be false on page 2")
	}

	// Verify no overlap between pages.
	page1IDs := map[string]bool{}
	for _, sa := range page1.StandingApprovals {
		page1IDs[sa.StandingApprovalID] = true
	}
	for _, sa := range page2.StandingApprovals {
		if page1IDs[sa.StandingApprovalID] {
			t.Errorf("standing approval %q appeared on both pages", sa.StandingApprovalID)
		}
	}
}

func TestAgentListStandingApprovals_InvalidLimit(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	pubKeySSH, privKey, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	agentID := testhelper.InsertAgentWithPublicKey(t, tx, uid, "registered", pubKeySSH)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	// limit=0 should be rejected.
	path := fmt.Sprintf("/agents/%d/standing-approvals?limit=0", agentID)
	r := signedJSONRequest(t, http.MethodGet, path, "", privKey, agentID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for limit=0, got %d: %s", w.Code, w.Body.String())
	}

	// limit=101 should be rejected.
	path2 := fmt.Sprintf("/agents/%d/standing-approvals?limit=101", agentID)
	r2 := signedJSONRequest(t, http.MethodGet, path2, "", privKey, agentID)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, r2)

	if w2.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for limit=101, got %d: %s", w2.Code, w2.Body.String())
	}
}

func TestAgentListStandingApprovals_InvalidCursor(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	pubKeySSH, privKey, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	agentID := testhelper.InsertAgentWithPublicKey(t, tx, uid, "registered", pubKeySSH)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	path := fmt.Sprintf("/agents/%d/standing-approvals?after=invalid-cursor", agentID)
	r := signedJSONRequest(t, http.MethodGet, path, "", privKey, agentID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}
