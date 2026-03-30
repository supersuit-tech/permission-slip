package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip/db"
	"github.com/supersuit-tech/permission-slip/db/testhelper"
)

// ── Phase 3: Error Paths & Edge Cases ────────────────────────────────────────
//
// These integration tests cover error paths and edge cases in the agent
// registration flow, as specified in issue #227 Phase 3.

// ── 1. Invalid public key formats ────────────────────────────────────────────

func TestInviteRegister_InvalidPublicKeyFormats(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		publicKey string
		wantCode  ErrorCode
	}{
		{
			name:      "wrong key type (ssh-rsa)",
			publicKey: "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQDFakeKeyDataHere==",
			wantCode:  ErrInvalidPublicKey,
		},
		{
			name:      "invalid base64 payload",
			publicKey: "ssh-ed25519 !!!not-valid-base64!!!",
			wantCode:  ErrInvalidPublicKey,
		},
		{
			name:      "truncated key data",
			publicKey: "ssh-ed25519 " + base64.StdEncoding.EncodeToString([]byte("short")),
			wantCode:  ErrInvalidPublicKey,
		},
		{
			name:      "empty key type",
			publicKey: "",
			wantCode:  ErrInvalidRequest, // caught by required field validation
		},
		{
			name:      "valid prefix but corrupted wire format",
			publicKey: "ssh-ed25519 " + base64.StdEncoding.EncodeToString(make([]byte, 51)),
			wantCode:  ErrInvalidPublicKey,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tx := testhelper.SetupTestDB(t)
			uid := testhelper.GenerateUID(t)
			testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

			inviteCode := testhelper.GenerateID(t, "PS-")
			codeHash := hashCodeHex(inviteCode, "")
			riID := testhelper.GenerateID(t, "ri_")
			if _, err := db.CreateRegistrationInvite(context.Background(), tx, riID, uid, codeHash, 900); err != nil {
				t.Fatalf("create invite: %v", err)
			}

			// Use a valid key pair for signing (the signature itself needs to be valid,
			// but the public_key in the body is the one being tested).
			_, privKey, err := GenerateEd25519OpenSSHKey()
			if err != nil {
				t.Fatalf("generate key: %v", err)
			}

			handler := InviteHandler(&Deps{DB: tx})

			body := fmt.Sprintf(`{"request_id":"req-%s","public_key":%q}`, testhelper.GenerateID(t, ""), tt.publicKey)
			bodyBytes := []byte(body)
			r := httptest.NewRequest(http.MethodPost, "/invite/"+inviteCode, io.NopCloser(strings.NewReader(body)))
			r.Header.Set("Content-Type", "application/json")
			SignRequest(privKey, 0, r, bodyBytes)

			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)

			if w.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
			}

			var errResp ErrorResponse
			if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil {
				t.Fatalf("unmarshal error: %v", err)
			}
			if errResp.Error.Code != tt.wantCode {
				t.Errorf("expected error code %q, got %q", tt.wantCode, errResp.Error.Code)
			}
		})
	}
}

// ── 2. Confirmation code normalization ───────────────────────────────────────

