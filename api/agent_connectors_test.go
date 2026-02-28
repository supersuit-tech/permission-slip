package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/db/testhelper"
)

// ── GET /agents/{agent_id}/connectors ───────────────────────────────────────

func decodeAgentConnectorList(t *testing.T, body []byte) agentConnectorListResponse {
	t.Helper()
	var resp agentConnectorListResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("failed to unmarshal agent connector list response: %v", err)
	}
	return resp
}

func TestListAgentConnectors_Empty(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, fmt.Sprintf("/agents/%d/connectors", agentID), uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := decodeAgentConnectorList(t, w.Body.Bytes())
	if len(resp.Data) != 0 {
		t.Errorf("expected 0 connectors, got %d", len(resp.Data))
	}
}

func TestListAgentConnectors_ReturnsEnabledConnectors(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertConnectorAction(t, tx, connID, "test.act", "Test Act")
	testhelper.InsertAgentConnector(t, tx, agentID, uid, connID)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, fmt.Sprintf("/agents/%d/connectors", agentID), uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := decodeAgentConnectorList(t, w.Body.Bytes())
	if len(resp.Data) != 1 {
		t.Fatalf("expected 1 connector, got %d", len(resp.Data))
	}
	if resp.Data[0].ID != connID {
		t.Errorf("expected connector id %q, got %q", connID, resp.Data[0].ID)
	}
	if len(resp.Data[0].Actions) != 1 {
		t.Errorf("expected 1 action, got %d", len(resp.Data[0].Actions))
	}
}

func TestListAgentConnectors_Unauthenticated(t *testing.T) {
	t.Parallel()
	deps := &Deps{SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := httptest.NewRequest(http.MethodGet, "/agents/1/connectors", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestListAgentConnectors_WrongOwner(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid1 := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid1, "u1_"+uid1[:6])

	uid2 := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid2, "u2_"+uid2[:6])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, fmt.Sprintf("/agents/%d/connectors", agentID), uid2)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for wrong owner, got %d: %s", w.Code, w.Body.String())
	}
}

func TestListAgentConnectors_InvalidAgentID(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	for _, id := range []string{"abc", "0", "-1"} {
		t.Run("id="+id, func(t *testing.T) {
			r := authenticatedRequest(t, http.MethodGet, "/agents/"+id+"/connectors", uid)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, r)

			if w.Code != http.StatusBadRequest {
				t.Errorf("expected 400 for id=%s, got %d: %s", id, w.Code, w.Body.String())
			}
		})
	}
}

// ── PUT /agents/{agent_id}/connectors/{connector_id} ────────────────────────

func decodeAgentConnectorResponse(t *testing.T, body []byte) agentConnectorResponse {
	t.Helper()
	var resp agentConnectorResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("failed to unmarshal agent connector response: %v", err)
	}
	return resp
}

func TestEnableAgentConnector_Success(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, connID)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodPut, fmt.Sprintf("/agents/%d/connectors/%s", agentID, connID), uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := decodeAgentConnectorResponse(t, w.Body.Bytes())
	if resp.AgentID != agentID {
		t.Errorf("expected agent_id %d, got %d", agentID, resp.AgentID)
	}
	if resp.ConnectorID != connID {
		t.Errorf("expected connector_id %q, got %q", connID, resp.ConnectorID)
	}
}

func TestEnableAgentConnector_Idempotent(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, connID)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	// First enable
	r := authenticatedRequest(t, http.MethodPut, fmt.Sprintf("/agents/%d/connectors/%s", agentID, connID), uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("first enable: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	resp1 := decodeAgentConnectorResponse(t, w.Body.Bytes())

	// Second enable (should return 200 with same enabled_at)
	r = authenticatedRequest(t, http.MethodPut, fmt.Sprintf("/agents/%d/connectors/%s", agentID, connID), uid)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("second enable: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	resp2 := decodeAgentConnectorResponse(t, w.Body.Bytes())

	if !resp1.EnabledAt.Equal(resp2.EnabledAt) {
		t.Errorf("expected same enabled_at on idempotent enable, got %v and %v", resp1.EnabledAt, resp2.EnabledAt)
	}
}

func TestEnableAgentConnector_AgentNotFound(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodPut, "/agents/999999/connectors/test", uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestEnableAgentConnector_ConnectorNotFound(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodPut, fmt.Sprintf("/agents/%d/connectors/nonexistent", agentID), uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestEnableAgentConnector_WrongOwner(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid1 := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid1, "u1_"+uid1[:6])

	uid2 := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid2, "u2_"+uid2[:6])

	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, connID)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodPut, fmt.Sprintf("/agents/%d/connectors/%s", agentID, connID), uid2)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for wrong owner, got %d: %s", w.Code, w.Body.String())
	}
}

func TestEnableAgentConnector_Unauthenticated(t *testing.T) {
	t.Parallel()
	deps := &Deps{SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := httptest.NewRequest(http.MethodPut, "/agents/1/connectors/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

// ── DELETE /agents/{agent_id}/connectors/{connector_id} ─────────────────────

func decodeDisableAgentConnectorResponse(t *testing.T, body []byte) disableAgentConnectorResponse {
	t.Helper()
	var resp disableAgentConnectorResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("failed to unmarshal disable agent connector response: %v", err)
	}
	return resp
}

func TestDisableAgentConnector_Success(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertAgentConnector(t, tx, agentID, uid, connID)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodDelete, fmt.Sprintf("/agents/%d/connectors/%s", agentID, connID), uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := decodeDisableAgentConnectorResponse(t, w.Body.Bytes())
	if resp.AgentID != agentID {
		t.Errorf("expected agent_id %d, got %d", agentID, resp.AgentID)
	}
	if resp.ConnectorID != connID {
		t.Errorf("expected connector_id %q, got %q", connID, resp.ConnectorID)
	}
	if resp.RevokedStandingApprovals != 0 {
		t.Errorf("expected 0 revoked standing approvals, got %d", resp.RevokedStandingApprovals)
	}
}

func TestDisableAgentConnector_NotEnabled(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodDelete, fmt.Sprintf("/agents/%d/connectors/nonexistent", agentID), uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDisableAgentConnector_WrongOwner(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid1 := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid1, "u1_"+uid1[:6])

	uid2 := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid2, "u2_"+uid2[:6])

	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertAgentConnector(t, tx, agentID, uid1, connID)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodDelete, fmt.Sprintf("/agents/%d/connectors/%s", agentID, connID), uid2)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for wrong owner, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDisableAgentConnector_Unauthenticated(t *testing.T) {
	t.Parallel()
	deps := &Deps{SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := httptest.NewRequest(http.MethodDelete, "/agents/1/connectors/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDisableAgentConnector_RevokesStandingApprovals(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertConnectorAction(t, tx, connID, "test.action1", "Action 1")
	testhelper.InsertAgentConnector(t, tx, agentID, uid, connID)

	// Create an active standing approval for the connector's action
	sa1 := testhelper.GenerateID(t, "sa_")
	testhelper.InsertStandingApprovalWithActionType(t, tx, sa1, agentID, uid, "test.action1")

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodDelete, fmt.Sprintf("/agents/%d/connectors/%s", agentID, connID), uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := decodeDisableAgentConnectorResponse(t, w.Body.Bytes())
	if resp.RevokedStandingApprovals != 1 {
		t.Errorf("expected 1 revoked standing approval, got %d", resp.RevokedStandingApprovals)
	}
}
