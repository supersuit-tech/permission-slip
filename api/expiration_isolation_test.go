package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/supersuit-tech/permission-slip/db"
	"github.com/supersuit-tech/permission-slip/db/testhelper"
)

// ── Phase 2: Expiration & Isolation (HTTP-level) ────────────────────────────

// TestInviteExpirationBoundary_HTTP registers an agent via an invite that has
// been backdated to just past its expiration. Validates that the handler returns
// 410 Gone with the invite_expired error code.
func TestInviteExpirationBoundary_HTTP(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	ctx := context.Background()

	inviteCode := testhelper.GenerateID(t, "PS-EXP-")
	codeHash := hashCodeHex(inviteCode, "")
	riID := testhelper.GenerateID(t, "ri_")
	if _, err := db.CreateRegistrationInvite(ctx, tx, riID, uid, codeHash, 900); err != nil {
		t.Fatalf("create invite: %v", err)
	}

	// Backdate expires_at to 1 second in the past.
	testhelper.MustExec(t, tx,
		`UPDATE registration_invites SET expires_at = now() - interval '1 second' WHERE id = $1`, riID)

	pubKeySSH, privKey, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	handler := InviteHandler(&Deps{DB: tx})

	body := fmt.Sprintf(`{"request_id":"req-exp","public_key":%q}`, pubKeySSH)
	bodyBytes := []byte(body)
	r := httptest.NewRequest(http.MethodPost, "/invite/"+inviteCode, io.NopCloser(strings.NewReader(body)))
	r.Header.Set("Content-Type", "application/json")
	SignRequest(privKey, 0, r, bodyBytes)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusGone {
		t.Fatalf("expected 410 Gone for expired invite, got %d: %s", w.Code, w.Body.String())
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("unmarshal error response: %v", err)
	}
	if errResp.Error.Code != ErrInviteExpired {
		t.Errorf("expected error code %q, got %q", ErrInviteExpired, errResp.Error.Code)
	}
}

// TestAgentRegistrationTTLBoundary_HTTP registers an agent via invite, then
// backdates the agent's expires_at and attempts verification. Validates that
// the handler returns 410 Gone with the registration_expired error code.
func TestAgentRegistrationTTLBoundary_HTTP(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	// Register agent via invite to get a real pending agent.
	reg := registerViaInvite(t, tx, uid)

	// Backdate the agent's expires_at to simulate TTL expiration.
	testhelper.MustExec(t, tx,
		`UPDATE agents SET expires_at = now() - interval '1 second' WHERE agent_id = $1`, reg.AgentID)

	router := NewRouter(&Deps{DB: tx, SupabaseJWTSecret: testJWTSecret})

	r, _ := signedVerifyRequest(t, reg, reg.ConfirmCode, "verify-exp")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusGone {
		t.Fatalf("expected 410 Gone for expired registration TTL, got %d: %s", w.Code, w.Body.String())
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("unmarshal error response: %v", err)
	}
	if errResp.Error.Code != ErrRegistrationExpired {
		t.Errorf("expected error code %q, got %q", ErrRegistrationExpired, errResp.Error.Code)
	}

	// Verify details include the expired_at timestamp.
	if errResp.Error.Details == nil {
		t.Error("expected error details with expired_at")
	} else if _, ok := errResp.Error.Details["expired_at"]; !ok {
		t.Error("expected 'expired_at' in error details")
	}
}

