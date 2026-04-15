package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip/db/testhelper"
)

// TestSignatureReplay_DuplicateRejected verifies that the same signed request,
// replayed a second time, is rejected with 401 by the agent-auth path.
// Uses SignRequestAt so both requests carry byte-identical signatures; Ed25519
// is deterministic for the same input, so SignRequest with two clock ticks
// would work too, but pinning the timestamp makes the intent explicit.
func TestSignatureReplay_DuplicateRejected(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	agentID, privKey := insertRegisteredAgentWithKey(t, tx, uid)

	router := NewRouter(&Deps{DB: tx, SupabaseJWTSecret: testJWTSecret, BaseURL: "https://app.permissionslip.dev"})

	path := fmt.Sprintf("/agents/%d/capabilities", agentID)
	ts := time.Now().Unix()

	// First request — should succeed.
	r1 := httptest.NewRequest(http.MethodGet, path, nil)
	SignRequestAt(privKey, agentID, r1, nil, ts)
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, r1)
	if w1.Code != http.StatusOK {
		t.Fatalf("first request: expected 200, got %d: %s", w1.Code, w1.Body.String())
	}

	// Replay — identical signature header, same timestamp. Must be rejected.
	r2 := httptest.NewRequest(http.MethodGet, path, nil)
	SignRequestAt(privKey, agentID, r2, nil, ts)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, r2)

	if w2.Code != http.StatusUnauthorized {
		t.Fatalf("replay: expected 401, got %d: %s", w2.Code, w2.Body.String())
	}
}

// TestSignatureReplay_DifferentTimestampsAllowed verifies the normal case:
// two distinct signatures (different timestamps → different Ed25519 output)
// both pass, confirming replay protection does not interfere with legitimate
// repeated API usage.
func TestSignatureReplay_DifferentTimestampsAllowed(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	agentID, privKey := insertRegisteredAgentWithKey(t, tx, uid)

	router := NewRouter(&Deps{DB: tx, SupabaseJWTSecret: testJWTSecret, BaseURL: "https://app.permissionslip.dev"})

	path := fmt.Sprintf("/agents/%d/capabilities", agentID)
	now := time.Now().Unix()

	r1 := httptest.NewRequest(http.MethodGet, path, nil)
	SignRequestAt(privKey, agentID, r1, nil, now)
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, r1)
	if w1.Code != http.StatusOK {
		t.Fatalf("first request: expected 200, got %d: %s", w1.Code, w1.Body.String())
	}

	r2 := httptest.NewRequest(http.MethodGet, path, nil)
	SignRequestAt(privKey, agentID, r2, nil, now-1) // different timestamp → different signature
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, r2)
	if w2.Code != http.StatusOK {
		t.Fatalf("second request with different timestamp: expected 200, got %d: %s", w2.Code, w2.Body.String())
	}
}
