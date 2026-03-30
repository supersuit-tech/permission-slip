package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/supersuit-tech/permission-slip/db"
	"github.com/supersuit-tech/permission-slip/db/testhelper"
)

// --- Test helpers ---

func decodeActionConfigResponse(t *testing.T, body []byte) actionConfigResponse {
	t.Helper()
	var resp actionConfigResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("failed to unmarshal action config response: %v", err)
	}
	return resp
}

func decodeActionConfigList(t *testing.T, body []byte) actionConfigListResponse {
	t.Helper()
	var resp actionConfigListResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("failed to unmarshal action config list response: %v", err)
	}
	return resp
}

func decodeDeleteActionConfigResponse(t *testing.T, body []byte) deleteActionConfigResponse {
	t.Helper()
	var resp deleteActionConfigResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("failed to unmarshal delete action config response: %v", err)
	}
	return resp
}

// setupActionConfigTest creates a user, registered agent, connector, and connector action.
// Returns (tx, userID, agentID, connectorID, actionType).
func setupActionConfigTest(t *testing.T) (db.DBTX, string, int64, string, string) {
	t.Helper()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	connID := testhelper.GenerateID(t, "conn_")
	actionType := connID + ".test_action"
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertConnectorAction(t, tx, connID, actionType, "Test Action")
	return tx, uid, agentID, connID, actionType
}

// ── POST /action-configurations ─────────────────────────────────────────────

func TestCreateActionConfig_Success(t *testing.T) {
	t.Parallel()
	tx, uid, agentID, connID, actionType := setupActionConfigTest(t)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	body := fmt.Sprintf(`{
		"agent_id": %d,
		"connector_id": %q,
		"action_type": %q,
		"parameters": {"repo": "supersuit-tech/webapp", "title": "*"},
		"name": "Create issues in webapp",
		"description": "Allow agent to create issues with any title"
	}`, agentID, connID, actionType)

	r := authenticatedJSONRequest(t, http.MethodPost, "/action-configurations", uid, body)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	resp := decodeActionConfigResponse(t, w.Body.Bytes())
	if resp.ID == "" {
		t.Error("expected non-empty ID")
	}
	if resp.AgentID != agentID {
		t.Errorf("expected agent_id %d, got %d", agentID, resp.AgentID)
	}
	if resp.ConnectorID != connID {
		t.Errorf("expected connector_id %q, got %q", connID, resp.ConnectorID)
	}
	if resp.ActionType != actionType {
		t.Errorf("expected action_type %q, got %q", actionType, resp.ActionType)
	}
	if resp.Status != "active" {
		t.Errorf("expected status 'active', got %q", resp.Status)
	}
	if resp.Name != "Create issues in webapp" {
		t.Errorf("expected name 'Create issues in webapp', got %q", resp.Name)
	}
}


func TestCreateActionConfig_MissingRequiredFields(t *testing.T) {
	t.Parallel()
	tx, uid, _, _, _ := setupActionConfigTest(t)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	cases := []struct {
		name string
		body string
	}{
		{"missing agent_id", `{"connector_id": "x", "action_type": "x.y", "name": "test"}`},
		{"missing connector_id", `{"agent_id": 1, "action_type": "x.y", "name": "test"}`},
		{"missing action_type", `{"agent_id": 1, "connector_id": "x", "name": "test"}`},
		{"missing name", `{"agent_id": 1, "connector_id": "x", "action_type": "x.y"}`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := authenticatedJSONRequest(t, http.MethodPost, "/action-configurations", uid, tc.body)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, r)
			if w.Code != http.StatusBadRequest {
				t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
			}
		})
	}
}

