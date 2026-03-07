package api

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/db"
	"github.com/supersuit-tech/permission-slip-web/db/testhelper"
)

// ── POST /invite/{invite_code} ──────────────────────────────────────────────

func TestInviteRegister_Success(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	inviteCode := "PS-TEST-1234"
	codeHash := hashCodeHex(inviteCode, "")
	riID := testhelper.GenerateID(t, "ri_")
	if _, err := db.CreateRegistrationInvite(context.Background(), tx, riID, uid, codeHash, 900); err != nil {
		t.Fatalf("create invite: %v", err)
	}

	pubKeySSH, privKey, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	handler := InviteHandler(&Deps{DB: tx})

	body := inviteRequestBody(t, "req-1", pubKeySSH)
	r := signedJSONRequest(t, http.MethodPost, "/invite/"+inviteCode, body, privKey, 0)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp registerAgentResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.AgentID == 0 {
		t.Error("expected non-zero agent_id")
	}
	if !resp.VerificationRequired {
		t.Error("expected verification_required=true")
	}
	if resp.ExpiresAt == nil {
		t.Error("expected expires_at to be set")
	}
	if resp.Approver == nil {
		t.Fatal("expected approver to be set")
	}
	expectedUsername := "u_" + uid[:8]
	if resp.Approver.Username != expectedUsername {
		t.Errorf("expected approver.username %q, got %q", expectedUsername, resp.Approver.Username)
	}
}

func TestInviteRegister_InvalidSignature(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	inviteCode := "PS-TEST-SIG1"
	codeHash := hashCodeHex(inviteCode, "")
	riID := testhelper.GenerateID(t, "ri_")
	if _, err := db.CreateRegistrationInvite(context.Background(), tx, riID, uid, codeHash, 900); err != nil {
		t.Fatalf("create invite: %v", err)
	}

	// Sign with a different key than the one in the body.
	pubKeySSH1, _, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key 1: %v", err)
	}
	_, privKey2, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key 2: %v", err)
	}

	handler := InviteHandler(&Deps{DB: tx})

	body := inviteRequestBody(t, "req-sig", pubKeySSH1)
	r := signedJSONRequest(t, http.MethodPost, "/invite/"+inviteCode, body, privKey2, 0)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestInviteRegister_NotFound(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	pubKeySSH, privKey, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	handler := InviteHandler(&Deps{DB: tx})

	body := inviteRequestBody(t, "req-nf", pubKeySSH)
	r := signedJSONRequest(t, http.MethodPost, "/invite/PS-NONEXISTENT", body, privKey, 0)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestInviteRegister_AlreadyConsumed(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	inviteCode := "PS-CONSUMED1"
	codeHash := hashCodeHex(inviteCode, "")
	riID := testhelper.GenerateID(t, "ri_")
	if _, err := db.CreateRegistrationInvite(context.Background(), tx, riID, uid, codeHash, 900); err != nil {
		t.Fatalf("create invite: %v", err)
	}
	if _, err := db.ConsumeInvite(context.Background(), tx, codeHash); err != nil {
		t.Fatalf("consume invite: %v", err)
	}

	pubKeySSH, privKey, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	handler := InviteHandler(&Deps{DB: tx})

	body := inviteRequestBody(t, "req-dup", pubKeySSH)
	r := signedJSONRequest(t, http.MethodPost, "/invite/"+inviteCode, body, privKey, 0)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", w.Code, w.Body.String())
	}
}

func TestInviteRegister_MissingPublicKey(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	inviteCode := "PS-NOPK-1234"
	codeHash := hashCodeHex(inviteCode, "")
	riID := testhelper.GenerateID(t, "ri_")
	if _, err := db.CreateRegistrationInvite(context.Background(), tx, riID, uid, codeHash, 900); err != nil {
		t.Fatalf("create invite: %v", err)
	}

	_, privKey, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	handler := InviteHandler(&Deps{DB: tx})

	r := signedJSONRequest(t, http.MethodPost, "/invite/"+inviteCode, `{"request_id":"req-nopk"}`, privKey, 0)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

// ── POST /agents/{agent_id}/verify ──────────────────────────────────────────

// inviteRegistration holds the result of a successful invite registration,
// providing everything needed to test the verify endpoint.
type inviteRegistration struct {
	AgentID     int64
	ConfirmCode string
	PrivKey     ed25519.PrivateKey
}

// registerViaInvite performs the full invite registration flow and returns
// the agent ID, confirmation code, and private key for subsequent signing.
func registerViaInvite(t *testing.T, tx db.DBTX, uid string) inviteRegistration {
	t.Helper()

	inviteCode := testhelper.GenerateID(t, "PS-")
	codeHash := hashCodeHex(inviteCode, "")
	riID := testhelper.GenerateID(t, "ri_")
	if _, err := db.CreateRegistrationInvite(context.Background(), tx, riID, uid, codeHash, 900); err != nil {
		t.Fatalf("create invite: %v", err)
	}

	pubKeySSH, privKey, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	handler := InviteHandler(&Deps{DB: tx})

	body := inviteRequestBody(t, "req-"+testhelper.GenerateID(t, ""), pubKeySSH)
	r := signedJSONRequest(t, http.MethodPost, "/invite/"+inviteCode, body, privKey, 0)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("register via invite: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp registerAgentResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal register response: %v", err)
	}

	agent, err := db.GetAgentByIDUnscoped(context.Background(), tx, resp.AgentID)
	if err != nil {
		t.Fatalf("get agent: %v", err)
	}
	if agent.ConfirmationCode == nil {
		t.Fatal("expected confirmation code to be set")
	}

	return inviteRegistration{
		AgentID:     resp.AgentID,
		ConfirmCode: *agent.ConfirmationCode,
		PrivKey:     privKey,
	}
}

