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

func TestRequestApproval_AutoApprove_RelativeDateInjectsSince(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	testhelper.InsertConnector(t, tx, "imessage")
	schema := []byte(`{"type":"object","properties":{"limit":{"type":"integer"},"since":{"type":"string","format":"date-time","x-ui":{"datetime_range_role":"lower"}},"before":{"type":"string","format":"date-time","x-ui":{"datetime_range_role":"upper"}}}}`)
	testhelper.InsertConnectorActionFull(t, tx, "imessage", "imessage.list_chats", "List Chats", testhelper.ConnectorActionOpts{
		ParametersSchema: schema,
		DataWindow:       []byte(`{"start_param":"since","end_param":"before"}`),
	})

	pubKeySSH, privKey, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	agentID := testhelper.InsertAgentWithPublicKey(t, tx, uid, "registered", pubKeySSH)

	saID := testhelper.GenerateID(t, "sa_")
	testhelper.InsertStandingApprovalFull(t, tx, saID, agentID, uid, testhelper.StandingApprovalOpts{
		ActionType:  "imessage.list_chats",
		Constraints: []byte(`{"limit":"*","since":"@today"}`),
	})

	router := NewRouter(testDepsForDB(t, tx))

	reqBody := `{"request_id":"rd-inject-001","action":{"type":"imessage.list_chats","version":"1","parameters":{"limit":20}},"context":{"description":"test"}}`
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
	sinceStr, ok := recorded["since"].(string)
	if !ok || sinceStr == "" {
		t.Fatalf("expected injected since param, got %v", recorded["since"])
	}
	sinceAt, err := time.Parse(time.RFC3339, sinceStr)
	if err != nil {
		t.Fatalf("parse since: %v", err)
	}
	startOfDay := time.Now().UTC().Truncate(24 * time.Hour)
	if sinceAt.Before(startOfDay) || sinceAt.After(startOfDay.Add(24*time.Hour)) {
		t.Fatalf("since %q is not start of today UTC", sinceStr)
	}
}

func TestValidateStandingApprovalConstraintKeys_RejectsRelativeDateOnNonTemporalField(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	testhelper.InsertConnector(t, tx, "imessage")
	schema := []byte(`{"type":"object","properties":{"query":{"type":"string"},"limit":{"type":"integer"}}}`)
	testhelper.InsertConnectorActionFull(t, tx, "imessage", "imessage.search", "Search", testhelper.ConnectorActionOpts{
		ParametersSchema: schema,
	})

	constraints := []byte(`{"query":"hello","limit":"@today"}`)
	err := validateStandingApprovalConstraintKeys(context.Background(), tx, nil, "imessage.search", constraints)
	if err == nil {
		t.Fatal("expected rejection for relative date on non-temporal field")
	}
}

func TestValidateStandingApprovalConstraintKeys_AllowsRelativeDateOnTemporalField(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	testhelper.InsertConnector(t, tx, "imessage")
	schema := []byte(`{"type":"object","properties":{"limit":{"type":"integer"},"since":{"type":"string","format":"date-time"}}}`)
	testhelper.InsertConnectorActionFull(t, tx, "imessage", "imessage.list_chats", "List Chats", testhelper.ConnectorActionOpts{
		ParametersSchema: schema,
	})

	constraints := []byte(`{"limit":"*","since":"@today"}`)
	if err := validateStandingApprovalConstraintKeys(context.Background(), tx, nil, "imessage.list_chats", constraints); err != nil {
		t.Fatalf("expected valid constraints, got: %v", err)
	}
}

func TestValidateStandingApprovalConstraintsForAction_RejectsInvalidRelativeDateToken(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	testhelper.InsertConnector(t, tx, "imessage")
	schema := []byte(`{"type":"object","properties":{"since":{"type":"string","format":"date-time"}}}`)
	testhelper.InsertConnectorActionFull(t, tx, "imessage", "imessage.list_chats", "List Chats", testhelper.ConnectorActionOpts{
		ParametersSchema: schema,
	})

	raw := json.RawMessage(`{"since":"@tomorrow","limit":"*"}`)
	_, err := validateStandingApprovalConstraintsForAction(context.Background(), tx, nil, "imessage.list_chats", raw)
	if err == nil {
		t.Fatal("expected invalid relative date token rejection")
	}
}
