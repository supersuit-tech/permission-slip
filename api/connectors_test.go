package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/db/testhelper"
)

// ── GET /connectors ─────────────────────────────────────────────────────────

func decodeConnectorList(t *testing.T, body []byte) connectorListResponse {
	t.Helper()
	var resp connectorListResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("failed to unmarshal connector list response: %v", err)
	}
	return resp
}

func TestListConnectors_Empty(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	// No auth required for connectors endpoint
	r := httptest.NewRequest(http.MethodGet, "/connectors", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := decodeConnectorList(t, w.Body.Bytes())
	if len(resp.Data) != 0 {
		t.Errorf("expected 0 connectors, got %d", len(resp.Data))
	}
}

func TestListConnectors_ReturnsAll(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	conn1 := testhelper.GenerateID(t, "conn_")
	conn2 := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, conn1)
	testhelper.InsertConnector(t, tx, conn2)
	testhelper.InsertConnectorAction(t, tx, conn1, "test.act1", "Act 1")
	testhelper.InsertConnectorRequiredCredential(t, tx, conn1, "svc1", "api_key")

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := httptest.NewRequest(http.MethodGet, "/connectors", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := decodeConnectorList(t, w.Body.Bytes())
	if len(resp.Data) != 2 {
		t.Fatalf("expected 2 connectors, got %d", len(resp.Data))
	}
}

func TestListConnectors_NoAuthRequired(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	// Request without Authorization header should still work
	r := httptest.NewRequest(http.MethodGet, "/connectors", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 without auth, got %d: %s", w.Code, w.Body.String())
	}
}

// ── GET /connectors/{connector_id} ──────────────────────────────────────────

func decodeConnectorDetail(t *testing.T, body []byte) connectorDetailResponse {
	t.Helper()
	var resp connectorDetailResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("failed to unmarshal connector detail response: %v", err)
	}
	return resp
}

func TestGetConnector_Found(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertConnectorAction(t, tx, connID, "test.act", "Test Action")
	testhelper.InsertConnectorRequiredCredential(t, tx, connID, "svc", "api_key")

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/connectors/%s", connID), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := decodeConnectorDetail(t, w.Body.Bytes())
	if resp.ID != connID {
		t.Errorf("expected id %q, got %q", connID, resp.ID)
	}
	if len(resp.Actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(resp.Actions))
	}
	if resp.Actions[0].ActionType != "test.act" {
		t.Errorf("expected action_type 'test.act', got %q", resp.Actions[0].ActionType)
	}
	if len(resp.RequiredCredentials) != 1 {
		t.Fatalf("expected 1 credential, got %d", len(resp.RequiredCredentials))
	}
	if resp.RequiredCredentials[0].Service != "svc" {
		t.Errorf("expected service 'svc', got %q", resp.RequiredCredentials[0].Service)
	}
}

func TestGetConnector_NotFound(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := httptest.NewRequest(http.MethodGet, "/connectors/nonexistent", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetConnector_NoAuthRequired(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, connID)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/connectors/%s", connID), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 without auth, got %d: %s", w.Code, w.Body.String())
	}
}
