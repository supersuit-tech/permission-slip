package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
	"github.com/supersuit-tech/permission-slip/db"
	"github.com/supersuit-tech/permission-slip/db/testhelper"
	"github.com/supersuit-tech/permission-slip/vault"
)

// End-to-end: two connector instances with distinct static credentials; approval execution uses the instance from the frozen action JSON.
func TestApprovalFlow_MultiInstance_UsesCorrectCredentialOnApprove(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	pubKeySSH, privKey, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	agentID := testhelper.InsertAgentWithPublicKey(t, tx, uid, "registered", pubKeySSH)

	connID := "flow_mi"
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertConnectorAction(t, tx, connID, connID+".ping", "Ping")
	testhelper.InsertConnectorRequiredCredential(t, tx, connID, connID, "api_key")

	testhelper.InsertAgentConnector(t, tx, agentID, uid, connID)
	ctx := context.Background()
	instSales, err := db.CreateAgentConnectorInstance(ctx, tx, db.CreateAgentConnectorInstanceParams{
		AgentID: agentID, ApproverID: uid, ConnectorID: connID,
	})
	if err != nil {
		t.Fatalf("create instance: %v", err)
	}

	v := vault.NewMockVaultStore()
	credJSON1, _ := json.Marshal(map[string]string{"api_key": "token-eng"})
	v1, err := v.CreateSecret(t.Context(), tx, "c1", credJSON1)
	if err != nil {
		t.Fatalf("vault: %v", err)
	}
	credID1 := testhelper.GenerateID(t, "cred_")
	testhelper.InsertCredentialWithVaultSecretIDAndLabel(t, tx, credID1, uid, connID, "default", v1)

	credJSON2, _ := json.Marshal(map[string]string{"api_key": "token-sales"})
	v2, err := v.CreateSecret(t.Context(), tx, "c2", credJSON2)
	if err != nil {
		t.Fatalf("vault: %v", err)
	}
	credID2 := testhelper.GenerateID(t, "cred_")
	testhelper.InsertCredentialWithVaultSecretIDAndLabel(t, tx, credID2, uid, connID, "sales", v2)

	defInst, err := db.GetDefaultAgentConnectorInstance(ctx, tx, agentID, uid, connID)
	if err != nil || defInst == nil {
		t.Fatalf("default instance: %v", defInst)
	}
	_, err = db.UpsertAgentConnectorCredentialByInstance(ctx, tx, db.UpsertAgentConnectorCredentialByInstanceParams{
		ID: testhelper.GenerateID(t, "accr_"), AgentID: agentID, ConnectorID: connID,
		ConnectorInstanceID: defInst.ConnectorInstanceID, ApproverID: uid, CredentialID: &credID1,
	})
	if err != nil {
		t.Fatalf("bind default: %v", err)
	}

	_, err = db.UpsertAgentConnectorCredentialByInstance(ctx, tx, db.UpsertAgentConnectorCredentialByInstanceParams{
		ID: testhelper.GenerateID(t, "accr_"), AgentID: agentID, ConnectorID: connID,
		ConnectorInstanceID: instSales.ConnectorInstanceID, ApproverID: uid, CredentialID: &credID2,
	})
	if err != nil {
		t.Fatalf("bind sales: %v", err)
	}

	var captured connectors.Credentials
	reg := connectors.NewRegistry()
	reg.Register(&credCapturingConnector{
		id:      connID,
		actions: []string{connID + ".ping"},
		onExec:  func(c connectors.Credentials) { captured = c },
	})

	deps := &Deps{DB: tx, Vault: v, Connectors: reg, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	reqBody := `{"request_id":"req_flow_mi","action":{"type":"` + connID + `.ping","parameters":{"connector_instance":"sales"}},"context":{"description":"x"}}`
	r := signedJSONRequest(t, http.MethodPost, "/approvals/request", reqBody, privKey, agentID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("request approval: %d %s", w.Code, w.Body.String())
	}
	var createResp agentRequestApprovalResponse
	if err := json.Unmarshal(w.Body.Bytes(), &createResp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	r2 := authenticatedRequest(t, http.MethodPost, "/approvals/"+createResp.ApprovalID+"/approve", uid)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, r2)
	if w2.Code != http.StatusOK {
		t.Fatalf("approve: %d %s", w2.Code, w2.Body.String())
	}

	tok, ok := captured.Get("api_key")
	if !ok {
		t.Fatal("expected api_key in executed credentials")
	}
	if tok != "token-sales" {
		t.Errorf("wrong OAuth/static credential used: got %q want token-sales", tok)
	}
}
