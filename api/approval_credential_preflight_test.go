package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
	"github.com/supersuit-tech/permission-slip/db"
	"github.com/supersuit-tech/permission-slip/db/testhelper"
	"github.com/supersuit-tech/permission-slip/oauth"
	"github.com/supersuit-tech/permission-slip/vault"
)

func TestAgentRequestApproval_PreflightRejectsNeedsReauth(t *testing.T) {
	t.Parallel()

	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	connID := "testgoogle"
	actionType := connID + ".send_email"
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertConnectorAction(t, tx, connID, actionType, "Send")
	testhelper.InsertConnectorRequiredCredentialOAuth(t, tx, connID, "google", "google", nil)

	pubKeySSH, privKey, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	agentID := testhelper.InsertAgentWithPublicKey(t, tx, uid, "registered", pubKeySSH)
	testhelper.InsertAgentConnector(t, tx, agentID, uid, connID)

	connOAuthID := testhelper.GenerateID(t, "oconn_")
	testhelper.InsertOAuthConnectionFull(t, tx, connOAuthID, uid, "google", testhelper.OAuthConnectionOpts{
		Status: "needs_reauth",
		Scopes: []string{},
	})
	conn, err := db.GetOAuthConnectionByProvider(t.Context(), tx, uid, "google")
	if err != nil {
		t.Fatalf("get oauth connection: %v", err)
	}
	bindingID := testhelper.GenerateID(t, "accr_")
	if _, err := db.UpsertAgentConnectorCredential(t.Context(), tx, db.UpsertAgentConnectorCredentialParams{
		ID: bindingID, AgentID: agentID, ConnectorID: connID,
		ApproverID: uid, OAuthConnectionID: &conn.ID,
	}); err != nil {
		t.Fatalf("bind credential: %v", err)
	}

	reg := connectors.NewRegistry()
	reg.Register(newTestStubConnector(connID, actionType))
	oauthReg := oauth.NewRegistry()
	_ = oauthReg.Register(oauth.Provider{
		ID: "google", TokenURL: "https://oauth2.googleapis.com/token",
		ClientID: "id", ClientSecret: "secret", Source: oauth.SourceBuiltIn,
	})

	deps := &Deps{
		DB:               tx,
		JWTSigningSecret: testJWTSecret,
		Vault:            vault.NewMockVaultStore(),
		Connectors:       reg,
		OAuthProviders:   oauthReg,
		BaseURL:          "http://test",
	}
	router := NewRouter(deps)

	reqBody := `{"request_id":"req_oauth_preflight","action":{"type":"testgoogle.send_email","parameters":{}},"context":{"description":"test"}}`
	r := signedJSONRequest(t, http.MethodPost, "/approvals/request", reqBody, privKey, agentID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if errResp.Error.Code != ErrOAuthRefreshFailed {
		t.Errorf("expected error code %q, got %q", ErrOAuthRefreshFailed, errResp.Error.Code)
	}
}

func TestApproveApproval_PreflightBlocksBeforeCommit(t *testing.T) {
	t.Parallel()

	f := setupOAuthExecutionTest(t, oauthExecOpts{
		Connection: &testhelper.OAuthConnectionOpts{
			Status: "needs_reauth",
			Scopes: []string{},
		},
	})

	apprID := testhelper.GenerateID(t, "appr_")
	actionJSON, _ := json.Marshal(map[string]any{
		"type":       "testgoogle.send_email",
		"parameters": map[string]any{},
	})
	testhelper.InsertApproval(t, f.TX, apprID, f.AgentID, f.UserID)
	if _, err := f.TX.Exec(t.Context(),
		`UPDATE approvals SET action = $1 WHERE approval_id = $2`, string(actionJSON), apprID); err != nil {
		t.Fatalf("update approval action: %v", err)
	}

	deps := f.Deps
	deps.JWTSigningSecret = testJWTSecret
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodPost, "/approvals/"+apprID+"/approve", f.UserID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}

	stored, err := db.GetApprovalByIDAndApprover(t.Context(), f.TX, apprID, f.UserID)
	if err != nil {
		t.Fatalf("get approval: %v", err)
	}
	if stored == nil || stored.Status != "pending" {
		t.Fatalf("expected approval to remain pending, got %+v", stored)
	}
}
