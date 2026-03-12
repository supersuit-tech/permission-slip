package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/db/testhelper"
)

func TestAgentRequestApproval_MissingRequiredParameter(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	pubKeySSH, privKey, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	agentID := testhelper.InsertAgentWithPublicKey(t, tx, uid, "registered", pubKeySSH)

	// Create a connector with an action that has required fields.
	testhelper.InsertConnector(t, tx, "testconn")
	testhelper.InsertConnectorActionFull(t, tx, "testconn", "testconn.do_thing", "Do Thing", testhelper.ConnectorActionOpts{
		ParametersSchema: []byte(`{
			"type": "object",
			"required": ["name", "target"],
			"properties": {
				"name":   {"type": "string"},
				"target": {"type": "string"},
				"note":   {"type": "string"}
			}
		}`),
	})

	deps := testDepsForDB(t, tx)
	router := NewRouter(deps)

	// Submit with missing "target" (only "name" present).
	reqBody := `{"request_id":"req_missing_param","action":{"type":"testconn.do_thing","parameters":{"name":"hello"}},"context":{"description":"test"}}`
	r := signedJSONRequest(t, http.MethodPost, "/approvals/request", reqBody, privKey, agentID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if errResp.Error.Code != ErrInvalidParameters {
		t.Errorf("expected error code %q, got %q", ErrInvalidParameters, errResp.Error.Code)
	}

	// Check that missing_fields contains "target".
	details := errResp.Error.Details
	if details == nil {
		t.Fatal("expected error details to be present")
	}
	missingRaw, ok := details["missing_fields"]
	if !ok {
		t.Fatal("expected missing_fields in details")
	}
	missingSlice, ok := missingRaw.([]any)
	if !ok {
		t.Fatalf("expected missing_fields to be a slice, got %T", missingRaw)
	}
	found := false
	for _, f := range missingSlice {
		if f == "target" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'target' in missing_fields, got %v", missingSlice)
	}

	// Check hint.
	hint, ok := details["hint"]
	if !ok {
		t.Fatal("expected hint in details")
	}
	hintStr, ok := hint.(string)
	if !ok || hintStr == "" {
		t.Errorf("expected non-empty hint string, got %v", hint)
	}
}

func TestAgentRequestApproval_ValidParameters(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	pubKeySSH, privKey, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	agentID := testhelper.InsertAgentWithPublicKey(t, tx, uid, "registered", pubKeySSH)

	// Create a connector with an action that has required fields.
	testhelper.InsertConnector(t, tx, "testconn")
	testhelper.InsertConnectorActionFull(t, tx, "testconn", "testconn.do_thing", "Do Thing", testhelper.ConnectorActionOpts{
		ParametersSchema: []byte(`{
			"type": "object",
			"required": ["name"],
			"properties": {
				"name": {"type": "string"}
			}
		}`),
	})

	deps := testDepsForDB(t, tx)
	router := NewRouter(deps)

	// Submit with all required fields present.
	reqBody := `{"request_id":"req_valid_params","action":{"type":"testconn.do_thing","parameters":{"name":"hello"}},"context":{"description":"test"}}`
	r := signedJSONRequest(t, http.MethodPost, "/approvals/request", reqBody, privKey, agentID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAgentRequestApproval_UnknownActionNoSchema(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	pubKeySSH, privKey, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	agentID := testhelper.InsertAgentWithPublicKey(t, tx, uid, "registered", pubKeySSH)

	deps := testDepsForDB(t, tx)
	router := NewRouter(deps)

	// Submit with an action type that has no schema in the DB — should pass through.
	reqBody := `{"request_id":"req_unknown_action","action":{"type":"unknown.action","parameters":{"anything":"goes"}},"context":{"description":"test"}}`
	r := signedJSONRequest(t, http.MethodPost, "/approvals/request", reqBody, privKey, agentID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 (fail-open), got %d: %s", w.Code, w.Body.String())
	}
}

func TestAgentRequestApproval_NoParametersWithRequiredFields(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	pubKeySSH, privKey, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	agentID := testhelper.InsertAgentWithPublicKey(t, tx, uid, "registered", pubKeySSH)

	// Create a connector with an action that has required fields.
	testhelper.InsertConnector(t, tx, "testconn")
	testhelper.InsertConnectorActionFull(t, tx, "testconn", "testconn.do_thing", "Do Thing", testhelper.ConnectorActionOpts{
		ParametersSchema: []byte(`{
			"type": "object",
			"required": ["name"],
			"properties": {
				"name": {"type": "string"}
			}
		}`),
	})

	deps := testDepsForDB(t, tx)
	router := NewRouter(deps)

	// Submit with no parameters at all — should fail with missing required fields.
	reqBody := `{"request_id":"req_no_params","action":{"type":"testconn.do_thing"},"context":{"description":"test"}}`
	r := signedJSONRequest(t, http.MethodPost, "/approvals/request", reqBody, privKey, agentID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if errResp.Error.Code != ErrInvalidParameters {
		t.Errorf("expected error code %q, got %q", ErrInvalidParameters, errResp.Error.Code)
	}
}

func TestAgentRequestApproval_NullParameterValue(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	pubKeySSH, privKey, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	agentID := testhelper.InsertAgentWithPublicKey(t, tx, uid, "registered", pubKeySSH)

	testhelper.InsertConnector(t, tx, "testconn")
	testhelper.InsertConnectorActionFull(t, tx, "testconn", "testconn.do_thing", "Do Thing", testhelper.ConnectorActionOpts{
		ParametersSchema: []byte(`{
			"type": "object",
			"required": ["name"],
			"properties": {
				"name": {"type": "string"}
			}
		}`),
	})

	deps := testDepsForDB(t, tx)
	router := NewRouter(deps)

	// Submit with required field set to null — should fail.
	reqBody := `{"request_id":"req_null_param","action":{"type":"testconn.do_thing","parameters":{"name":null}},"context":{"description":"test"}}`
	r := signedJSONRequest(t, http.MethodPost, "/approvals/request", reqBody, privKey, agentID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}
