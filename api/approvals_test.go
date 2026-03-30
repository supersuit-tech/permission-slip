package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip/db/testhelper"
)

func decodeApprovalList(t *testing.T, body []byte) approvalListResponse {
	t.Helper()
	var resp approvalListResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("failed to unmarshal approval list response: %v", err)
	}
	return resp
}

// ── GET /approvals ───────────────────────────────────────────────────────────

func TestListApprovals_Empty(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, "/approvals", uid)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := decodeApprovalList(t, w.Body.Bytes())
	if len(resp.Data) != 0 {
		t.Errorf("expected 0 approvals, got %d", len(resp.Data))
	}
}

func TestListApprovals_ReturnsPendingByDefault(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	// Create 2 pending approvals and 1 approved
	testhelper.InsertApproval(t, tx, testhelper.GenerateID(t, "appr_"), agentID, uid)
	testhelper.InsertApproval(t, tx, testhelper.GenerateID(t, "appr_"), agentID, uid)
	testhelper.InsertApprovalWithStatus(t, tx, testhelper.GenerateID(t, "appr_"), agentID, uid, "approved")

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, "/approvals", uid)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := decodeApprovalList(t, w.Body.Bytes())
	if len(resp.Data) != 2 {
		t.Fatalf("expected 2 pending approvals, got %d", len(resp.Data))
	}
	for _, a := range resp.Data {
		if a.Status != "pending" {
			t.Errorf("expected status 'pending', got %q", a.Status)
		}
	}
}

func TestListApprovals_StatusFilterAll(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	testhelper.InsertApproval(t, tx, testhelper.GenerateID(t, "appr_"), agentID, uid)
	testhelper.InsertApprovalWithStatus(t, tx, testhelper.GenerateID(t, "appr_"), agentID, uid, "approved")
	testhelper.InsertApprovalWithStatus(t, tx, testhelper.GenerateID(t, "appr_"), agentID, uid, "denied")

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, "/approvals?status=all", uid)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := decodeApprovalList(t, w.Body.Bytes())
	if len(resp.Data) != 3 {
		t.Errorf("expected 3 approvals with status=all, got %d", len(resp.Data))
	}
}

func TestListApprovals_InvalidStatusFilter(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, "/approvals?status=invalid", uid)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestListApprovals_DoesNotReturnOtherUsersApprovals(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	uid1 := testhelper.GenerateUID(t)
	agent1 := testhelper.InsertUserWithAgent(t, tx, uid1, "u1_"+uid1[:6])
	testhelper.InsertApproval(t, tx, testhelper.GenerateID(t, "appr_"), agent1, uid1)

	uid2 := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid2, "u2_"+uid2[:6])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	// uid2 should see no approvals
	r := authenticatedRequest(t, http.MethodGet, "/approvals", uid2)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := decodeApprovalList(t, w.Body.Bytes())
	if len(resp.Data) != 0 {
		t.Errorf("expected 0 approvals for other user, got %d", len(resp.Data))
	}
}

func TestListApprovals_ExcludesExpiredPending(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	// Insert an expired pending approval (expires_at in the past)
	testhelper.InsertApprovalWithExpiresAt(t, tx, testhelper.GenerateID(t, "appr_"), agentID, uid, time.Now().Add(-1*time.Hour))

	// Insert a non-expired pending approval
	testhelper.InsertApproval(t, tx, testhelper.GenerateID(t, "appr_"), agentID, uid)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, "/approvals", uid)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := decodeApprovalList(t, w.Body.Bytes())
	if len(resp.Data) != 1 {
		t.Fatalf("expected 1 non-expired approval, got %d", len(resp.Data))
	}
}

