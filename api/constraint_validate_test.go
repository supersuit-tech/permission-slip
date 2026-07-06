package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
	"github.com/supersuit-tech/permission-slip/connectors/protonmail"
	"github.com/supersuit-tech/permission-slip/db"
	"github.com/supersuit-tech/permission-slip/db/testhelper"
)

type mockMetadataConnector struct {
	mockConnector
	metadata map[string]any
	metaErr  error
}

func (c *mockMetadataConnector) ResolveConstraintMetadata(_ context.Context, actionType string, _ json.RawMessage, _ connectors.Credentials) (map[string]any, error) {
	if c.metaErr != nil {
		return nil, c.metaErr
	}
	return c.metadata, nil
}

func setupProtonArchiveStandingApprovalTest(t *testing.T, constraints []byte, metadata map[string]any) (db.DBTX, http.Handler, int64, []byte, string) {
	t.Helper()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	pubKeySSH, privKey, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	agentID := testhelper.InsertAgentWithPublicKey(t, tx, uid, "registered", pubKeySSH)

	testhelper.InsertConnector(t, tx, "protonmail")
	testhelper.InsertConnectorAction(t, tx, "protonmail", "protonmail.archive_email", "Archive Email")

	saID := testhelper.GenerateID(t, "sa_")
	testhelper.InsertStandingApprovalFull(t, tx, saID, agentID, uid, testhelper.StandingApprovalOpts{
		ActionType:  "protonmail.archive_email",
		Constraints: constraints,
	})

	action := &mockAction{result: &connectors.ActionResult{Data: json.RawMessage(`{"status":"archived"}`)}}
	metaConn := &mockMetadataConnector{
		mockConnector: mockConnector{
			id:      "protonmail",
			actions: map[string]connectors.Action{"protonmail.archive_email": action},
		},
		metadata: metadata,
	}
	registry := connectors.NewRegistry()
	registry.Register(metaConn)

	deps := &Deps{DB: tx, Connectors: registry, JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	return tx, router, agentID, privKey, saID
}

func TestRequestApproval_AutoApprove_MetaSenderMatch(t *testing.T) {
	t.Parallel()
	tx, router, agentID, privKey, saID := setupProtonArchiveStandingApprovalTest(t,
		[]byte(`{"message_id":"*","folder":"*","$meta":{"sender":{"$pattern":"alice@example.com"}}}`),
		map[string]any{"sender": "alice@example.com"},
	)

	reqBody := `{"request_id":"meta-sender-match-001","action":{"type":"protonmail.archive_email","parameters":{"message_id":42,"folder":"INBOX"}},"context":{"description":"archive"}}`
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
	if resp.Status != "approved" {
		t.Errorf("expected approved, got %q", resp.Status)
	}
	if resp.StandingApprovalID != saID {
		t.Errorf("expected standing approval %q, got %q", saID, resp.StandingApprovalID)
	}
	testhelper.RequireStandingApprovalExecutionCount(t, tx, saID, 1)
}

func TestRequestApproval_AutoApprove_MetaSenderMismatchFallsThrough(t *testing.T) {
	t.Parallel()
	_, router, agentID, privKey, _ := setupProtonArchiveStandingApprovalTest(t,
		[]byte(`{"message_id":"*","folder":"*","$meta":{"sender":{"$pattern":"alice@example.com"}}}`),
		map[string]any{"sender": "bob@example.com"},
	)

	reqBody := `{"request_id":"meta-sender-mismatch-001","action":{"type":"protonmail.archive_email","parameters":{"message_id":42,"folder":"INBOX"}},"context":{"description":"archive"}}`
	r := signedJSONRequest(t, http.MethodPost, "/approvals/request", reqBody, privKey, agentID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 pending, got %d: %s", w.Code, w.Body.String())
	}
	var resp agentRequestApprovalResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Status != "pending" {
		t.Errorf("expected pending fallthrough, got %q", resp.Status)
	}
}

func TestRequestApproval_AutoApprove_SpoofedSenderParamIgnored(t *testing.T) {
	t.Parallel()
	tx, router, agentID, privKey, saID := setupProtonArchiveStandingApprovalTest(t,
		[]byte(`{"message_id":"*","folder":"*","$meta":{"sender":{"$pattern":"alice@example.com"}}}`),
		map[string]any{"sender": "bob@example.com"},
	)

	reqBody := `{"request_id":"meta-sender-spoof-001","action":{"type":"protonmail.archive_email","parameters":{"message_id":42,"folder":"INBOX","sender":"alice@example.com"}},"context":{"description":"archive"}}`
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
	if resp.Status == "approved" {
		t.Fatal("spoofed sender param must not auto-approve when verified sender differs")
	}
	testhelper.RequireStandingApprovalExecutionCount(t, tx, saID, 0)
}

func TestAgentCreateStandingApprovalRequest_RejectsEmptyKeyMetaConstraint(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	pubKeySSH, privKey, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	agentID := testhelper.InsertAgentWithPublicKey(t, tx, uid, "registered", pubKeySSH)

	testhelper.InsertConnector(t, tx, "protonmail")
	schema := []byte(`{"type":"object","properties":{"message_id":{"type":"integer"},"folder":{"type":"string"}}}`)
	testhelper.InsertConnectorActionFull(t, tx, "protonmail", "protonmail.read_email", "Read Email", testhelper.ConnectorActionOpts{
		ParametersSchema: schema,
	})

	registry := connectors.NewRegistry()
	registry.Register(protonmail.New())

	deps := &Deps{DB: tx, Connectors: registry, JWTSigningSecret: testJWTSecret, BaseURL: "https://app.example.com"}
	router := NewRouter(deps)

	reqBody := `{"action_type":"protonmail.read_email","constraints":{"":{"from":"automated@airbnb.com"}}}`
	r := signedJSONRequest(t, http.MethodPost, "/standing-approvals/request", reqBody, privKey, agentID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `$meta`) {
		t.Fatalf("expected $meta hint in response, got: %s", w.Body.String())
	}
}

func TestRequestApproval_AutoApprove_ReadEmailMetaFromMismatch(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	pubKeySSH, privKey, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	agentID := testhelper.InsertAgentWithPublicKey(t, tx, uid, "registered", pubKeySSH)

	testhelper.InsertConnector(t, tx, "protonmail")
	testhelper.InsertConnectorAction(t, tx, "protonmail", "protonmail.read_email", "Read Email")

	saID := testhelper.GenerateID(t, "sa_")
	testhelper.InsertStandingApprovalFull(t, tx, saID, agentID, uid, testhelper.StandingApprovalOpts{
		ActionType:  "protonmail.read_email",
		Constraints: []byte(`{"message_id":"*","folder":"*","$meta":{"from":"automated@airbnb.com"}}`),
	})

	action := &mockAction{result: &connectors.ActionResult{Data: json.RawMessage(`{"uid":42}`)}}
	metaConn := &mockMetadataConnector{
		mockConnector: mockConnector{
			id:      "protonmail",
			actions: map[string]connectors.Action{"protonmail.read_email": action},
		},
		metadata: map[string]any{
			"messages": []map[string]any{{
				"from": "other@example.com",
				"to":   []string{"me@example.com"},
				"cc":   []string{},
				"bcc":  []string{},
			}},
			"senders": []string{"other@example.com"},
			"sender":  "other@example.com",
		},
	}
	registry := connectors.NewRegistry()
	registry.Register(metaConn)

	deps := &Deps{DB: tx, Connectors: registry, JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	reqBody := `{"request_id":"read-email-meta-mismatch-001","action":{"type":"protonmail.read_email","parameters":{"message_id":42,"folder":"INBOX"}},"context":{"description":"read"}}`
	r := signedJSONRequest(t, http.MethodPost, "/approvals/request", reqBody, privKey, agentID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 pending, got %d: %s", w.Code, w.Body.String())
	}
	var resp agentRequestApprovalResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Status != "pending" {
		t.Errorf("expected pending for non-matching sender, got %q", resp.Status)
	}
	testhelper.RequireStandingApprovalExecutionCount(t, tx, saID, 0)
}

func TestRequestApproval_AutoApprove_ReadEmailMetaFromMatch(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	pubKeySSH, privKey, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	agentID := testhelper.InsertAgentWithPublicKey(t, tx, uid, "registered", pubKeySSH)

	testhelper.InsertConnector(t, tx, "protonmail")
	testhelper.InsertConnectorAction(t, tx, "protonmail", "protonmail.read_email", "Read Email")

	saID := testhelper.GenerateID(t, "sa_")
	testhelper.InsertStandingApprovalFull(t, tx, saID, agentID, uid, testhelper.StandingApprovalOpts{
		ActionType:  "protonmail.read_email",
		Constraints: []byte(`{"message_id":"*","folder":"*","$meta":{"from":{"$pattern":"auto-confirm@amazon.com"}}}`),
	})

	action := &mockAction{result: &connectors.ActionResult{Data: json.RawMessage(`{"uid":42}`)}}
	metaConn := &mockMetadataConnector{
		mockConnector: mockConnector{
			id:      "protonmail",
			actions: map[string]connectors.Action{"protonmail.read_email": action},
		},
		metadata: map[string]any{
			"messages": []map[string]any{{
				"from": "auto-confirm@amazon.com",
				"to":   []string{"me@example.com"},
				"cc":   []string{},
				"bcc":  []string{},
			}},
			"senders": []string{"auto-confirm@amazon.com"},
			"sender":  "auto-confirm@amazon.com",
		},
	}
	registry := connectors.NewRegistry()
	registry.Register(metaConn)

	deps := &Deps{DB: tx, Connectors: registry, JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	reqBody := `{"request_id":"read-email-meta-match-001","action":{"type":"protonmail.read_email","parameters":{"message_id":42,"folder":"INBOX"}},"context":{"description":"read"}}`
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
	if resp.Status != "approved" {
		t.Errorf("expected approved, got %q", resp.Status)
	}
	if resp.StandingApprovalID != saID {
		t.Errorf("expected standing approval %q, got %q", saID, resp.StandingApprovalID)
	}
}

func TestRequestApproval_Fallthrough_SurfacesMetadataUnavailableInContext(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	pubKeySSH, privKey, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	agentID := testhelper.InsertAgentWithPublicKey(t, tx, uid, "registered", pubKeySSH)

	testhelper.InsertConnector(t, tx, "protonmail")
	testhelper.InsertConnectorAction(t, tx, "protonmail", "protonmail.read_email", "Read Email")

	saID := testhelper.GenerateID(t, "sa_")
	testhelper.InsertStandingApprovalFull(t, tx, saID, agentID, uid, testhelper.StandingApprovalOpts{
		ActionType:  "protonmail.read_email",
		Constraints: []byte(`{"message_id":"*","folder":"*","$meta":{"from":{"$pattern":"auto-confirm@amazon.com"}}}`),
	})

	action := &mockAction{result: &connectors.ActionResult{Data: json.RawMessage(`{"uid":42}`)}}
	metaConn := &mockMetadataConnector{
		mockConnector: mockConnector{
			id:      "protonmail",
			actions: map[string]connectors.Action{"protonmail.read_email": action},
		},
		metaErr: connectors.ErrConstraintMetadataUnavailable,
	}
	registry := connectors.NewRegistry()
	registry.Register(metaConn)

	deps := &Deps{DB: tx, Connectors: registry, JWTSigningSecret: testJWTSecret}
	router := NewRouter(deps)

	reqBody := `{"request_id":"read-email-meta-unresolved-001","action":{"type":"protonmail.read_email","parameters":{"message_id":42,"folder":"INBOX"}},"context":{"description":"read"}}`
	r := signedJSONRequest(t, http.MethodPost, "/approvals/request", reqBody, privKey, agentID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 pending, got %d: %s", w.Code, w.Body.String())
	}
	var resp agentRequestApprovalResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Status != "pending" {
		t.Fatalf("expected pending, got %q", resp.Status)
	}
	if resp.ApprovalID == "" {
		t.Fatal("expected approval_id on pending response")
	}

	approval, err := db.GetApprovalByID(t.Context(), tx, resp.ApprovalID)
	if err != nil {
		t.Fatalf("GetApprovalByID: %v", err)
	}
	if approval == nil {
		t.Fatal("approval not found")
	}

	var ctxObj map[string]json.RawMessage
	if err := json.Unmarshal(approval.Context, &ctxObj); err != nil {
		t.Fatalf("unmarshal context: %v", err)
	}
	var details map[string]any
	if err := json.Unmarshal(ctxObj["details"], &details); err != nil {
		t.Fatalf("unmarshal details: %v", err)
	}
	ft, ok := details["standing_approval_fallthrough"].(map[string]any)
	if !ok {
		t.Fatalf("expected standing_approval_fallthrough in context, got %#v", details)
	}
	if ft["reason"] != standingApprovalFallthroughMetadataUnavailable {
		t.Fatalf("reason = %#v", ft["reason"])
	}
}
