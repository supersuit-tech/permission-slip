package api

import (
	"context"
	"crypto/ed25519"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/supersuit-tech/permission-slip/db"
)

// signedJSONRequest creates an HTTP request with a JSON body and an Ed25519
// signature header. This is the standard pattern for agent-authenticated
// endpoints (invite registration, verification).
func signedJSONRequest(t *testing.T, method, path, body string, privKey ed25519.PrivateKey, agentID int64) *http.Request {
	t.Helper()
	bodyBytes := []byte(body)
	r := httptest.NewRequest(method, path, io.NopCloser(strings.NewReader(body)))
	r.Header.Set("Content-Type", "application/json")
	SignRequest(privKey, agentID, r, bodyBytes)
	return r
}

// requireExactlyOneSuccess asserts that exactly one goroutine returned 200 OK
// and the rest returned the expected failure status code. This is the standard
// assertion pattern for concurrent race tests.
func requireExactlyOneSuccess(t *testing.T, results []int, expectedFailStatus int) {
	t.Helper()

	successCount := 0
	failCount := 0
	for i, code := range results {
		switch code {
		case http.StatusOK:
			successCount++
		case expectedFailStatus:
			failCount++
		default:
			t.Errorf("goroutine %d: unexpected status %d (expected 200 or %d)", i, code, expectedFailStatus)
		}
	}

	expectedFails := len(results) - 1
	if successCount != 1 {
		t.Errorf("expected exactly 1 success (200), got %d (results: %v)", successCount, results)
	}
	if failCount != expectedFails {
		t.Errorf("expected %d failures (%d), got %d (results: %v)", expectedFails, expectedFailStatus, failCount, results)
	}
}

// inviteRequestBody returns a JSON body for POST /invite/{code} registration.
func inviteRequestBody(t *testing.T, requestID, pubKeySSH string) string {
	t.Helper()
	return fmt.Sprintf(`{"request_id":%q,"public_key":%q}`, requestID, pubKeySSH)
}

// verifyRequestBody returns a JSON body for POST /agents/{id}/verify.
func verifyRequestBody(requestID, confirmCode string) string {
	return fmt.Sprintf(`{"request_id":%q,"confirmation_code":%q}`, requestID, confirmCode)
}

// insertRegisteredAgentWithKey creates a registered agent with a real Ed25519
// public key. Returns the agent_id and private key for signing.
func insertRegisteredAgentWithKey(t *testing.T, tx db.DBTX, approverID string) (int64, ed25519.PrivateKey) {
	t.Helper()
	pubKeySSH, privKey, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	var agentID int64
	err = tx.QueryRow(context.Background(),
		`INSERT INTO agents (public_key, approver_id, status, registered_at) VALUES ($1, $2, 'registered', now()) RETURNING agent_id`,
		pubKeySSH, approverID).Scan(&agentID)
	if err != nil {
		t.Fatalf("insert agent with key: %v", err)
	}
	return agentID, privKey
}
