package api

import (
	"crypto/ed25519"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/db/testhelper"
)

func setupBulkTestAgent(t *testing.T) (agentID int64, privKey ed25519.PrivateKey, router http.Handler) {
	t.Helper()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	pubKeySSH, priv, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	agentID = testhelper.InsertAgentWithPublicKey(t, tx, uid, "registered", pubKeySSH)
	deps := &Deps{DB: tx, JWTSigningSecret: testJWTSecret, BaseURL: "http://localhost:8080"}
	return agentID, priv, NewRouter(deps)
}

func TestBulkRequestRejectsMixedActionTypes(t *testing.T) {
	t.Parallel()
	agentID, privKey, router := setupBulkTestAgent(t)

	reqBody := `{"items":[{"request_id":"req_m1","action":{"type":"email.send","parameters":{"to":"a@example.com"}},"context":{}},{"request_id":"req_m2","action":{"type":"calendar.create_event","parameters":{"title":"x"}},"context":{}}]}`
	r := signedJSONRequest(t, http.MethodPost, "/approvals/bulk-request", reqBody, privKey, agentID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestBulkRequestRejectsSingleItem(t *testing.T) {
	t.Parallel()
	agentID, privKey, router := setupBulkTestAgent(t)

	reqBody := `{"items":[{"request_id":"req_one","action":{"type":"email.send","parameters":{"to":"a@example.com"}},"context":{}}]}`
	r := signedJSONRequest(t, http.MethodPost, "/approvals/bulk-request", reqBody, privKey, agentID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestBulkRequestCreatesGroup(t *testing.T) {
	t.Parallel()
	agentID, privKey, router := setupBulkTestAgent(t)

	reqBody := `{"items":[{"request_id":"req_b1","action":{"type":"email.send","parameters":{"to":"a@example.com","subject":"Hi","body":"One"}},"context":{"description":"first"}},{"request_id":"req_b2","action":{"type":"email.send","parameters":{"to":"b@example.com","subject":"Hi","body":"Two"}},"context":{"description":"second"}}]}`
	r := signedJSONRequest(t, http.MethodPost, "/approvals/bulk-request", reqBody, privKey, agentID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp agentBulkRequestApprovalResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.BulkGroupID == "" {
		t.Fatal("expected bulk_group_id")
	}
	if resp.ItemCount != 2 {
		t.Fatalf("expected item_count 2, got %d", resp.ItemCount)
	}
	if len(resp.Results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(resp.Results))
	}

	statusReq := signedJSONRequest(t, http.MethodGet, "/approval-groups/"+resp.BulkGroupID+"/status", "", privKey, agentID)
	statusW := httptest.NewRecorder()
	router.ServeHTTP(statusW, statusReq)
	if statusW.Code != http.StatusOK {
		t.Fatalf("group status: expected 200, got %d: %s", statusW.Code, statusW.Body.String())
	}
}