func TestVerifyRegistration_ConfirmationCodeNormalization(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		transform  func(code string) string
		wantStatus int
	}{
		{
			name:       "lowercase input",
			transform:  strings.ToLower,
			wantStatus: http.StatusOK,
		},
		{
			name: "mixed case",
			transform: func(code string) string {
				// Alternate case: first char lower, second upper, etc.
				var out []byte
				for i, c := range code {
					if i%2 == 0 {
						out = append(out, byte(c|0x20)) // lowercase ASCII letter
					} else {
						out = append(out, byte(c))
					}
				}
				return string(out)
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "with extra hyphens",
			transform: func(code string) string {
				// Add hyphens at non-standard positions: e.g. "X-K-7-M-9-P"
				var out []byte
				for i, c := range code {
					if c == '-' {
						continue
					}
					if i > 0 && len(out) > 0 {
						out = append(out, '-')
					}
					out = append(out, byte(c))
				}
				return string(out)
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "no hyphens",
			transform: func(code string) string {
				return strings.ReplaceAll(code, "-", "")
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "wrong code entirely",
			transform: func(_ string) string {
				return "ZZZ-ZZZ"
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "special characters appended",
			transform: func(code string) string {
				return code + "!"
			},
			wantStatus: http.StatusBadRequest, // length check fails after normalization (7+ chars)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tx := testhelper.SetupTestDB(t)
			uid := testhelper.GenerateUID(t)
			testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

			reg := registerViaInvite(t, tx, uid)
			router := NewRouter(&Deps{DB: tx, SupabaseJWTSecret: testJWTSecret})

			submittedCode := tt.transform(reg.ConfirmCode)
			body := fmt.Sprintf(`{"request_id":"verify-norm","confirmation_code":%q}`, submittedCode)
			bodyBytes := []byte(body)
			path := fmt.Sprintf("/agents/%d/verify", reg.AgentID)
			r := httptest.NewRequest(http.MethodPost, path, io.NopCloser(strings.NewReader(body)))
			r.Header.Set("Content-Type", "application/json")
			SignRequest(reg.PrivKey, reg.AgentID, r, bodyBytes)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, r)

			if w.Code != tt.wantStatus {
				t.Fatalf("expected %d, got %d: %s", tt.wantStatus, w.Code, w.Body.String())
			}
		})
	}
}

// ── 3. Metadata preservation through lifecycle ───────────────────────────────

func TestMetadataPreservation_ThroughLifecycle(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	// Step 1: Register via invite with metadata.
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

	metadata := `{"agent_name":"test-bot","version":"1.2.3","tags":["ci","deploy"]}`
	handler := InviteHandler(&Deps{DB: tx})

	body := fmt.Sprintf(`{"request_id":"req-meta","public_key":%q,"metadata":%s}`, pubKeySSH, metadata)
	bodyBytes := []byte(body)
	r := httptest.NewRequest(http.MethodPost, "/invite/"+inviteCode, io.NopCloser(strings.NewReader(body)))
	r.Header.Set("Content-Type", "application/json")
	SignRequest(privKey, 0, r, bodyBytes)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("register: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var regResp registerAgentResponse
	if err := json.Unmarshal(w.Body.Bytes(), &regResp); err != nil {
		t.Fatalf("unmarshal register response: %v", err)
	}
	agentID := regResp.AgentID

	// Step 2: Verify the pending agent has metadata.
	pendingAgent, err := db.GetAgentByIDUnscoped(context.Background(), tx, agentID)
	if err != nil {
		t.Fatalf("get pending agent: %v", err)
	}
	if pendingAgent.Metadata == nil {
		t.Fatal("expected metadata on pending agent, got nil")
	}

	var pendingMeta map[string]any
	if err := json.Unmarshal(pendingAgent.Metadata, &pendingMeta); err != nil {
		t.Fatalf("unmarshal pending metadata: %v", err)
	}
	if pendingMeta["agent_name"] != "test-bot" {
		t.Errorf("pending metadata agent_name: expected 'test-bot', got %v", pendingMeta["agent_name"])
	}
	if pendingMeta["version"] != "1.2.3" {
		t.Errorf("pending metadata version: expected '1.2.3', got %v", pendingMeta["version"])
	}

	// Step 3: Verify the confirmation code (complete registration).
	confirmCode := *pendingAgent.ConfirmationCode

	router := NewRouter(&Deps{DB: tx, SupabaseJWTSecret: testJWTSecret})

	verifyBody := fmt.Sprintf(`{"request_id":"verify-meta","confirmation_code":%q}`, confirmCode)
	verifyBodyBytes := []byte(verifyBody)
	verifyPath := fmt.Sprintf("/agents/%d/verify", agentID)
	vr := httptest.NewRequest(http.MethodPost, verifyPath, io.NopCloser(strings.NewReader(verifyBody)))
	vr.Header.Set("Content-Type", "application/json")
	SignRequest(privKey, agentID, vr, verifyBodyBytes)

	vw := httptest.NewRecorder()
	router.ServeHTTP(vw, vr)

	if vw.Code != http.StatusOK {
		t.Fatalf("verify: expected 200, got %d: %s", vw.Code, vw.Body.String())
	}

	// Step 4: Verify metadata survives through verification.
	registeredAgent, err := db.GetAgentByIDUnscoped(context.Background(), tx, agentID)
	if err != nil {
		t.Fatalf("get registered agent: %v", err)
	}
	if registeredAgent.Status != "registered" {
		t.Fatalf("expected status 'registered', got %q", registeredAgent.Status)
	}
	if registeredAgent.Metadata == nil {
		t.Fatal("expected metadata on registered agent, got nil")
	}

	var registeredMeta map[string]any
	if err := json.Unmarshal(registeredAgent.Metadata, &registeredMeta); err != nil {
		t.Fatalf("unmarshal registered metadata: %v", err)
	}
	if registeredMeta["agent_name"] != "test-bot" {
		t.Errorf("registered metadata agent_name: expected 'test-bot', got %v", registeredMeta["agent_name"])
	}
	if registeredMeta["version"] != "1.2.3" {
		t.Errorf("registered metadata version: expected '1.2.3', got %v", registeredMeta["version"])
	}

	// Verify tags array survived.
	tags, ok := registeredMeta["tags"].([]any)
	if !ok {
		t.Fatalf("expected tags to be an array, got %T", registeredMeta["tags"])
	}
	if len(tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(tags))
	}

	// Step 5: Verify metadata is returned correctly via GET /agents/{id}.
	getR := authenticatedRequest(t, http.MethodGet, fmt.Sprintf("/agents/%d", agentID), uid)
	getW := httptest.NewRecorder()
	router.ServeHTTP(getW, getR)

	if getW.Code != http.StatusOK {
		t.Fatalf("get agent: expected 200, got %d: %s", getW.Code, getW.Body.String())
	}

	var agentResp agentResponse
	if err := json.Unmarshal(getW.Body.Bytes(), &agentResp); err != nil {
		t.Fatalf("unmarshal agent response: %v", err)
	}

	// The metadata field is deserialized as any — re-marshal and compare.
	if agentResp.Metadata == nil {
		t.Fatal("expected metadata in GET response, got nil")
	}
	respMetaBytes, err := json.Marshal(agentResp.Metadata)
	if err != nil {
		t.Fatalf("re-marshal metadata: %v", err)
	}
	var respMeta map[string]any
	if err := json.Unmarshal(respMetaBytes, &respMeta); err != nil {
		t.Fatalf("unmarshal response metadata: %v", err)
	}
	if respMeta["agent_name"] != "test-bot" {
		t.Errorf("GET response metadata agent_name: expected 'test-bot', got %v", respMeta["agent_name"])
	}
}

// ── 4. Cannot verify deactivated agent ───────────────────────────────────────

func TestVerifyRegistration_DeactivatedAgent(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	// Register via invite to get a pending agent.
	reg := registerViaInvite(t, tx, uid)

	// Deactivate the pending agent via the dashboard endpoint.
	router := NewRouter(&Deps{DB: tx, SupabaseJWTSecret: testJWTSecret})

	deactivateR := authenticatedRequest(t, http.MethodPost, fmt.Sprintf("/agents/%d/deactivate", reg.AgentID), uid)
	deactivateW := httptest.NewRecorder()
	router.ServeHTTP(deactivateW, deactivateR)

	if deactivateW.Code != http.StatusOK {
		t.Fatalf("deactivate: expected 200, got %d: %s", deactivateW.Code, deactivateW.Body.String())
	}

	// Confirm the agent is now deactivated.
	agent, err := db.GetAgentByIDUnscoped(context.Background(), tx, reg.AgentID)
	if err != nil {
		t.Fatalf("get agent: %v", err)
	}
	if agent.Status != "deactivated" {
		t.Fatalf("expected status 'deactivated', got %q", agent.Status)
	}

	// Attempt to verify with the correct confirmation code — should fail.
	body := fmt.Sprintf(`{"request_id":"verify-deact","confirmation_code":%q}`, reg.ConfirmCode)
	bodyBytes := []byte(body)
	path := fmt.Sprintf("/agents/%d/verify", reg.AgentID)
	r := httptest.NewRequest(http.MethodPost, path, io.NopCloser(strings.NewReader(body)))
	r.Header.Set("Content-Type", "application/json")
	SignRequest(reg.PrivKey, reg.AgentID, r, bodyBytes)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	// The agent is not pending, so verification should fail.
	// Depending on the code path, it returns 404 (not found/not pending) or 409 (already registered).
	// For a deactivated agent, it should be 404 (not pending, not registered).
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if errResp.Error.Code != ErrAgentNotFound {
		t.Errorf("expected error code %q, got %q", ErrAgentNotFound, errResp.Error.Code)
	}
}

// ── 5. Deactivated agent re-registration ─────────────────────────────────────

func TestRegisterAgent_DeactivatedCannotReRegister(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	// Step 1: Register via invite and verify.
	reg := registerViaInvite(t, tx, uid)

	router := NewRouter(&Deps{DB: tx, SupabaseJWTSecret: testJWTSecret})

	// Verify the agent.
	verifyBody := fmt.Sprintf(`{"request_id":"verify-re","confirmation_code":%q}`, reg.ConfirmCode)
	verifyBodyBytes := []byte(verifyBody)
	verifyPath := fmt.Sprintf("/agents/%d/verify", reg.AgentID)
	vr := httptest.NewRequest(http.MethodPost, verifyPath, io.NopCloser(strings.NewReader(verifyBody)))
	vr.Header.Set("Content-Type", "application/json")
	SignRequest(reg.PrivKey, reg.AgentID, vr, verifyBodyBytes)

	vw := httptest.NewRecorder()
	router.ServeHTTP(vw, vr)
	if vw.Code != http.StatusOK {
		t.Fatalf("verify: expected 200, got %d: %s", vw.Code, vw.Body.String())
	}

	// Step 2: Deactivate the agent.
	deactivateR := authenticatedRequest(t, http.MethodPost, fmt.Sprintf("/agents/%d/deactivate", reg.AgentID), uid)
	deactivateW := httptest.NewRecorder()
	router.ServeHTTP(deactivateW, deactivateR)
	if deactivateW.Code != http.StatusOK {
		t.Fatalf("deactivate: expected 200, got %d: %s", deactivateW.Code, deactivateW.Body.String())
	}

	// Confirm agent is deactivated.
	agent, err := db.GetAgentByIDUnscoped(context.Background(), tx, reg.AgentID)
	if err != nil {
		t.Fatalf("get agent: %v", err)
	}
	if agent.Status != "deactivated" {
		t.Fatalf("expected status 'deactivated', got %q", agent.Status)
	}

	// Step 3: Attempt to re-register via the dashboard endpoint — should fail.
	reRegR := authenticatedRequest(t, http.MethodPost, fmt.Sprintf("/agents/%d/register", reg.AgentID), uid)
	reRegW := httptest.NewRecorder()
	router.ServeHTTP(reRegW, reRegR)

	if reRegW.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", reRegW.Code, reRegW.Body.String())
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(reRegW.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if errResp.Error.Code != ErrAgentAlreadyRegistered {
		t.Errorf("expected error code %q, got %q", ErrAgentAlreadyRegistered, errResp.Error.Code)
	}
	// Verify the message mentions "deactivated".
	if !strings.Contains(errResp.Error.Message, "deactivated") {
		t.Errorf("expected error message to mention 'deactivated', got %q", errResp.Error.Message)
	}
}

// ── 6. Request signature timestamp validation ────────────────────────────────

func TestInviteRegister_TimestampValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		offset    time.Duration
		wantCode  int
		wantError ErrorCode
	}{
		{
			name:      "timestamp 10 minutes in the past",
			offset:    -10 * time.Minute,
			wantCode:  http.StatusUnauthorized,
			wantError: ErrTimestampExpired,
		},
		{
			name:      "timestamp 10 minutes in the future",
			offset:    10 * time.Minute,
			wantCode:  http.StatusUnauthorized,
			wantError: ErrTimestampExpired,
		},
		{
			name:      "timestamp 1 hour in the past",
			offset:    -1 * time.Hour,
			wantCode:  http.StatusUnauthorized,
			wantError: ErrTimestampExpired,
		},
		{
			name:      "timestamp at boundary (4 minutes ago, within window)",
			offset:    -4 * time.Minute,
			wantCode:  http.StatusOK,
			wantError: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tx := testhelper.SetupTestDB(t)
			uid := testhelper.GenerateUID(t)
			testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

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

			body := fmt.Sprintf(`{"request_id":"req-ts","public_key":%q}`, pubKeySSH)
			bodyBytes := []byte(body)
			r := httptest.NewRequest(http.MethodPost, "/invite/"+inviteCode, io.NopCloser(strings.NewReader(body)))
			r.Header.Set("Content-Type", "application/json")

			customTimestamp := time.Now().Add(tt.offset).Unix()
			SignRequestAt(privKey, 0, r, bodyBytes, customTimestamp)

			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)

			if w.Code != tt.wantCode {
				t.Fatalf("expected %d, got %d: %s", tt.wantCode, w.Code, w.Body.String())
			}

			if tt.wantError != "" {
				var errResp ErrorResponse
				if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil {
					t.Fatalf("unmarshal error: %v", err)
				}
				if errResp.Error.Code != tt.wantError {
					t.Errorf("expected error code %q, got %q", tt.wantError, errResp.Error.Code)
				}
			}
		})
	}
}

