package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/db/testhelper"
)

func decodeApplyTemplateResponse(t *testing.T, body []byte) applyActionConfigTemplateResponse {
	t.Helper()
	var resp applyActionConfigTemplateResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("unmarshal apply response: %v", err)
	}
	return resp
}

func TestApplyActionConfigTemplate_Plain(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, connID)
	at := connID + ".plain_action"
	testhelper.InsertConnectorAction(t, tx, connID, at, "Plain")

	testhelper.InsertActionConfigTemplateFull(t, tx, "tpl_plain", connID, at, "Plain tpl", testhelper.ActionConfigTemplateOpts{
		Parameters: []byte(`{"x":"*"}`),
	})

	testhelper.InsertAgentConnector(t, tx, agentID, uid, connID)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	body := fmt.Sprintf(`{"agent_id": %d}`, agentID)
	r := authenticatedJSONRequest(t, http.MethodPost, "/action-config-templates/tpl_plain/apply", uid, body)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	resp := decodeApplyTemplateResponse(t, w.Body.Bytes())
	if resp.StandingApproval != nil {
		t.Fatal("expected no standing approval for plain template")
	}
	if resp.ActionConfiguration.ID == "" {
		t.Fatal("expected action_configuration.id")
	}
}

func TestApplyActionConfigTemplate_WithStandingApproval(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, connID)
	at := connID + ".sa_action"
	testhelper.InsertConnectorAction(t, tx, connID, at, "SA Action")

	testhelper.InsertActionConfigTemplateFull(t, tx, "tpl_sa", connID, at, "SA tpl", testhelper.ActionConfigTemplateOpts{
		Parameters:           []byte(`{"repo":"*"}`),
		StandingApprovalSpec: []byte(`{"duration_days":7}`),
	})

	testhelper.InsertAgentConnector(t, tx, agentID, uid, connID)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	body := fmt.Sprintf(`{"agent_id": %d}`, agentID)
	r := authenticatedJSONRequest(t, http.MethodPost, "/action-config-templates/tpl_sa/apply", uid, body)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	resp := decodeApplyTemplateResponse(t, w.Body.Bytes())
	if resp.StandingApproval == nil {
		t.Fatal("expected standing approval in response")
	}
	if resp.StandingApproval.SourceActionConfigurationID == nil ||
		*resp.StandingApproval.SourceActionConfigurationID != resp.ActionConfiguration.ID {
		t.Fatalf("standing approval should reference created config id")
	}
}

func TestApplyActionConfigTemplate_WithStandingApproval_NonWildcardConstraints(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, connID)
	at := connID + ".sa_action2"
	testhelper.InsertConnectorAction(t, tx, connID, at, "SA Action 2")

	testhelper.InsertActionConfigTemplateFull(t, tx, "tpl_sa2", connID, at, "SA tpl 2", testhelper.ActionConfigTemplateOpts{
		Parameters:           []byte(`{"repo":"my-org/fixture","title":"*"}`),
		StandingApprovalSpec: []byte(`{"duration_days":7}`),
	})

	testhelper.InsertAgentConnector(t, tx, agentID, uid, connID)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	body := fmt.Sprintf(`{"agent_id": %d}`, agentID)
	r := authenticatedJSONRequest(t, http.MethodPost, "/action-config-templates/tpl_sa2/apply", uid, body)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	resp := decodeApplyTemplateResponse(t, w.Body.Bytes())
	if resp.StandingApproval == nil {
		t.Fatal("expected standing approval")
	}
	if resp.StandingApproval.Constraints == nil {
		t.Fatal("expected non-nil constraints for fixed repo")
	}
}

func TestApplyActionConfigTemplate_NotFound(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	body := fmt.Sprintf(`{"agent_id": %d}`, agentID)
	r := authenticatedJSONRequest(t, http.MethodPost, "/action-config-templates/tpl_missing/apply", uid, body)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestApplyActionConfigTemplate_Unauthorized(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := httptest.NewRequest(http.MethodPost, "/action-config-templates/tpl_x/apply", bytes.NewReader([]byte(`{"agent_id":1}`)))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}
