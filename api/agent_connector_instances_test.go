package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/db/testhelper"
)

func decodeInstanceList(t *testing.T, body []byte) agentConnectorInstanceListResponse {
	t.Helper()
	var resp agentConnectorInstanceListResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("unmarshal instance list: %v", err)
	}
	return resp
}

func TestListAgentConnectorInstances_Default(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertAgentConnector(t, tx, agentID, uid, connID)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, fmt.Sprintf("/agents/%d/connectors/%s/instances", agentID, connID), uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	list := decodeInstanceList(t, w.Body.Bytes())
	if len(list.Data) != 1 {
		t.Fatalf("expected 1 instance, got %d", len(list.Data))
	}
	if !list.Data[0].IsDefault {
		t.Error("expected default instance")
	}
}

func TestCreateAgentConnectorInstance_Second(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertAgentConnector(t, tx, agentID, uid, connID)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequestWithBody(t, http.MethodPost, fmt.Sprintf("/agents/%d/connectors/%s/instances", agentID, connID), uid, []byte(`{"label":"Sales"}`))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	r2 := authenticatedRequest(t, http.MethodGet, fmt.Sprintf("/agents/%d/connectors/%s/instances", agentID, connID), uid)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, r2)
	if w2.Code != http.StatusOK {
		t.Fatalf("list: %d %s", w2.Code, w2.Body.String())
	}
	list := decodeInstanceList(t, w2.Body.Bytes())
	if len(list.Data) != 2 {
		t.Fatalf("expected 2 instances, got %d", len(list.Data))
	}
}

func TestPatchAgentConnectorInstance_SetDefault(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertAgentConnector(t, tx, agentID, uid, connID)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r1 := authenticatedRequestWithBody(t, http.MethodPost, fmt.Sprintf("/agents/%d/connectors/%s/instances", agentID, connID), uid, []byte(`{"label":"Sales"}`))
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, r1)
	if w1.Code != http.StatusCreated {
		t.Fatalf("create second: %d %s", w1.Code, w1.Body.String())
	}
	var created agentConnectorInstanceResponse
	if err := json.Unmarshal(w1.Body.Bytes(), &created); err != nil {
		t.Fatalf("unmarshal created: %v", err)
	}

	patchPayload, err := json.Marshal(map[string]bool{"is_default": true})
	if err != nil {
		t.Fatal(err)
	}
	r2 := authenticatedRequestWithBody(t, http.MethodPatch,
		fmt.Sprintf("/agents/%d/connectors/%s/instances/%s", agentID, connID, created.ConnectorInstanceID), uid, patchPayload)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, r2)
	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w2.Code, w2.Body.String())
	}
	var patched agentConnectorInstanceResponse
	if err := json.Unmarshal(w2.Body.Bytes(), &patched); err != nil {
		t.Fatalf("unmarshal patched: %v", err)
	}
	if !patched.IsDefault {
		t.Error("expected patched instance to be default")
	}
}

func TestPatchAgentConnectorInstance_DuplicateLabel409(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertAgentConnector(t, tx, agentID, uid, connID)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r0 := authenticatedRequest(t, http.MethodGet, fmt.Sprintf("/agents/%d/connectors/%s/instances", agentID, connID), uid)
	w0 := httptest.NewRecorder()
	router.ServeHTTP(w0, r0)
	list := decodeInstanceList(t, w0.Body.Bytes())
	if len(list.Data) != 1 {
		t.Fatalf("expected 1 default instance")
	}
	defaultLabel := list.Data[0].Label

	// Create second instance
	r1 := authenticatedRequestWithBody(t, http.MethodPost, fmt.Sprintf("/agents/%d/connectors/%s/instances", agentID, connID), uid, []byte(`{"label":"Sales"}`))
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, r1)
	if w1.Code != http.StatusCreated {
		t.Fatalf("create second: %d %s", w1.Code, w1.Body.String())
	}
	var created agentConnectorInstanceResponse
	if err := json.Unmarshal(w1.Body.Bytes(), &created); err != nil {
		t.Fatalf("unmarshal created: %v", err)
	}

	patchPayload, err := json.Marshal(map[string]string{"label": defaultLabel})
	if err != nil {
		t.Fatal(err)
	}
	r2 := authenticatedRequestWithBody(t, http.MethodPatch,
		fmt.Sprintf("/agents/%d/connectors/%s/instances/%s", agentID, connID, created.ConnectorInstanceID), uid, patchPayload)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, r2)
	if w2.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", w2.Code, w2.Body.String())
	}
}

func TestGetAgentConnectorInstance_InvalidUUID400(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertAgentConnector(t, tx, agentID, uid, connID)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet,
		fmt.Sprintf("/agents/%d/connectors/%s/instances/not-a-uuid", agentID, connID), uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDeleteAgentConnectorInstance_SecondInstance204(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertAgentConnector(t, tx, agentID, uid, connID)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r1 := authenticatedRequestWithBody(t, http.MethodPost, fmt.Sprintf("/agents/%d/connectors/%s/instances", agentID, connID), uid, []byte(`{"label":"Extra"}`))
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, r1)
	if w1.Code != http.StatusCreated {
		t.Fatalf("create second: %d %s", w1.Code, w1.Body.String())
	}
	var created agentConnectorInstanceResponse
	if err := json.Unmarshal(w1.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}

	rDel := authenticatedRequest(t, http.MethodDelete,
		fmt.Sprintf("/agents/%d/connectors/%s/instances/%s", agentID, connID, created.ConnectorInstanceID), uid)
	wDel := httptest.NewRecorder()
	router.ServeHTTP(wDel, rDel)
	if wDel.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", wDel.Code, wDel.Body.String())
	}

	rList := authenticatedRequest(t, http.MethodGet, fmt.Sprintf("/agents/%d/connectors/%s/instances", agentID, connID), uid)
	wList := httptest.NewRecorder()
	router.ServeHTTP(wList, rList)
	list := decodeInstanceList(t, wList.Body.Bytes())
	if len(list.Data) != 1 {
		t.Fatalf("after delete expected 1 instance, got %d", len(list.Data))
	}
}
