package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip/db/testhelper"
)

func TestRequestApproval_AutoApprove_DataWindowInjectsStart(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	testhelper.InsertConnector(t, tx, "imessage")
	schema := []byte(`{"type":"object","properties":{"chat_id":{"type":"integer"},"start":{"type":"string"},"end":{"type":"string"}}}`)
	testhelper.InsertConnectorActionWithDataWindow(t, tx, "imessage", "imessage.read_history", "Read History", schema,
		`{"start_param":"start","end_param":"end"}`)

	pubKeySSH, privKey, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	agentID := testhelper.InsertAgentWithPublicKey(t, tx, uid, "registered", pubKeySSH)

	saID := testhelper.GenerateID(t, "sa_")
	testhelper.InsertStandingApprovalFull(t, tx, saID, agentID, uid, testhelper.StandingApprovalOpts{
		ActionType:  "imessage.read_history",
		Constraints: []byte(`{"chat_id":42,"$data_window":{"last_days":30}}`),
	})

	router := NewRouter(testDepsForDB(t, tx))

	reqBody := `{"request_id":"dw-inject-001","action":{"type":"imessage.read_history","version":"1","parameters":{"chat_id":42}},"context":{"description":"test"}}`
	r := signedJSONRequest(t, http.MethodPost, "/approvals/request", reqBody, privKey, agentID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var params []byte
	err = tx.QueryRow(context.Background(),
		`SELECT parameters FROM standing_approval_executions WHERE standing_approval_id = $1 ORDER BY id DESC LIMIT 1`,
		saID,
	).Scan(&params)
	if err != nil {
		t.Fatalf("query execution params: %v", err)
	}
	var recorded map[string]any
	if err := json.Unmarshal(params, &recorded); err != nil {
		t.Fatalf("unmarshal params: %v", err)
	}
	startStr, ok := recorded["start"].(string)
	if !ok || startStr == "" {
		t.Fatalf("expected injected start param, got %v", recorded["start"])
	}
	startAt, err := time.Parse(time.RFC3339, startStr)
	if err != nil {
		t.Fatalf("parse start: %v", err)
	}
	age := time.Now().UTC().Sub(startAt)
	if age < 29*24*time.Hour || age > 31*24*time.Hour {
		t.Fatalf("start %q is %v old, want ~30 days", startStr, age)
	}
}

func TestRequestApproval_AutoApprove_DataWindowUnsupported_FallsThrough(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	testhelper.InsertConnector(t, tx, "imessage")
	schema := []byte(`{"type":"object","properties":{"query":{"type":"string"}}}`)
	testhelper.InsertConnectorActionFull(t, tx, "imessage", "imessage.search", "Search", testhelper.ConnectorActionOpts{
		ParametersSchema: schema,
	})

	pubKeySSH, privKey, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	agentID := testhelper.InsertAgentWithPublicKey(t, tx, uid, "registered", pubKeySSH)

	saID := testhelper.GenerateID(t, "sa_")
	// Bypass API validation: corrupt/defensive case with $data_window on unsupported action.
	testhelper.InsertStandingApprovalFull(t, tx, saID, agentID, uid, testhelper.StandingApprovalOpts{
		ActionType:  "imessage.search",
		Constraints: []byte(`{"query":"hello","$data_window":{"last_days":30}}`),
	})

	router := NewRouter(testDepsForDB(t, tx))
	reqBody := `{"request_id":"dw-unsupported-001","action":{"type":"imessage.search","version":"1","parameters":{"query":"hello"}},"context":{"description":"test"}}`
	r := signedJSONRequest(t, http.MethodPost, "/approvals/request", reqBody, privKey, agentID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp agentRequestApprovalResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Status != "pending" {
		t.Fatalf("expected pending (fail closed), got %q", resp.Status)
	}
	testhelper.RequireStandingApprovalExecutionCount(t, tx, saID, 0)
}

func TestValidateStandingApprovalConstraintKeys_RejectsDataWindowOnUnsupportedAction(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	testhelper.InsertConnector(t, tx, "imessage")
	schema := []byte(`{"type":"object","properties":{"query":{"type":"string"}}}`)
	testhelper.InsertConnectorActionFull(t, tx, "imessage", "imessage.search", "Search", testhelper.ConnectorActionOpts{
		ParametersSchema: schema,
	})

	constraints := []byte(`{"query":"*","$data_window":{"last_days":30}}`)
	err := validateStandingApprovalConstraintKeys(context.Background(), tx, nil, "imessage.search", constraints)
	if err == nil {
		t.Fatal("expected rejection for unsupported $data_window")
	}
}

func TestValidateStandingApprovalConstraintKeys_AllowsDataWindowOnSupportedAction(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	testhelper.InsertConnector(t, tx, "imessage")
	schema := []byte(`{"type":"object","properties":{"chat_id":{"type":"integer"},"start":{"type":"string"},"end":{"type":"string"}}}`)
	testhelper.InsertConnectorActionWithDataWindow(t, tx, "imessage", "imessage.read_history", "Read History", schema,
		`{"start_param":"start","end_param":"end"}`)

	constraints := []byte(`{"chat_id":42,"$data_window":{"last_days":30}}`)
	if err := validateStandingApprovalConstraintKeys(context.Background(), tx, nil, "imessage.read_history", constraints); err != nil {
		t.Fatalf("expected valid constraints, got: %v", err)
	}
}

func TestValidateStandingApprovalConstraintsForAction_RejectsInvalidDataWindow(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	testhelper.InsertConnector(t, tx, "imessage")
	schema := []byte(`{"type":"object","properties":{"chat_id":{"type":"integer"}}}`)
	testhelper.InsertConnectorActionWithDataWindow(t, tx, "imessage", "imessage.read_history", "Read History", schema,
		`{"start_param":"start","end_param":"end"}`)

	raw := json.RawMessage(`{"chat_id":42,"$data_window":{"last_days":0}}`)
	_, err := validateStandingApprovalConstraintsForAction(context.Background(), tx, nil, "imessage.read_history", raw)
	if err == nil {
		t.Fatal("expected invalid last_days rejection")
	}
}
