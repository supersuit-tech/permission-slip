package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/db/testhelper"
)

// ── PUT /agents/{agent_id}/connectors/{connector_id}/credential ─────────────

func decodeCredentialResponse(t *testing.T, body []byte) agentConnectorCredentialResponse {
	t.Helper()
	var resp agentConnectorCredentialResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("failed to unmarshal credential response: %v", err)
	}
	return resp
}

func TestAssignCredential_StaticCredential(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertAgentConnector(t, tx, agentID, uid, connID)

	credID := testhelper.GenerateID(t, "cred_")
	testhelper.InsertCredential(t, tx, credID, uid, connID) // service matches connector

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	body := fmt.Sprintf(`{"credential_id":"%s"}`, credID)
	r := authenticatedJSONRequest(t, http.MethodPut,
		fmt.Sprintf("/agents/%d/connectors/%s/credential", agentID, connID), uid, body)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := decodeCredentialResponse(t, w.Body.Bytes())
	if resp.CredentialID == nil || *resp.CredentialID != credID {
		t.Errorf("expected credential_id %q, got %v", credID, resp.CredentialID)
	}
}

func TestAssignCredential_OAuthConnection(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertAgentConnector(t, tx, agentID, uid, connID)

	oauthID := testhelper.GenerateID(t, "oac_")
	testhelper.InsertOAuthConnection(t, tx, oauthID, uid, connID)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	body := fmt.Sprintf(`{"oauth_connection_id":"%s"}`, oauthID)
	r := authenticatedJSONRequest(t, http.MethodPut,
		fmt.Sprintf("/agents/%d/connectors/%s/credential", agentID, connID), uid, body)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := decodeCredentialResponse(t, w.Body.Bytes())
	if resp.OAuthConnectionID == nil || *resp.OAuthConnectionID != oauthID {
		t.Errorf("expected oauth_connection_id %q, got %v", oauthID, resp.OAuthConnectionID)
	}
}

func TestAssignCredential_BothProvided(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	body := `{"credential_id":"cred_1","oauth_connection_id":"oac_1"}`
	r := authenticatedJSONRequest(t, http.MethodPut,
		fmt.Sprintf("/agents/%d/connectors/test/credential", agentID), uid, body)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAssignCredential_NeitherProvided(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	body := `{}`
	r := authenticatedJSONRequest(t, http.MethodPut,
		fmt.Sprintf("/agents/%d/connectors/test/credential", agentID), uid, body)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAssignCredential_CredentialBelongsToOtherUser(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid1 := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid1, "u1_"+uid1[:6])

	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertAgentConnector(t, tx, agentID, uid1, connID)

	// Create credential owned by different user
	uid2 := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid2, "u2_"+uid2[:6])
	credID := testhelper.GenerateID(t, "cred_")
	testhelper.InsertCredential(t, tx, credID, uid2, connID)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	body := fmt.Sprintf(`{"credential_id":"%s"}`, credID)
	r := authenticatedJSONRequest(t, http.MethodPut,
		fmt.Sprintf("/agents/%d/connectors/%s/credential", agentID, connID), uid1, body)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAssignCredential_OAuthBelongsToOtherUser(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid1 := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid1, "u1_"+uid1[:6])

	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertAgentConnector(t, tx, agentID, uid1, connID)

	uid2 := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid2, "u2_"+uid2[:6])
	oauthID := testhelper.GenerateID(t, "oac_")
	testhelper.InsertOAuthConnection(t, tx, oauthID, uid2, connID)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	body := fmt.Sprintf(`{"oauth_connection_id":"%s"}`, oauthID)
	r := authenticatedJSONRequest(t, http.MethodPut,
		fmt.Sprintf("/agents/%d/connectors/%s/credential", agentID, connID), uid1, body)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAssignCredential_ServiceMismatch(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertAgentConnector(t, tx, agentID, uid, connID)

	// Credential for a different service
	credID := testhelper.GenerateID(t, "cred_")
	testhelper.InsertCredential(t, tx, credID, uid, "other_service")

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	body := fmt.Sprintf(`{"credential_id":"%s"}`, credID)
	r := authenticatedJSONRequest(t, http.MethodPut,
		fmt.Sprintf("/agents/%d/connectors/%s/credential", agentID, connID), uid, body)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for service mismatch, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAssignCredential_OAuthProviderMismatch(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertAgentConnector(t, tx, agentID, uid, connID)

	// OAuth connection for a different provider
	oauthID := testhelper.GenerateID(t, "oac_")
	testhelper.InsertOAuthConnection(t, tx, oauthID, uid, "other_provider")

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	body := fmt.Sprintf(`{"oauth_connection_id":"%s"}`, oauthID)
	r := authenticatedJSONRequest(t, http.MethodPut,
		fmt.Sprintf("/agents/%d/connectors/%s/credential", agentID, connID), uid, body)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for provider mismatch, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAssignCredential_WrongOwner(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid1 := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid1, "u1_"+uid1[:6])

	uid2 := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid2, "u2_"+uid2[:6])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	body := `{"credential_id":"cred_1"}`
	r := authenticatedJSONRequest(t, http.MethodPut,
		fmt.Sprintf("/agents/%d/connectors/test/credential", agentID), uid2, body)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for wrong owner, got %d: %s", w.Code, w.Body.String())
	}
}