func TestCreateActionConfig_InvalidParameters(t *testing.T) {
	t.Parallel()
	tx, uid, agentID, connID, actionType := setupActionConfigTest(t)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	body := fmt.Sprintf(`{
		"agent_id": %d,
		"connector_id": %q,
		"action_type": %q,
		"parameters": "not-an-object",
		"name": "Test"
	}`, agentID, connID, actionType)

	r := authenticatedJSONRequest(t, http.MethodPost, "/action-configurations", uid, body)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateActionConfig_PatternWithoutStarRejected(t *testing.T) {
	t.Parallel()
	tx, uid, agentID, connID, actionType := setupActionConfigTest(t)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	body := fmt.Sprintf(`{
		"agent_id": %d,
		"connector_id": %q,
		"action_type": %q,
		"parameters": {"tag": {"$pattern": "hello"}},
		"name": "Test"
	}`, agentID, connID, actionType)

	r := authenticatedJSONRequest(t, http.MethodPost, "/action-configurations", uid, body)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for $pattern without *, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "must contain at least one '*'") {
		t.Errorf("expected error about missing wildcard, got: %s", w.Body.String())
	}
}

func TestCreateActionConfig_AgentNotOwned(t *testing.T) {
	t.Parallel()
	tx, _, _, connID, actionType := setupActionConfigTest(t)

	// Create a different user
	uid2 := testhelper.GenerateUID(t)
	agentID2 := testhelper.InsertUserWithAgent(t, tx, uid2, "u2_"+uid2[:6])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	// uid (first user) tries to create config for agent owned by uid2
	uid3 := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid3, "u3_"+uid3[:6])

	body := fmt.Sprintf(`{
		"agent_id": %d,
		"connector_id": %q,
		"action_type": %q,
		"name": "Test"
	}`, agentID2, connID, actionType)

	r := authenticatedJSONRequest(t, http.MethodPost, "/action-configurations", uid3, body)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for agent not owned by user, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateActionConfig_InvalidConnectorRef(t *testing.T) {
	t.Parallel()
	tx, uid, agentID, _, _ := setupActionConfigTest(t)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	body := fmt.Sprintf(`{
		"agent_id": %d,
		"connector_id": "nonexistent",
		"action_type": "nonexistent.action",
		"name": "Test"
	}`, agentID)

	r := authenticatedJSONRequest(t, http.MethodPost, "/action-configurations", uid, body)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid connector reference, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateActionConfig_Unauthenticated(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := httptest.NewRequest(http.MethodPost, "/action-configurations", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

// ── GET /action-configurations?agent_id={id} ────────────────────────────────

func TestListActionConfigs_Success(t *testing.T) {
	t.Parallel()
	tx, uid, agentID, connID, actionType := setupActionConfigTest(t)

	testhelper.InsertActionConfig(t, tx, testhelper.GenerateID(t, "ac_"), agentID, uid, connID, actionType)
	testhelper.InsertActionConfig(t, tx, testhelper.GenerateID(t, "ac_"), agentID, uid, connID, actionType)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, fmt.Sprintf("/action-configurations?agent_id=%d", agentID), uid)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := decodeActionConfigList(t, w.Body.Bytes())
	if len(resp.Data) != 2 {
		t.Errorf("expected 2 action configs, got %d", len(resp.Data))
	}
}

func TestListActionConfigs_Empty(t *testing.T) {
	t.Parallel()
	tx, uid, agentID, _, _ := setupActionConfigTest(t)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, fmt.Sprintf("/action-configurations?agent_id=%d", agentID), uid)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := decodeActionConfigList(t, w.Body.Bytes())
	if len(resp.Data) != 0 {
		t.Errorf("expected 0 action configs, got %d", len(resp.Data))
	}
}

func TestListActionConfigs_MissingAgentID(t *testing.T) {
	t.Parallel()
	tx, uid, _, _, _ := setupActionConfigTest(t)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, "/action-configurations", uid)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestListActionConfigs_InvalidAgentID(t *testing.T) {
	t.Parallel()
	tx, uid, _, _, _ := setupActionConfigTest(t)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, "/action-configurations?agent_id=notanumber", uid)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestListActionConfigs_DoesNotReturnOtherUsersConfigs(t *testing.T) {
	t.Parallel()
	tx, uid1, agentID1, connID, actionType := setupActionConfigTest(t)

	testhelper.InsertActionConfig(t, tx, testhelper.GenerateID(t, "ac_"), agentID1, uid1, connID, actionType)

	uid2 := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid2, "u2_"+uid2[:6])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, fmt.Sprintf("/action-configurations?agent_id=%d", agentID1), uid2)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := decodeActionConfigList(t, w.Body.Bytes())
	if len(resp.Data) != 0 {
		t.Errorf("expected 0 configs for other user, got %d", len(resp.Data))
	}
}