// TestLockoutVsExpirationPrecedence_HTTP triggers lockout (5 wrong codes) and
// then expires the TTL. Verifies the HTTP response reflects expiration taking
// precedence over lockout in the diagnosis path.
func TestLockoutVsExpirationPrecedence_HTTP(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	reg := registerViaInvite(t, tx, uid)

	router := NewRouter(&Deps{DB: tx, SupabaseJWTSecret: testJWTSecret})

	// Submit 5 wrong codes to trigger lockout.
	submitWrongVerifyCodes(t, router, reg, 5)

	// 6th attempt — agent should be locked out (410 with verification_locked).
	r, _ := signedVerifyRequest(t, reg, "ZZZZZ-ZZZZZ", "lockexp-check")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusGone {
		t.Fatalf("expected 410 Gone for locked out agent, got %d: %s", w.Code, w.Body.String())
	}

	var lockResp ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &lockResp); err != nil {
		t.Fatalf("unmarshal lock response: %v", err)
	}
	if lockResp.Error.Code != ErrVerificationLocked {
		t.Errorf("expected %q, got %q", ErrVerificationLocked, lockResp.Error.Code)
	}

	// Now also expire the TTL.
	testhelper.MustExec(t, tx,
		`UPDATE agents SET expires_at = now() - interval '1 hour' WHERE agent_id = $1`, reg.AgentID)

	// With both lockout and expiration, expiration should take precedence in the
	// diagnosis path (it's checked before lockout in diagnosePendingAgent).
	r2, _ := signedVerifyRequest(t, reg, reg.ConfirmCode, "lockexp-both")
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, r2)

	if w2.Code != http.StatusGone {
		t.Fatalf("expected 410 for lockout+expiration, got %d: %s", w2.Code, w2.Body.String())
	}

	var bothResp ErrorResponse
	if err := json.Unmarshal(w2.Body.Bytes(), &bothResp); err != nil {
		t.Fatalf("unmarshal both response: %v", err)
	}
	// Expiration is checked before lockout in diagnosePendingAgent, so
	// registration_expired should be the error code.
	if bothResp.Error.Code != ErrRegistrationExpired {
		t.Errorf("expected %q (expiration precedence over lockout), got %q",
			ErrRegistrationExpired, bothResp.Error.Code)
	}
}

// TestCrossUserInviteIsolation verifies that an invite created by user A
// correctly assigns the agent to user A's approver_id, and user B cannot
// see or manage that agent via the dashboard API.
func TestCrossUserInviteIsolation(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	// Create two separate users.
	uidA := testhelper.GenerateUID(t)
	uidB := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uidA, "userA_"+uidA[:6])
	testhelper.InsertUser(t, tx, uidB, "userB_"+uidB[:6])

	// User A creates an invite and an agent registers via it.
	reg := registerViaInvite(t, tx, uidA)

	router := NewRouter(&Deps{DB: tx, SupabaseJWTSecret: testJWTSecret})

	// Verify the agent belongs to user A.
	agent, err := db.GetAgentByIDUnscoped(context.Background(), tx, reg.AgentID)
	if err != nil {
		t.Fatalf("GetAgentByIDUnscoped: %v", err)
	}
	if agent.ApproverID != uidA {
		t.Fatalf("expected approver_id=%q (user A), got %q", uidA, agent.ApproverID)
	}

	// User A can see the agent via GET /agents/{id}.
	rA := authenticatedRequest(t, http.MethodGet, fmt.Sprintf("/agents/%d", reg.AgentID), uidA)
	wA := httptest.NewRecorder()
	router.ServeHTTP(wA, rA)

	if wA.Code != http.StatusOK {
		t.Fatalf("user A GET agent: expected 200, got %d: %s", wA.Code, wA.Body.String())
	}

	// User B cannot see user A's agent — should get 404.
	rB := authenticatedRequest(t, http.MethodGet, fmt.Sprintf("/agents/%d", reg.AgentID), uidB)
	wB := httptest.NewRecorder()
	router.ServeHTTP(wB, rB)

	if wB.Code != http.StatusNotFound {
		t.Fatalf("user B GET agent: expected 404, got %d: %s", wB.Code, wB.Body.String())
	}

	// User B cannot register (approve via dashboard) user A's agent — should get 404.
	rBReg := authenticatedRequest(t, http.MethodPost, fmt.Sprintf("/agents/%d/register", reg.AgentID), uidB)
	wBReg := httptest.NewRecorder()
	router.ServeHTTP(wBReg, rBReg)

	if wBReg.Code != http.StatusNotFound {
		t.Fatalf("user B register agent: expected 404, got %d: %s", wBReg.Code, wBReg.Body.String())
	}

	// User B cannot deactivate user A's agent — should get 404.
	rBDeact := authenticatedRequest(t, http.MethodPost, fmt.Sprintf("/agents/%d/deactivate", reg.AgentID), uidB)
	wBDeact := httptest.NewRecorder()
	router.ServeHTTP(wBDeact, rBDeact)

	if wBDeact.Code != http.StatusNotFound {
		t.Fatalf("user B deactivate agent: expected 404, got %d: %s", wBDeact.Code, wBDeact.Body.String())
	}

	// User B cannot update metadata on user A's agent — should get 404.
	rBUpdate := authenticatedJSONRequest(t, http.MethodPatch, fmt.Sprintf("/agents/%d", reg.AgentID), uidB, `{"metadata":{"evil":"data"}}`)
	wBUpdate := httptest.NewRecorder()
	router.ServeHTTP(wBUpdate, rBUpdate)

	if wBUpdate.Code != http.StatusNotFound {
		t.Fatalf("user B update agent: expected 404, got %d: %s", wBUpdate.Code, wBUpdate.Body.String())
	}

	// User A CAN register the agent via dashboard (it's still pending).
	rAReg := authenticatedRequest(t, http.MethodPost, fmt.Sprintf("/agents/%d/register", reg.AgentID), uidA)
	wAReg := httptest.NewRecorder()
	router.ServeHTTP(wAReg, rAReg)

	if wAReg.Code != http.StatusOK {
		t.Fatalf("user A register agent: expected 200, got %d: %s", wAReg.Code, wAReg.Body.String())
	}

	var resp agentResponse
	if err := json.Unmarshal(wAReg.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Status != "registered" {
		t.Errorf("expected status 'registered', got %q", resp.Status)
	}
}