func TestListApprovals_Unauthenticated(t *testing.T) {
	t.Parallel()
	deps := &Deps{SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := httptest.NewRequest(http.MethodGet, "/approvals", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestListApprovals_ResponseShape(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	apprID := testhelper.GenerateID(t, "appr_")
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertApproval(t, tx, apprID, agentID, uid)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, "/approvals", uid)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := decodeApprovalList(t, w.Body.Bytes())
	if len(resp.Data) != 1 {
		t.Fatalf("expected 1 approval, got %d", len(resp.Data))
	}

	a := resp.Data[0]
	if a.ApprovalID != apprID {
		t.Errorf("expected approval_id %q, got %q", apprID, a.ApprovalID)
	}
	if a.AgentID != agentID {
		t.Errorf("expected agent_id %d, got %d", agentID, a.AgentID)
	}
	if a.Status != "pending" {
		t.Errorf("expected status 'pending', got %q", a.Status)
	}
	if a.Action == nil {
		t.Error("expected action to be non-nil")
	}
	if a.Context == nil {
		t.Error("expected context to be non-nil")
	}
	if a.ExpiresAt.IsZero() {
		t.Error("expected expires_at to be set")
	}
	if a.CreatedAt.IsZero() {
		t.Error("expected created_at to be set")
	}
}

// ── Pagination ───────────────────────────────────────────────────────────────

func TestListApprovals_Pagination(t *testing.T) {
	t.Parallel()

	t.Run("LimitParam", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)
		uid := testhelper.GenerateUID(t)
		agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

		// Create 3 approvals with distinct created_at times
		for i := 0; i < 3; i++ {
			testhelper.InsertApprovalWithCreatedAt(t, tx, testhelper.GenerateID(t, "appr_"), agentID, uid,
				time.Date(2026, 6, 1+i, 0, 0, 0, 0, time.UTC))
		}

		deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
		router := NewRouter(deps)

		r := authenticatedRequest(t, http.MethodGet, "/approvals?limit=2", uid)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		resp := decodeApprovalList(t, w.Body.Bytes())
		if len(resp.Data) != 2 {
			t.Fatalf("expected 2 approvals, got %d", len(resp.Data))
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

		// Create 5 approvals with distinct created_at times
		for i := 0; i < 5; i++ {
			testhelper.InsertApprovalWithCreatedAt(t, tx, testhelper.GenerateID(t, "appr_"), agentID, uid,
				time.Date(2026, 6, 1+i, 0, 0, 0, 0, time.UTC))
		}

		deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
		router := NewRouter(deps)

		// First page: limit=2
		r := authenticatedRequest(t, http.MethodGet, "/approvals?limit=2", uid)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)

		if w.Code != http.StatusOK {
			t.Fatalf("page 1: expected 200, got %d: %s", w.Code, w.Body.String())
		}

		page1 := decodeApprovalList(t, w.Body.Bytes())
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
		r = authenticatedRequest(t, http.MethodGet, "/approvals?limit=2&after="+url.QueryEscape(*page1.NextCursor), uid)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, r)

		if w.Code != http.StatusOK {
			t.Fatalf("page 2: expected 200, got %d: %s", w.Code, w.Body.String())
		}

		page2 := decodeApprovalList(t, w.Body.Bytes())
		if len(page2.Data) != 2 {
			t.Fatalf("page 2: expected 2 approvals, got %d", len(page2.Data))
		}
		if !page2.HasMore {
			t.Error("page 2: expected has_more=true")
		}

		// Third page: should have 1 remaining
		r = authenticatedRequest(t, http.MethodGet, "/approvals?limit=2&after="+url.QueryEscape(*page2.NextCursor), uid)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, r)

		if w.Code != http.StatusOK {
			t.Fatalf("page 3: expected 200, got %d: %s", w.Code, w.Body.String())
		}

		page3 := decodeApprovalList(t, w.Body.Bytes())
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
		for _, a := range page1.Data {
			seen[a.ApprovalID] = true
		}
		for _, a := range page2.Data {
			if seen[a.ApprovalID] {
				t.Errorf("duplicate approval_id %s across pages", a.ApprovalID)
			}
			seen[a.ApprovalID] = true
		}
		for _, a := range page3.Data {
			if seen[a.ApprovalID] {
				t.Errorf("duplicate approval_id %s across pages", a.ApprovalID)
			}
			seen[a.ApprovalID] = true
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

		deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
		router := NewRouter(deps)

		for _, limit := range []string{"0", "-1", "abc", "101"} {
			t.Run("limit="+limit, func(t *testing.T) {
				r := authenticatedRequest(t, http.MethodGet, "/approvals?limit="+limit, uid)
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
			"bad-timestamp,appr_123",
			"2026-01-01T00:00:00Z,",
		} {
			t.Run("cursor="+cursor, func(t *testing.T) {
				r := authenticatedRequest(t, http.MethodGet, "/approvals?after="+cursor, uid)
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
			testhelper.InsertApprovalWithCreatedAt(t, tx, testhelper.GenerateID(t, "appr_"), agentID, uid,
				time.Date(2026, 6, 1+i, 0, 0, 0, 0, time.UTC))
		}

		deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
		router := NewRouter(deps)

		r := authenticatedRequest(t, http.MethodGet, "/approvals", uid)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		resp := decodeApprovalList(t, w.Body.Bytes())
		if len(resp.Data) != 3 {
			t.Fatalf("expected 3 approvals, got %d", len(resp.Data))
		}
		if resp.HasMore {
			t.Error("expected has_more=false when all approvals fit in one page")
		}
		if resp.NextCursor != nil {
			t.Error("expected next_cursor to be nil when has_more=false")
		}
	})

	t.Run("EmptyPage", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)
		uid := testhelper.GenerateUID(t)
		testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

		deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
		router := NewRouter(deps)

		r := authenticatedRequest(t, http.MethodGet, "/approvals?limit=10", uid)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		resp := decodeApprovalList(t, w.Body.Bytes())
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

// ── POST /approvals/{approval_id}/approve ────────────────────────────────────

func TestApproveApproval_Success(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	apprID := testhelper.GenerateID(t, "appr_")
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertApproval(t, tx, apprID, agentID, uid)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodPost, "/approvals/"+apprID+"/approve", uid)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp approveResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp.ApprovalID != apprID {
		t.Errorf("expected approval_id %q, got %q", apprID, resp.ApprovalID)
	}
	if resp.Status != "approved" {
		t.Errorf("expected status 'approved', got %q", resp.Status)
	}
	if resp.ApprovedAt.IsZero() {
		t.Error("expected approved_at to be set")
	}
	if resp.ConfirmationCode == "" {
		t.Error("expected confirmation_code to be set")
	}
	if len(resp.ConfirmationCode) != 7 || resp.ConfirmationCode[3] != '-' {
		t.Errorf("expected confirmation_code in XXX-XXX format, got %q", resp.ConfirmationCode)
	}

	// Execution should have been attempted (no connector in test → "error").
	if resp.ExecutionStatus == nil {
		t.Fatal("expected execution_status to be set")
	}
	if *resp.ExecutionStatus != "error" {
		t.Errorf("expected execution_status 'error' (no connector in test), got %q", *resp.ExecutionStatus)
	}
	if resp.ExecutionResult == nil {
		t.Fatal("expected execution_result to be set")
	}

	// Verify it no longer appears in pending list
	r2 := authenticatedRequest(t, http.MethodGet, "/approvals", uid)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, r2)

	list := decodeApprovalList(t, w2.Body.Bytes())
	for _, a := range list.Data {
		if a.ApprovalID == apprID {
			t.Error("approved approval should not appear in pending list")
		}
	}
}

func TestApproveApproval_NotFound(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodPost, "/approvals/appr_nonexistent/approve", uid)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestApproveApproval_AlreadyApproved(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	apprID := testhelper.GenerateID(t, "appr_")
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertApprovalWithStatus(t, tx, apprID, agentID, uid, "approved")

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodPost, "/approvals/"+apprID+"/approve", uid)
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

func TestApproveApproval_Expired(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	apprID := testhelper.GenerateID(t, "appr_")
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertApprovalWithExpiresAt(t, tx, apprID, agentID, uid, time.Now().Add(-1*time.Hour))

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodPost, "/approvals/"+apprID+"/approve", uid)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusGone {
		t.Fatalf("expected 410, got %d: %s", w.Code, w.Body.String())
	}
}

func TestApproveApproval_OtherUsersApproval(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	uid1 := testhelper.GenerateUID(t)
	apprID := testhelper.GenerateID(t, "appr_")
	agentID := testhelper.InsertUserWithAgent(t, tx, uid1, "u1_"+uid1[:6])
	testhelper.InsertApproval(t, tx, apprID, agentID, uid1)

	uid2 := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid2, "u2_"+uid2[:6])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	// uid2 tries to approve uid1's approval
	r := authenticatedRequest(t, http.MethodPost, "/approvals/"+apprID+"/approve", uid2)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestApproveApproval_Unauthenticated(t *testing.T) {
	t.Parallel()
	deps := &Deps{SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := httptest.NewRequest(http.MethodPost, "/approvals/appr_xyz/approve", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

// ── POST /approvals/{approval_id}/deny ───────────────────────────────────────

func TestDenyApproval_Success(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	apprID := testhelper.GenerateID(t, "appr_")
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertApproval(t, tx, apprID, agentID, uid)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodPost, "/approvals/"+apprID+"/deny", uid)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp denyResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp.ApprovalID != apprID {
		t.Errorf("expected approval_id %q, got %q", apprID, resp.ApprovalID)
	}
	if resp.Status != "denied" {
		t.Errorf("expected status 'denied', got %q", resp.Status)
	}
	if resp.DeniedAt.IsZero() {
		t.Error("expected denied_at to be set")
	}
}

func TestDenyApproval_NotFound(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodPost, "/approvals/appr_nonexistent/deny", uid)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDenyApproval_AlreadyDenied(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	apprID := testhelper.GenerateID(t, "appr_")
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertApprovalWithStatus(t, tx, apprID, agentID, uid, "denied")

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodPost, "/approvals/"+apprID+"/deny", uid)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDenyApproval_Expired(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	apprID := testhelper.GenerateID(t, "appr_")
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertApprovalWithExpiresAt(t, tx, apprID, agentID, uid, time.Now().Add(-1*time.Hour))

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodPost, "/approvals/"+apprID+"/deny", uid)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusGone {
		t.Fatalf("expected 410, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDenyApproval_OtherUsersApproval(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	uid1 := testhelper.GenerateUID(t)
	apprID := testhelper.GenerateID(t, "appr_")
	agentID := testhelper.InsertUserWithAgent(t, tx, uid1, "u1_"+uid1[:6])
	testhelper.InsertApproval(t, tx, apprID, agentID, uid1)

	uid2 := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid2, "u2_"+uid2[:6])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodPost, "/approvals/"+apprID+"/deny", uid2)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDenyApproval_Unauthenticated(t *testing.T) {
	t.Parallel()
	deps := &Deps{SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := httptest.NewRequest(http.MethodPost, "/approvals/appr_xyz/deny", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}