// ── GET /agents/{agent_id}/connectors/{connector_id}/credential ─────────────

func TestGetCredentialBinding_NoBinding(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet,
		fmt.Sprintf("/agents/%d/connectors/test/credential", agentID), uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := decodeCredentialResponse(t, w.Body.Bytes())
	if resp.CredentialID != nil {
		t.Errorf("expected nil credential_id, got %v", resp.CredentialID)
	}
	if resp.OAuthConnectionID != nil {
		t.Errorf("expected nil oauth_connection_id, got %v", resp.OAuthConnectionID)
	}
}

func TestGetCredentialBinding_WithBinding(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertAgentConnector(t, tx, agentID, uid, connID)

	credID := testhelper.GenerateID(t, "cred_")
	testhelper.InsertCredential(t, tx, credID, uid, connID)

	// First assign
	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	body := fmt.Sprintf(`{"credential_id":"%s"}`, credID)
	r := authenticatedJSONRequest(t, http.MethodPut,
		fmt.Sprintf("/agents/%d/connectors/%s/credential", agentID, connID), uid, body)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("assign: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Then GET
	r = authenticatedRequest(t, http.MethodGet,
		fmt.Sprintf("/agents/%d/connectors/%s/credential", agentID, connID), uid)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("get: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := decodeCredentialResponse(t, w.Body.Bytes())
	if resp.CredentialID == nil || *resp.CredentialID != credID {
		t.Errorf("expected credential_id %q, got %v", credID, resp.CredentialID)
	}
}

func TestGetCredentialBinding_WrongOwner(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid1 := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid1, "u1_"+uid1[:6])

	uid2 := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid2, "u2_"+uid2[:6])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet,
		fmt.Sprintf("/agents/%d/connectors/test/credential", agentID), uid2)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for wrong owner, got %d: %s", w.Code, w.Body.String())
	}
}

// ── DELETE /agents/{agent_id}/connectors/{connector_id}/credential ──────────

func TestRemoveCredentialBinding_Success(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertAgentConnector(t, tx, agentID, uid, connID)

	credID := testhelper.GenerateID(t, "cred_")
	testhelper.InsertCredential(t, tx, credID, uid, connID)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	// Assign first
	body := fmt.Sprintf(`{"credential_id":"%s"}`, credID)
	r := authenticatedJSONRequest(t, http.MethodPut,
		fmt.Sprintf("/agents/%d/connectors/%s/credential", agentID, connID), uid, body)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("assign: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Delete
	r = authenticatedRequest(t, http.MethodDelete,
		fmt.Sprintf("/agents/%d/connectors/%s/credential", agentID, connID), uid)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("delete: expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRemoveCredentialBinding_NotFound(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodDelete,
		fmt.Sprintf("/agents/%d/connectors/nonexistent/credential", agentID), uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRemoveCredentialBinding_WrongOwner(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid1 := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid1, "u1_"+uid1[:6])

	uid2 := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid2, "u2_"+uid2[:6])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodDelete,
		fmt.Sprintf("/agents/%d/connectors/test/credential", agentID), uid2)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for wrong owner, got %d: %s", w.Code, w.Body.String())
	}
}