// ── GET /action-configurations/{config_id} ──────────────────────────────────

func TestGetActionConfig_Success(t *testing.T) {
	t.Parallel()
	tx, uid, agentID, connID, actionType := setupActionConfigTest(t)

	configID := testhelper.GenerateID(t, "ac_")
	desc := "A test configuration"
	testhelper.InsertActionConfigFull(t, tx, configID, agentID, uid, connID, actionType, testhelper.ActionConfigOpts{
		Name:        "Test Config",
		Description: &desc,
		Parameters:  []byte(`{"repo": "test/repo", "title": "*"}`),
	})

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, "/action-configurations/"+configID, uid)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := decodeActionConfigResponse(t, w.Body.Bytes())
	if resp.ID != configID {
		t.Errorf("expected ID %q, got %q", configID, resp.ID)
	}
	if resp.Name != "Test Config" {
		t.Errorf("expected name 'Test Config', got %q", resp.Name)
	}
	if resp.Description == nil || *resp.Description != desc {
		t.Errorf("expected description %q, got %v", desc, resp.Description)
	}
}

func TestGetActionConfig_NotFound(t *testing.T) {
	t.Parallel()
	tx, uid, _, _, _ := setupActionConfigTest(t)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, "/action-configurations/nonexistent", uid)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetActionConfig_OtherUserNotVisible(t *testing.T) {
	t.Parallel()
	tx, uid1, agentID1, connID, actionType := setupActionConfigTest(t)

	configID := testhelper.GenerateID(t, "ac_")
	testhelper.InsertActionConfig(t, tx, configID, agentID1, uid1, connID, actionType)

	uid2 := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid2, "u2_"+uid2[:6])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, "/action-configurations/"+configID, uid2)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for other user's config, got %d: %s", w.Code, w.Body.String())
	}
}

// ── PUT /action-configurations/{config_id} ──────────────────────────────────

func TestUpdateActionConfig_Success(t *testing.T) {
	t.Parallel()
	tx, uid, agentID, connID, actionType := setupActionConfigTest(t)

	configID := testhelper.GenerateID(t, "ac_")
	testhelper.InsertActionConfig(t, tx, configID, agentID, uid, connID, actionType)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	body := `{"name": "Updated Name", "status": "disabled"}`
	r := authenticatedJSONRequest(t, http.MethodPut, "/action-configurations/"+configID, uid, body)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := decodeActionConfigResponse(t, w.Body.Bytes())
	if resp.Name != "Updated Name" {
		t.Errorf("expected name 'Updated Name', got %q", resp.Name)
	}
	if resp.Status != "disabled" {
		t.Errorf("expected status 'disabled', got %q", resp.Status)
	}
}

func TestUpdateActionConfig_PartialUpdate(t *testing.T) {
	t.Parallel()
	tx, uid, agentID, connID, actionType := setupActionConfigTest(t)

	configID := testhelper.GenerateID(t, "ac_")
	testhelper.InsertActionConfigFull(t, tx, configID, agentID, uid, connID, actionType, testhelper.ActionConfigOpts{
		Name:       "Original Name",
		Parameters: []byte(`{"key": "value"}`),
	})

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	// Only update parameters, name and status should be unchanged
	body := `{"parameters": {"new_key": "new_value"}}`
	r := authenticatedJSONRequest(t, http.MethodPut, "/action-configurations/"+configID, uid, body)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := decodeActionConfigResponse(t, w.Body.Bytes())
	if resp.Name != "Original Name" {
		t.Errorf("expected name to remain 'Original Name', got %q", resp.Name)
	}
	if resp.Status != "active" {
		t.Errorf("expected status to remain 'active', got %q", resp.Status)
	}
}

