package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/supersuit-tech/permission-slip/db/testhelper"
)

// ── POST /standing-approvals/{id}/execute ────────────────────────────────────

func TestExecuteStandingApproval_Success(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	saID := testhelper.GenerateID(t, "sa_")
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertStandingApproval(t, tx, saID, agentID, uid)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedJSONRequest(t, http.MethodPost, "/standing-approvals/"+saID+"/execute", uid, `{"parameters":{"sender":"*@github.com"}}`)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp executeStandingApprovalResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp.StandingApprovalID != saID {
		t.Errorf("expected standing_approval_id %q, got %q", saID, resp.StandingApprovalID)
	}
	if resp.ExecutionID == 0 {
		t.Error("expected non-zero execution_id")
	}
	if resp.ExecutedAt.IsZero() {
		t.Error("expected non-zero executed_at")
	}

	testhelper.RequireStandingApprovalExecutionCount(t, tx, saID, 1)
}

func TestExecuteStandingApproval_IncrementsCount(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	saID := testhelper.GenerateID(t, "sa_")
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertStandingApproval(t, tx, saID, agentID, uid)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	// Execute twice.
	for i := 0; i < 2; i++ {
		r := authenticatedJSONRequest(t, http.MethodPost, "/standing-approvals/"+saID+"/execute", uid, `{}`)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		if w.Code != http.StatusOK {
			t.Fatalf("execution %d: expected 200, got %d: %s", i+1, w.Code, w.Body.String())
		}
	}

	testhelper.RequireStandingApprovalExecutionCount(t, tx, saID, 2)
}

func TestExecuteStandingApproval_EmitsAuditEvent(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	saID := testhelper.GenerateID(t, "sa_")
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertStandingApproval(t, tx, saID, agentID, uid)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedJSONRequest(t, http.MethodPost, "/standing-approvals/"+saID+"/execute", uid, `{}`)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify an audit event was emitted.
	testhelper.RequireAuditEventCount(t, tx, uid, "standing_approval.executed", 1)
}

func TestExecuteStandingApproval_NotFound(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedJSONRequest(t, http.MethodPost, "/standing-approvals/sa_nonexistent/execute", uid, `{}`)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestExecuteStandingApproval_AlreadyRevoked(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	saID := testhelper.GenerateID(t, "sa_")
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertStandingApprovalWithStatus(t, tx, saID, agentID, uid, "revoked")

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedJSONRequest(t, http.MethodPost, "/standing-approvals/"+saID+"/execute", uid, `{}`)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", w.Code, w.Body.String())
	}
}

func TestExecuteStandingApproval_Expired(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	saID := testhelper.GenerateID(t, "sa_")
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertStandingApprovalWithStatus(t, tx, saID, agentID, uid, "expired")

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedJSONRequest(t, http.MethodPost, "/standing-approvals/"+saID+"/execute", uid, `{}`)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusGone {
		t.Fatalf("expected 410, got %d: %s", w.Code, w.Body.String())
	}
}

func TestExecuteStandingApproval_OtherUsersApproval(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	uid1 := testhelper.GenerateUID(t)
	saID := testhelper.GenerateID(t, "sa_")
	agentID := testhelper.InsertUserWithAgent(t, tx, uid1, "u1_"+uid1[:6])
	testhelper.InsertStandingApproval(t, tx, saID, agentID, uid1)

	uid2 := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid2, "u2_"+uid2[:6])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedJSONRequest(t, http.MethodPost, "/standing-approvals/"+saID+"/execute", uid2, `{}`)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestExecuteStandingApproval_Unauthenticated(t *testing.T) {
	t.Parallel()
	deps := &Deps{SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := httptest.NewRequest(http.MethodPost, "/standing-approvals/sa_xyz/execute", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestExecuteStandingApproval_InvalidParametersType(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	saID := testhelper.GenerateID(t, "sa_")
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertStandingApproval(t, tx, saID, agentID, uid)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	// Array is not a valid parameters value.
	r := authenticatedJSONRequest(t, http.MethodPost, "/standing-approvals/"+saID+"/execute", uid, `{"parameters":[1,2,3]}`)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestExecuteStandingApproval_ParametersTooLarge(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	saID := testhelper.GenerateID(t, "sa_")
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertStandingApproval(t, tx, saID, agentID, uid)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	// Build a parameters object that exceeds 16 KB.
	bigValue := strings.Repeat("x", 17*1024)
	body := `{"parameters":{"data":"` + bigValue + `"}}`
	r := authenticatedJSONRequest(t, http.MethodPost, "/standing-approvals/"+saID+"/execute", uid, body)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestExecuteStandingApproval_NilParameters(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	saID := testhelper.GenerateID(t, "sa_")
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertStandingApproval(t, tx, saID, agentID, uid)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	// Empty body with no parameters should succeed.
	r := authenticatedJSONRequest(t, http.MethodPost, "/standing-approvals/"+saID+"/execute", uid, `{}`)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}