func TestVerifyRegistration_TimestampValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		offset    time.Duration
		wantCode  int
		wantError ErrorCode
	}{
		{
			name:      "timestamp far in the past",
			offset:    -10 * time.Minute,
			wantCode:  http.StatusUnauthorized,
			wantError: ErrTimestampExpired,
		},
		{
			name:      "timestamp far in the future",
			offset:    10 * time.Minute,
			wantCode:  http.StatusUnauthorized,
			wantError: ErrTimestampExpired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tx := testhelper.SetupTestDB(t)
			uid := testhelper.GenerateUID(t)
			testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

			reg := registerViaInvite(t, tx, uid)
			router := NewRouter(&Deps{DB: tx, SupabaseJWTSecret: testJWTSecret})

			body := fmt.Sprintf(`{"request_id":"verify-ts","confirmation_code":%q}`, reg.ConfirmCode)
			bodyBytes := []byte(body)
			path := fmt.Sprintf("/agents/%d/verify", reg.AgentID)
			r := httptest.NewRequest(http.MethodPost, path, io.NopCloser(strings.NewReader(body)))
			r.Header.Set("Content-Type", "application/json")

			customTimestamp := time.Now().Add(tt.offset).Unix()
			SignRequestAt(reg.PrivKey, reg.AgentID, r, bodyBytes, customTimestamp)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, r)

			if w.Code != tt.wantCode {
				t.Fatalf("expected %d, got %d: %s", tt.wantCode, w.Code, w.Body.String())
			}

			var errResp ErrorResponse
			if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil {
				t.Fatalf("unmarshal error: %v", err)
			}
			if errResp.Error.Code != tt.wantError {
				t.Errorf("expected error code %q, got %q", tt.wantError, errResp.Error.Code)
			}
		})
	}
}