func TestUpdateActionConfig_NotFound(t *testing.T) {
	t.Parallel()
	tx, uid, _, _, _ := setupActionConfigTest(t)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	body := `{"name": "Updated"}`
	r := authenticatedJSONRequest(t, http.MethodPut, "/action-configurations/nonexistent", uid, body)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdateActionConfig_EmptyBody(t *testing.T) {
	t.Parallel()
	tx, uid, agentID, connID, actionType := setupActionConfigTest(t)

	configID := testhelper.GenerateID(t, "ac_")
	testhelper.InsertActionConfig(t, tx, configID, agentID, uid, connID, actionType)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	body := `{}`
	r := authenticatedJSONRequest(t, http.MethodPut, "/action-configurations/"+configID, uid, body)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty update, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdateActionConfig_InvalidStatus(t *testing.T) {
	t.Parallel()
	tx, uid, agentID, connID, actionType := setupActionConfigTest(t)

	configID := testhelper.GenerateID(t, "ac_")
	testhelper.InsertActionConfig(t, tx, configID, agentID, uid, connID, actionType)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	body := `{"status": "invalid_status"}`
	r := authenticatedJSONRequest(t, http.MethodPut, "/action-configurations/"+configID, uid, body)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid status, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdateActionConfig_InvalidCredentialRef(t *testing.T) {
	t.Parallel()
	tx, uid, agentID, connID, actionType := setupActionConfigTest(t)

	configID := testhelper.GenerateID(t, "ac_")
	testhelper.InsertActionConfig(t, tx, configID, agentID, uid, connID, actionType)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	body := `{"credential_id": "nonexistent-cred"}`
	r := authenticatedJSONRequest(t, http.MethodPut, "/action-configurations/"+configID, uid, body)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid credential reference, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdateActionConfig_OtherUserNotVisible(t *testing.T) {
	t.Parallel()
	tx, uid1, agentID1, connID, actionType := setupActionConfigTest(t)

	configID := testhelper.GenerateID(t, "ac_")
	testhelper.InsertActionConfig(t, tx, configID, agentID1, uid1, connID, actionType)

	uid2 := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid2, "u2_"+uid2[:6])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	body := `{"name": "Hacked"}`
	r := authenticatedJSONRequest(t, http.MethodPut, "/action-configurations/"+configID, uid2, body)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for other user's config, got %d: %s", w.Code, w.Body.String())
	}
}

// ── DELETE /action-configurations/{config_id} ───────────────────────────────

func TestDeleteActionConfig_Success(t *testing.T) {
	t.Parallel()
	tx, uid, agentID, connID, actionType := setupActionConfigTest(t)

	configID := testhelper.GenerateID(t, "ac_")
	testhelper.InsertActionConfig(t, tx, configID, agentID, uid, connID, actionType)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodDelete, "/action-configurations/"+configID, uid)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := decodeDeleteActionConfigResponse(t, w.Body.Bytes())
	if resp.ID != configID {
		t.Errorf("expected ID %q, got %q", configID, resp.ID)
	}
	if resp.DeletedAt.IsZero() {
		t.Error("expected non-zero deleted_at")
	}

	// Verify it's actually deleted
	getReq := authenticatedRequest(t, http.MethodGet, "/action-configurations/"+configID, uid)
	getW := httptest.NewRecorder()
	router.ServeHTTP(getW, getReq)
	if getW.Code != http.StatusNotFound {
		t.Fatalf("expected 404 after delete, got %d", getW.Code)
	}
}

func TestDeleteActionConfig_NotFound(t *testing.T) {
	t.Parallel()
	tx, uid, _, _, _ := setupActionConfigTest(t)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodDelete, "/action-configurations/nonexistent", uid)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDeleteActionConfig_OtherUserNotVisible(t *testing.T) {
	t.Parallel()
	tx, uid1, agentID1, connID, actionType := setupActionConfigTest(t)

	configID := testhelper.GenerateID(t, "ac_")
	testhelper.InsertActionConfig(t, tx, configID, agentID1, uid1, connID, actionType)

	uid2 := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid2, "u2_"+uid2[:6])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodDelete, "/action-configurations/"+configID, uid2)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for other user's config, got %d: %s", w.Code, w.Body.String())
	}

	// Verify it wasn't deleted
	getReq := authenticatedRequest(t, http.MethodGet, "/action-configurations/"+configID, uid1)
	getW := httptest.NewRecorder()
	router.ServeHTTP(getW, getReq)
	if getW.Code != http.StatusOK {
		t.Fatalf("expected config to still exist for owner, got %d", getW.Code)
	}
}
