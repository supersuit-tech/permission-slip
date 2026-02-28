package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/db/testhelper"
)

func decodeTemplateList(t *testing.T, body []byte) actionConfigTemplateListResponse {
	t.Helper()
	var resp actionConfigTemplateListResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("failed to unmarshal template list response: %v", err)
	}
	return resp
}

// ── GET /action-config-templates ────────────────────────────────────────────

func TestListActionConfigTemplates_Success(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "user_"+uid[:8])

	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertConnectorAction(t, tx, connID, connID+".action_a", "Action A")
	testhelper.InsertConnectorAction(t, tx, connID, connID+".action_b", "Action B")

	desc := "A template description"
	testhelper.InsertActionConfigTemplateFull(t, tx, "tpl_1", connID, connID+".action_a", "Template 1", testhelper.ActionConfigTemplateOpts{
		Description: &desc,
		Parameters:  []byte(`{"repo": "*", "title": "fixed-title"}`),
	})
	testhelper.InsertActionConfigTemplate(t, tx, "tpl_2", connID, connID+".action_b", "Template 2")

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, "/action-config-templates?connector_id="+connID, uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := decodeTemplateList(t, w.Body.Bytes())
	if len(resp.Data) != 2 {
		t.Fatalf("expected 2 templates, got %d", len(resp.Data))
	}

	// First template should be action_a (sorted by action_type).
	if resp.Data[0].ID != "tpl_1" {
		t.Errorf("expected first template ID 'tpl_1', got %q", resp.Data[0].ID)
	}
	if resp.Data[0].Name != "Template 1" {
		t.Errorf("expected name 'Template 1', got %q", resp.Data[0].Name)
	}
	if resp.Data[0].Description == nil || *resp.Data[0].Description != desc {
		t.Errorf("expected description %q, got %v", desc, resp.Data[0].Description)
	}
	if resp.Data[0].ConnectorID != connID {
		t.Errorf("expected connector_id %q, got %q", connID, resp.Data[0].ConnectorID)
	}

	// Check parameters are returned as parsed JSON.
	paramsJSON, err := json.Marshal(resp.Data[0].Parameters)
	if err != nil {
		t.Fatalf("failed to marshal parameters: %v", err)
	}
	if string(paramsJSON) == "{}" {
		t.Error("expected non-empty parameters for first template")
	}
}

func TestListActionConfigTemplates_EmptyForUnknownConnector(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "user_"+uid[:8])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, "/action-config-templates?connector_id=nonexistent", uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := decodeTemplateList(t, w.Body.Bytes())
	if len(resp.Data) != 0 {
		t.Errorf("expected 0 templates, got %d", len(resp.Data))
	}
}

func TestListActionConfigTemplates_MissingConnectorID(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "user_"+uid[:8])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, "/action-config-templates", uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestListActionConfigTemplates_RequiresAuth(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := httptest.NewRequest(http.MethodGet, "/action-config-templates?connector_id=github", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestListActionConfigTemplates_FiltersToConnector(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "user_"+uid[:8])

	conn1 := testhelper.GenerateID(t, "conn_")
	conn2 := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, conn1)
	testhelper.InsertConnector(t, tx, conn2)
	testhelper.InsertConnectorAction(t, tx, conn1, conn1+".action", "Action")
	testhelper.InsertConnectorAction(t, tx, conn2, conn2+".action", "Other")

	testhelper.InsertActionConfigTemplate(t, tx, "tpl_c1", conn1, conn1+".action", "Conn1 Template")
	testhelper.InsertActionConfigTemplate(t, tx, "tpl_c2", conn2, conn2+".action", "Conn2 Template")

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	// Request for conn1 only.
	r := authenticatedRequest(t, http.MethodGet, "/action-config-templates?connector_id="+conn1, uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	resp := decodeTemplateList(t, w.Body.Bytes())
	if len(resp.Data) != 1 {
		t.Fatalf("expected 1 template for conn1, got %d", len(resp.Data))
	}
	if resp.Data[0].ID != "tpl_c1" {
		t.Errorf("expected template 'tpl_c1', got %q", resp.Data[0].ID)
	}
}
