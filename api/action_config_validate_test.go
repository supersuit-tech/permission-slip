package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/db"
	"github.com/supersuit-tech/permission-slip-web/db/testhelper"
)

// withTraceID adds a trace ID to the context for test requests.
func withTraceID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, traceIDKey{}, id)
}

// newTestRequest creates a Deps, ResponseRecorder, and Request with a trace ID
// for use in ValidateConfigurationReference tests.
func newTestRequest(tx db.DBTX) (*Deps, *httptest.ResponseRecorder, *http.Request) {
	deps := &Deps{DB: tx}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/test", nil)
	r = r.WithContext(withTraceID(r.Context(), "test-trace"))
	return deps, w, r
}

// setupConfigValidateTest creates a user, registered agent, connector, action,
// and an action configuration with the given parameters.
// Returns (tx, userID, agentID, connectorID, actionType, configID).
func setupConfigValidateTest(t *testing.T, params []byte) (db.DBTX, string, int64, string, string, string) {
	t.Helper()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	agentID := testhelper.InsertAgentWithStatus(t, tx, uid, "registered")
	connID := testhelper.GenerateID(t, "conn_")
	actionType := connID + ".test_action"
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertConnectorAction(t, tx, connID, actionType, "Test Action")

	configID := testhelper.GenerateID(t, "ac_")
	testhelper.InsertActionConfigFull(t, tx, configID, agentID, uid, connID, actionType, testhelper.ActionConfigOpts{
		Name:       "Test Config",
		Parameters: params,
	})

	return tx, uid, agentID, connID, actionType, configID
}

func TestValidateConfigurationReference_Success(t *testing.T) {
	t.Parallel()
	params := []byte(`{"repo":"supersuit-tech/webapp","title":"*","body":"*"}`)
	tx, _, agentID, _, actionType, configID := setupConfigValidateTest(t, params)

	deps, w, r := newTestRequest(tx)

	execParams := json.RawMessage(`{"repo":"supersuit-tech/webapp","title":"Fix bug","body":"Details here"}`)

	result := ValidateConfigurationReference(w, r, deps, configID, agentID, actionType, execParams)
	if result == nil {
		t.Fatalf("expected success, got error: %s", w.Body.String())
	}
	if result.Config.ID != configID {
		t.Errorf("expected config ID %q, got %q", configID, result.Config.ID)
	}
}

