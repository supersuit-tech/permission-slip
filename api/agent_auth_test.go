package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/db/testhelper"
)

// ── RequireAgentSignature middleware tests ────────────────────────────────────

func TestRequireAgentSignature_Valid(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	pubKeySSH, privKey, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	agentID := testhelper.InsertAgentWithPublicKey(t, tx, uid, "registered", pubKeySSH)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	handler := RequireAgentSignature(deps)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		agent := AuthenticatedAgent(r.Context())
		if agent == nil {
			t.Error("expected agent in context")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if agent.AgentID != agentID {
			t.Errorf("expected agent_id %d, got %d", agentID, agent.AgentID)
		}
		sig := AuthenticatedSignature(r.Context())
		if sig == nil {
			t.Error("expected signature in context")
		}
		RespondJSON(w, http.StatusOK, map[string]int64{"agent_id": agent.AgentID})
	}))

	r := signedJSONRequest(t, http.MethodGet, "/agents/me", "", privKey, agentID)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRequireAgentSignature_MissingSigHeader(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	handler := RequireAgentSignature(deps)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	r := httptest.NewRequest(http.MethodGet, "/agents/me", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
	var errResp ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if errResp.Error.Code != ErrInvalidSignature {
		t.Errorf("expected code %q, got %q", ErrInvalidSignature, errResp.Error.Code)
	}
}

func TestRequireAgentSignature_AgentNotFound(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	handler := RequireAgentSignature(deps)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	_, privKey, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	// Use a non-existent agent_id.
	r := signedJSONRequest(t, http.MethodGet, "/agents/me", "", privKey, 999999)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
	var errResp ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if errResp.Error.Code != ErrAgentNotFound {
		t.Errorf("expected code %q, got %q", ErrAgentNotFound, errResp.Error.Code)
	}
}

func TestRequireAgentSignature_WrongKey(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	pubKeySSH, _, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	agentID := testhelper.InsertAgentWithPublicKey(t, tx, uid, "registered", pubKeySSH)

	// Sign with a different key.
	_, wrongPrivKey, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	handler := RequireAgentSignature(deps)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	r := signedJSONRequest(t, http.MethodGet, "/agents/me", "", wrongPrivKey, agentID)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
	var errResp ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if errResp.Error.Code != ErrInvalidSignature {
		t.Errorf("expected code %q, got %q", ErrInvalidSignature, errResp.Error.Code)
	}
}

func TestRequireAgentSignature_PendingAgentRejected(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	pubKeySSH, privKey, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	agentID := testhelper.InsertAgentWithPublicKey(t, tx, uid, "pending", pubKeySSH)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	handler := RequireAgentSignature(deps)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	r := signedJSONRequest(t, http.MethodGet, "/agents/me", "", privKey, agentID)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
	var errResp ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if errResp.Error.Code != ErrAgentNotAuthorized {
		t.Errorf("expected code %q, got %q", ErrAgentNotAuthorized, errResp.Error.Code)
	}
}

func TestRequireAgentSignature_DeactivatedAgentRejected(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	pubKeySSH, privKey, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	agentID := testhelper.InsertAgentWithPublicKey(t, tx, uid, "deactivated", pubKeySSH)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	handler := RequireAgentSignature(deps)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	r := signedJSONRequest(t, http.MethodGet, "/agents/me", "", privKey, agentID)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRequireAgentSignature_ExpiredTimestamp(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	pubKeySSH, privKey, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	agentID := testhelper.InsertAgentWithPublicKey(t, tx, uid, "registered", pubKeySSH)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	handler := RequireAgentSignature(deps)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	// Sign with a timestamp 10 minutes in the past (outside the 5-minute window).
	r := httptest.NewRequest(http.MethodGet, "/agents/me", nil)
	SignRequestAt(privKey, agentID, r, nil, 1000000000) // very old timestamp
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
	var errResp ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if errResp.Error.Code != ErrTimestampExpired {
		t.Errorf("expected code %q, got %q", ErrTimestampExpired, errResp.Error.Code)
	}
}

// ── GET /agents/me integration tests ─────────────────────────────────────────

func TestGetAgentMe_Success(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	pubKeySSH, privKey, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	agentID := testhelper.InsertAgentWithPublicKey(t, tx, uid, "registered", pubKeySSH)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := signedJSONRequest(t, http.MethodGet, "/agents/me", "", privKey, agentID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp agentSelfResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.AgentID != agentID {
		t.Errorf("expected agent_id %d, got %d", agentID, resp.AgentID)
	}
	if resp.Status != "registered" {
		t.Errorf("expected status registered, got %q", resp.Status)
	}
	if resp.RegisteredAt == nil {
		t.Error("expected registered_at to be set")
	}
}

func TestGetAgentMe_WithMetadata(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	pubKeySSH, privKey, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	agentID := testhelper.InsertAgentWithPublicKey(t, tx, uid, "registered", pubKeySSH)

	// Set metadata on the agent.
	testhelper.MustExec(t, tx,
		`UPDATE agents SET metadata = '{"name":"test-agent","version":"1.0"}' WHERE agent_id = $1`, agentID)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := signedJSONRequest(t, http.MethodGet, "/agents/me", "", privKey, agentID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(w.Body.Bytes(), &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	metaJSON, ok := raw["metadata"]
	if !ok {
		t.Fatal("expected metadata in response")
	}
	var meta map[string]string
	if err := json.Unmarshal(metaJSON, &meta); err != nil {
		t.Fatalf("unmarshal metadata: %v", err)
	}
	if meta["name"] != "test-agent" {
		t.Errorf("expected name test-agent, got %q", meta["name"])
	}
}

func TestGetAgentMe_DoesNotExposeApproverID(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	pubKeySSH, privKey, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	agentID := testhelper.InsertAgentWithPublicKey(t, tx, uid, "registered", pubKeySSH)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := signedJSONRequest(t, http.MethodGet, "/agents/me", "", privKey, agentID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(w.Body.Bytes(), &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// These fields should not be in the response.
	for _, field := range []string{"approver_id", "public_key", "confirmation_code", "request_count_30d"} {
		if _, ok := raw[field]; ok {
			t.Errorf("response should not contain %q", field)
		}
	}
}

// ── Improved error message tests ─────────────────────────────────────────────

func TestRequireSession_AgentSigHeaderGivesBetterError(t *testing.T) {
	t.Parallel()
	deps := &Deps{SupabaseJWTSecret: testJWTSecret}
	handler := RequireSession(deps)(sessionTestHandler())

	_, privKey, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	// Send a request with X-Permission-Slip-Signature but no Authorization header.
	r := httptest.NewRequest(http.MethodGet, "/agents", nil)
	SignRequest(privKey, 42, r, nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if errResp.Error.Code != ErrInvalidToken {
		t.Errorf("expected code %q, got %q", ErrInvalidToken, errResp.Error.Code)
	}
	// The message should mention agent API endpoints.
	if errResp.Error.Message == "Missing Authorization header" {
		t.Error("expected improved error message, got generic 'Missing Authorization header'")
	}
}

func TestRequireSession_NoSigHeaderKeepsOriginalError(t *testing.T) {
	t.Parallel()
	deps := &Deps{SupabaseJWTSecret: testJWTSecret}
	handler := RequireSession(deps)(sessionTestHandler())

	// Regular request with no auth at all.
	r := httptest.NewRequest(http.MethodGet, "/agents", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if errResp.Error.Message != "Missing Authorization header" {
		t.Errorf("expected 'Missing Authorization header', got %q", errResp.Error.Message)
	}
}

// ── Context accessor tests ───────────────────────────────────────────────────

func TestAuthenticatedAgent_NoContext(t *testing.T) {
	t.Parallel()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	if agent := AuthenticatedAgent(r.Context()); agent != nil {
		t.Error("expected nil agent without middleware")
	}
}

func TestAuthenticatedSignature_NoContext(t *testing.T) {
	t.Parallel()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	if sig := AuthenticatedSignature(r.Context()); sig != nil {
		t.Error("expected nil signature without middleware")
	}
}

func TestRequireAgentSignature_NilDB(t *testing.T) {
	t.Parallel()
	deps := &Deps{DB: nil, SupabaseJWTSecret: testJWTSecret}
	handler := RequireAgentSignature(deps)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	_, privKey, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	r := signedJSONRequest(t, http.MethodGet, "/agents/me", "", privKey, 1)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d: %s", w.Code, w.Body.String())
	}
}