// TestConfirmationCodeIsolation_HTTP verifies that using agent A's confirmation
// code on agent B's verify endpoint fails. Each code is bound to a specific
// agent and is not transferable.
func TestConfirmationCodeIsolation_HTTP(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	// Register two agents via invite (each gets its own confirmation code).
	regA := registerViaInvite(t, tx, uid)
	regB := registerViaInvite(t, tx, uid)

	// Sanity check: codes are different.
	if regA.ConfirmCode == regB.ConfirmCode {
		t.Fatalf("expected different confirmation codes, both got %q", regA.ConfirmCode)
	}

	router := NewRouter(&Deps{DB: tx, SupabaseJWTSecret: testJWTSecret})

	// Try to verify agent B using agent A's code — should fail with 401.
	r, _ := signedVerifyRequest(t, regB, regA.ConfirmCode, "iso-b-with-a")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("agent B with agent A's code: expected 401, got %d: %s", w.Code, w.Body.String())
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if errResp.Error.Code != ErrInvalidCode {
		t.Errorf("expected error code %q, got %q", ErrInvalidCode, errResp.Error.Code)
	}

	// Verify that attempts_remaining is present and decremented.
	if errResp.Error.Details == nil {
		t.Fatal("expected error details with attempts_remaining")
	}
	remaining, ok := errResp.Error.Details["attempts_remaining"]
	if !ok {
		t.Fatal("expected 'attempts_remaining' in error details")
	}
	if rem, ok := remaining.(float64); !ok || rem != 4 {
		t.Errorf("expected 4 attempts_remaining, got %v", remaining)
	}

	// Try to verify agent A using agent B's code — should also fail.
	r2, _ := signedVerifyRequest(t, regA, regB.ConfirmCode, "iso-a-with-b")
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, r2)

	if w2.Code != http.StatusUnauthorized {
		t.Fatalf("agent A with agent B's code: expected 401, got %d: %s", w2.Code, w2.Body.String())
	}

	// Now verify each agent with its OWN code — both should succeed.
	rA, _ := signedVerifyRequest(t, regA, regA.ConfirmCode, "iso-a-own")
	wA := httptest.NewRecorder()
	router.ServeHTTP(wA, rA)

	if wA.Code != http.StatusOK {
		t.Fatalf("agent A with own code: expected 200, got %d: %s", wA.Code, wA.Body.String())
	}

	rB, _ := signedVerifyRequest(t, regB, regB.ConfirmCode, "iso-b-own")
	wB := httptest.NewRecorder()
	router.ServeHTTP(wB, rB)

	if wB.Code != http.StatusOK {
		t.Fatalf("agent B with own code: expected 200, got %d: %s", wB.Code, wB.Body.String())
	}
}