func TestValidateConfigurationReference_WildcardAcceptsAnyValue(t *testing.T) {
	t.Parallel()
	params := []byte(`{"data":"*"}`)
	tx, _, agentID, _, actionType, configID := setupConfigValidateTest(t, params)

	// Wildcard should accept string, number, array, object, boolean, null.
	tests := []struct {
		name       string
		execParams string
	}{
		{"string", `{"data":"hello"}`},
		{"number", `{"data":42}`},
		{"array", `{"data":[1,2,3]}`},
		{"object", `{"data":{"nested":"value"}}`},
		{"boolean", `{"data":true}`},
		{"null", `{"data":null}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// No t.Parallel() — subtests share the same DB transaction.
			deps, w, r := newTestRequest(tx)

			result := ValidateConfigurationReference(w, r, deps, configID, agentID, actionType, json.RawMessage(tc.execParams))
			if result == nil {
				t.Errorf("expected wildcard to accept %s, got error: %s", tc.name, w.Body.String())
			}
		})
	}
}

func TestValidateConfigurationReference_ConfigNotFound(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	agentID := testhelper.InsertAgentWithStatus(t, tx, uid, "registered")

	deps, w, r := newTestRequest(tx)

	result := ValidateConfigurationReference(w, r, deps, "ac_nonexistent", agentID, "test.action", json.RawMessage(`{}`))
	if result != nil {
		t.Fatal("expected nil result for nonexistent config")
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
	assertErrorCode(t, w.Body.Bytes(), string(ErrInvalidConfiguration))
}

func TestValidateConfigurationReference_DisabledConfig(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	agentID := testhelper.InsertAgentWithStatus(t, tx, uid, "registered")
	connID := testhelper.GenerateID(t, "conn_")
	actionType := connID + ".test_action"
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertConnectorAction(t, tx, connID, actionType, "Test Action")

	configID := testhelper.GenerateID(t, "ac_")
	testhelper.InsertActionConfigFull(t, tx, configID, agentID, uid, connID, actionType, testhelper.ActionConfigOpts{
		Name:   "Disabled Config",
		Status: "disabled",
	})

	deps, w, r := newTestRequest(tx)

	// Disabled configs are filtered out by GetActiveActionConfigForAgent, so we
	// expect the same "not found" response as a nonexistent config.
	result := ValidateConfigurationReference(w, r, deps, configID, agentID, actionType, json.RawMessage(`{}`))
	if result != nil {
		t.Fatal("expected nil result for disabled config")
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
	assertErrorCode(t, w.Body.Bytes(), string(ErrInvalidConfiguration))
}

func TestValidateConfigurationReference_ActionTypeMismatch(t *testing.T) {
	t.Parallel()
	params := []byte(`{"repo":"test/repo"}`)
	tx, _, agentID, _, _, configID := setupConfigValidateTest(t, params)

	deps, w, r := newTestRequest(tx)

	// Supply a different action type than the config's action_type.
	result := ValidateConfigurationReference(w, r, deps, configID, agentID, "different.action", json.RawMessage(`{"repo":"test/repo"}`))
	if result != nil {
		t.Fatal("expected nil result for action type mismatch")
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
	assertErrorCode(t, w.Body.Bytes(), string(ErrConfigActionTypeMismatch))
}

func TestValidateConfigurationReference_FixedParamMismatch(t *testing.T) {
	t.Parallel()
	params := []byte(`{"repo":"supersuit-tech/webapp","title":"*"}`)
	tx, _, agentID, _, actionType, configID := setupConfigValidateTest(t, params)

	deps, w, r := newTestRequest(tx)

	// Provide a different repo value.
	execParams := json.RawMessage(`{"repo":"supersuit-tech/api","title":"Bug fix"}`)
	result := ValidateConfigurationReference(w, r, deps, configID, agentID, actionType, execParams)
	if result != nil {
		t.Fatal("expected nil result for param mismatch")
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
	assertErrorCode(t, w.Body.Bytes(), string(ErrInvalidParameters))
}

func TestValidateConfigurationReference_ExtraParam(t *testing.T) {
	t.Parallel()
	params := []byte(`{"repo":"supersuit-tech/webapp"}`)
	tx, _, agentID, _, actionType, configID := setupConfigValidateTest(t, params)

	deps, w, r := newTestRequest(tx)

	// Provide an extra parameter not in the config.
	execParams := json.RawMessage(`{"repo":"supersuit-tech/webapp","extra":"value"}`)
	result := ValidateConfigurationReference(w, r, deps, configID, agentID, actionType, execParams)
	if result != nil {
		t.Fatal("expected nil result for extra param")
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
	assertErrorCode(t, w.Body.Bytes(), string(ErrInvalidParameters))
}

func TestValidateConfigurationReference_MissingFixedParam(t *testing.T) {
	t.Parallel()
	params := []byte(`{"repo":"supersuit-tech/webapp","label":"bug"}`)
	tx, _, agentID, _, actionType, configID := setupConfigValidateTest(t, params)

	deps, w, r := newTestRequest(tx)

	// Only provide repo but miss label.
	execParams := json.RawMessage(`{"repo":"supersuit-tech/webapp"}`)
	result := ValidateConfigurationReference(w, r, deps, configID, agentID, actionType, execParams)
	if result != nil {
		t.Fatal("expected nil result for missing fixed param")
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
	assertErrorCode(t, w.Body.Bytes(), string(ErrInvalidParameters))
}

func TestValidateConfigurationReference_WrongAgent(t *testing.T) {
	t.Parallel()
	params := []byte(`{"repo":"test/repo"}`)
	tx, _, _, _, _, configID := setupConfigValidateTest(t, params)

	// Create a different agent (registered).
	uid2 := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid2, "u2_"+uid2[:6])
	agentID2 := testhelper.InsertAgentWithStatus(t, tx, uid2, "registered")

	deps, w, r := newTestRequest(tx)

	// Try to use a config that belongs to a different agent.
	result := ValidateConfigurationReference(w, r, deps, configID, agentID2, "test.action", json.RawMessage(`{"repo":"test/repo"}`))
	if result != nil {
		t.Fatal("expected nil result for wrong agent")
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
	assertErrorCode(t, w.Body.Bytes(), string(ErrInvalidConfiguration))
}

// ── Pattern Matching Integration Tests ──────────────────────────────────────

func TestValidateConfigurationReference_PatternMatch(t *testing.T) {
	t.Parallel()
	// Config uses suffix pattern: email must end with @mycompany.com
	params := []byte(`{"to":{"$pattern":"*@mycompany.com"},"subject":"*"}`)
	tx, _, agentID, _, actionType, configID := setupConfigValidateTest(t, params)

	deps, w, r := newTestRequest(tx)

	execParams := json.RawMessage(`{"to":"alice@mycompany.com","subject":"Hello"}`)
	result := ValidateConfigurationReference(w, r, deps, configID, agentID, actionType, execParams)
	if result == nil {
		t.Fatalf("expected success for pattern match, got error: %s", w.Body.String())
	}
}

func TestValidateConfigurationReference_PatternPrefixMatch(t *testing.T) {
	t.Parallel()
	// Config uses prefix pattern: repo must start with supersuit-tech/
	params := []byte(`{"repo":{"$pattern":"supersuit-tech/*"},"title":"*"}`)
	tx, _, agentID, _, actionType, configID := setupConfigValidateTest(t, params)

	deps, w, r := newTestRequest(tx)

	execParams := json.RawMessage(`{"repo":"supersuit-tech/webapp","title":"Fix bug"}`)
	result := ValidateConfigurationReference(w, r, deps, configID, agentID, actionType, execParams)
	if result == nil {
		t.Fatalf("expected success for prefix pattern match, got error: %s", w.Body.String())
	}
}

func TestValidateConfigurationReference_PatternMismatch(t *testing.T) {
	t.Parallel()
	params := []byte(`{"to":{"$pattern":"*@mycompany.com"},"subject":"*"}`)
	tx, _, agentID, _, actionType, configID := setupConfigValidateTest(t, params)

	deps, w, r := newTestRequest(tx)

	// Email doesn't match the pattern
	execParams := json.RawMessage(`{"to":"alice@other.com","subject":"Hello"}`)
	result := ValidateConfigurationReference(w, r, deps, configID, agentID, actionType, execParams)
	if result != nil {
		t.Fatal("expected nil result for pattern mismatch")
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
	assertErrorCode(t, w.Body.Bytes(), string(ErrInvalidParameters))
}

func TestValidateConfigurationReference_PatternMissing(t *testing.T) {
	t.Parallel()
	params := []byte(`{"repo":{"$pattern":"supersuit-tech/*"},"title":"*"}`)
	tx, _, agentID, _, actionType, configID := setupConfigValidateTest(t, params)

	deps, w, r := newTestRequest(tx)

	// Missing the repo pattern parameter (only providing wildcard title)
	execParams := json.RawMessage(`{"title":"Fix bug"}`)
	result := ValidateConfigurationReference(w, r, deps, configID, agentID, actionType, execParams)
	if result != nil {
		t.Fatal("expected nil result for missing pattern param")
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
	assertErrorCode(t, w.Body.Bytes(), string(ErrInvalidParameters))
}

func TestValidateConfigurationReference_PatternRejectsNonString(t *testing.T) {
	t.Parallel()
	params := []byte(`{"tag":{"$pattern":"v*"}}`)
	tx, _, agentID, _, actionType, configID := setupConfigValidateTest(t, params)

	deps, w, r := newTestRequest(tx)

	// Provide a number instead of a string for a pattern param
	execParams := json.RawMessage(`{"tag":42}`)
	result := ValidateConfigurationReference(w, r, deps, configID, agentID, actionType, execParams)
	if result != nil {
		t.Fatal("expected nil result for non-string pattern value")
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
	assertErrorCode(t, w.Body.Bytes(), string(ErrInvalidParameters))
}

func TestValidateConfigurationReference_MixedFixedPatternWildcard(t *testing.T) {
	t.Parallel()
	// All three types: fixed repo org, pattern on branch, wildcard on message
	params := []byte(`{"repo":"supersuit-tech/webapp","branch":{"$pattern":"release/*"},"message":"*"}`)
	tx, _, agentID, _, actionType, configID := setupConfigValidateTest(t, params)

	deps, w, r := newTestRequest(tx)

	execParams := json.RawMessage(`{"repo":"supersuit-tech/webapp","branch":"release/v2.1","message":"Deploy"}`)
	result := ValidateConfigurationReference(w, r, deps, configID, agentID, actionType, execParams)
	if result == nil {
		t.Fatalf("expected success for mixed config, got error: %s", w.Body.String())
	}
}

// assertErrorCode checks that the response body contains the expected error code.
func assertErrorCode(t *testing.T, body []byte, expectedCode string) {
	t.Helper()
	var resp ErrorResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("failed to unmarshal error response: %v", err)
	}
	if string(resp.Error.Code) != expectedCode {
		t.Errorf("expected error code %q, got %q", expectedCode, resp.Error.Code)
	}
}

// ── Wildcard Action Type Tests ──────────────────────────────────────────────

// setupWildcardConfigTest creates a wildcard (action_type="*") config.
// Returns (tx, userID, agentID, connectorID, configID).
func setupWildcardConfigTest(t *testing.T) (db.DBTX, string, int64, string, string) {
	t.Helper()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	agentID := testhelper.InsertAgentWithStatus(t, tx, uid, "registered")
	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertConnectorAction(t, tx, connID, connID+".some_action", "Some Action")

	configID := testhelper.GenerateID(t, "ac_")
	testhelper.InsertActionConfigFull(t, tx, configID, agentID, uid, connID, "*", testhelper.ActionConfigOpts{
		Name: "All Actions",
	})

	return tx, uid, agentID, connID, configID
}

func TestValidateConfigurationReference_WildcardMatchesAnyActionType(t *testing.T) {
	t.Parallel()
	tx, _, agentID, connID, configID := setupWildcardConfigTest(t)

	// Wildcard config should accept any action type.
	actionTypes := []string{
		connID + ".some_action",
		connID + ".another_action",
		"completely.different.action",
	}

	for _, at := range actionTypes {
		t.Run(at, func(t *testing.T) {
			deps, w, r := newTestRequest(tx)
			execParams := json.RawMessage(`{"any_param":"any_value"}`)
			result := ValidateConfigurationReference(w, r, deps, configID, agentID, at, execParams)
			if result == nil {
				t.Errorf("expected wildcard config to match action type %q, got error: %s", at, w.Body.String())
			}
		})
	}
}

func TestValidateConfigurationReference_WildcardAcceptsAnyParameters(t *testing.T) {
	t.Parallel()
	tx, _, agentID, connID, configID := setupWildcardConfigTest(t)

	// Wildcard config should accept any parameters (including extra ones).
	tests := []struct {
		name       string
		execParams string
	}{
		{"empty params", `{}`},
		{"single param", `{"key":"value"}`},
		{"many params", `{"a":"1","b":"2","c":"3","d":"4"}`},
		{"nested objects", `{"config":{"nested":{"deep":"value"}}}`},
		{"arrays", `{"items":[1,2,3]}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			deps, w, r := newTestRequest(tx)
			result := ValidateConfigurationReference(w, r, deps, configID, agentID, connID+".some_action", json.RawMessage(tc.execParams))
			if result == nil {
				t.Errorf("expected wildcard config to accept %s, got error: %s", tc.name, w.Body.String())
			}
		})
	}
}