// signedVerifyRequest builds a signed POST /agents/{agent_id}/verify request.
// Use this to avoid duplicating the body→sign→request pattern across tests.
func signedVerifyRequest(t *testing.T, reg inviteRegistration, confirmCode, requestID string) (*http.Request, []byte) {
	t.Helper()
	body := verifyRequestBody(requestID, confirmCode)
	path := fmt.Sprintf("/agents/%d/verify", reg.AgentID)
	r := signedJSONRequest(t, http.MethodPost, path, body, reg.PrivKey, reg.AgentID)
	return r, []byte(body)
}

// submitWrongVerifyCodes submits n wrong verification codes and asserts each
// returns 401 Unauthorized. Use this as setup for lockout tests.
func submitWrongVerifyCodes(t *testing.T, router http.Handler, reg inviteRegistration, n int) {
	t.Helper()
	for i := range n {
		r, _ := signedVerifyRequest(t, reg, "ZZZ-ZZZ", fmt.Sprintf("wrong-%d", i))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("wrong code attempt %d: expected 401, got %d: %s", i, w.Code, w.Body.String())
		}
	}
}

func TestVerifyRegistration_Success(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	reg := registerViaInvite(t, tx, uid)

	router := NewRouter(&Deps{DB: tx, SupabaseJWTSecret: testJWTSecret})

	r, _ := signedVerifyRequest(t, reg, reg.ConfirmCode, "verify-1")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp verifyRegistrationResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Status != "registered" {
		t.Errorf("expected status 'registered', got %q", resp.Status)
	}
	if resp.RegisteredAt == nil {
		t.Error("expected registered_at to be set")
	}
	if resp.Approver == nil {
		t.Fatal("expected approver to be set in verify response")
	}
	expectedUsername := "u_" + uid[:8]
	if resp.Approver.Username != expectedUsername {
		t.Errorf("expected approver.username %q, got %q", expectedUsername, resp.Approver.Username)
	}
}

func TestVerifyRegistration_WrongCode(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	reg := registerViaInvite(t, tx, uid)

	router := NewRouter(&Deps{DB: tx, SupabaseJWTSecret: testJWTSecret})

	r, _ := signedVerifyRequest(t, reg, "AAA-BBB", "verify-wrong")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("unmarshal error response: %v", err)
	}
	if errResp.Error.Code != ErrInvalidCode {
		t.Errorf("expected error code %q, got %q", ErrInvalidCode, errResp.Error.Code)
	}
}

func TestVerifyRegistration_Lockout(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	reg := registerViaInvite(t, tx, uid)

	router := NewRouter(&Deps{DB: tx, SupabaseJWTSecret: testJWTSecret})

	// Submit 5 wrong codes to trigger lockout.
	submitWrongVerifyCodes(t, router, reg, 5)

	// 6th attempt (wrong code) should be locked out (410 Gone).
	r, _ := signedVerifyRequest(t, reg, "ZZZ-ZZZ", "lock-final")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusGone {
		t.Fatalf("expected 410, got %d: %s", w.Code, w.Body.String())
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("unmarshal error response: %v", err)
	}
	if errResp.Error.Code != ErrVerificationLocked {
		t.Errorf("expected error code %q, got %q", ErrVerificationLocked, errResp.Error.Code)
	}

	// The correct code must also be rejected after lockout.
	rCorrect, _ := signedVerifyRequest(t, reg, reg.ConfirmCode, "lock-correct")
	wCorrect := httptest.NewRecorder()
	router.ServeHTTP(wCorrect, rCorrect)

	if wCorrect.Code != http.StatusGone {
		t.Fatalf("correct code after lockout: expected 410, got %d: %s", wCorrect.Code, wCorrect.Body.String())
	}

	var correctResp ErrorResponse
	if err := json.Unmarshal(wCorrect.Body.Bytes(), &correctResp); err != nil {
		t.Fatalf("unmarshal correct-after-lockout response: %v", err)
	}
	if correctResp.Error.Code != ErrVerificationLocked {
		t.Errorf("correct code after lockout: expected error code %q, got %q",
			ErrVerificationLocked, correctResp.Error.Code)
	}
}

func TestVerifyRegistration_EmitsAuditEvent(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	reg := registerViaInvite(t, tx, uid)

	router := NewRouter(&Deps{DB: tx, SupabaseJWTSecret: testJWTSecret})

	r, _ := signedVerifyRequest(t, reg, reg.ConfirmCode, "verify-audit")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Two audit events: one for pending registration (invite), one for completed verification.
	testhelper.RequireAuditEventCount(t, tx, uid, "agent.registered", 2)
}
