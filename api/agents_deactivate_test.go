package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/db/testhelper"
)

func TestDeactivateAgent(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)
		uid := testhelper.GenerateUID(t)
		testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
		agentID := testhelper.InsertAgentWithStatus(t, tx, uid, "registered")

		deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
		router := NewRouter(deps)

		r := authenticatedRequest(t, http.MethodPost, fmt.Sprintf("/agents/%d/deactivate", agentID), uid)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, r)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var resp agentResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
		if resp.Status != "deactivated" {
			t.Errorf("expected status 'deactivated', got %q", resp.Status)
		}
		if resp.AgentID != agentID {
			t.Errorf("expected agent_id %d, got %d", agentID, resp.AgentID)
		}
		if resp.DeactivatedAt == nil {
			t.Error("expected deactivated_at to be set, got nil")
		}
	})

	t.Run("RevokesStandingApprovals", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)
		uid := testhelper.GenerateUID(t)
		testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
		agentID := testhelper.InsertAgentWithStatus(t, tx, uid, "registered")

		sa1 := testhelper.GenerateID(t, "sa_")
		sa2 := testhelper.GenerateID(t, "sa_")
		testhelper.InsertStandingApproval(t, tx, sa1, agentID, uid)
		testhelper.InsertStandingApprovalWithStatus(t, tx, sa2, agentID, uid, "expired")

		deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
		router := NewRouter(deps)

		r := authenticatedRequest(t, http.MethodPost, fmt.Sprintf("/agents/%d/deactivate", agentID), uid)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, r)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		// Active standing approval should be revoked
		testhelper.RequireRowValue(t, tx, "standing_approvals", "standing_approval_id", sa1, "status", "revoked")
		// Already-expired standing approval should remain unchanged
		testhelper.RequireRowValue(t, tx, "standing_approvals", "standing_approval_id", sa2, "status", "expired")
	})

	t.Run("PendingAgent", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)
		uid := testhelper.GenerateUID(t)
		agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8]) // pending status

		deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
		router := NewRouter(deps)

		r := authenticatedRequest(t, http.MethodPost, fmt.Sprintf("/agents/%d/deactivate", agentID), uid)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, r)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var resp agentResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
		if resp.Status != "deactivated" {
			t.Errorf("expected status 'deactivated', got %q", resp.Status)
		}
	})

	t.Run("AlreadyDeactivated", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)
		uid := testhelper.GenerateUID(t)
		testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
		agentID := testhelper.InsertAgentWithStatus(t, tx, uid, "deactivated")

		deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
		router := NewRouter(deps)

		r := authenticatedRequest(t, http.MethodPost, fmt.Sprintf("/agents/%d/deactivate", agentID), uid)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, r)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("NotFound", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)
		uid := testhelper.GenerateUID(t)
		testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

		deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
		router := NewRouter(deps)

		r := authenticatedRequest(t, http.MethodPost, "/agents/999999/deactivate", uid)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, r)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("OtherUsersAgent", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)

		uid1 := testhelper.GenerateUID(t)
		testhelper.InsertUser(t, tx, uid1, "u1_"+uid1[:6])
		agentID := testhelper.InsertAgentWithStatus(t, tx, uid1, "registered")

		uid2 := testhelper.GenerateUID(t)
		testhelper.InsertUser(t, tx, uid2, "u2_"+uid2[:6])

		deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
		router := NewRouter(deps)

		// uid2 tries to deactivate uid1's agent
		r := authenticatedRequest(t, http.MethodPost, fmt.Sprintf("/agents/%d/deactivate", agentID), uid2)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, r)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("Unauthenticated", func(t *testing.T) {
		t.Parallel()
		deps := &Deps{SupabaseJWTSecret: testJWTSecret}
		router := NewRouter(deps)

		r := httptest.NewRequest(http.MethodPost, "/agents/1/deactivate", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, r)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
		}
	})
}
