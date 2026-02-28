package connectors

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// writeMockConnector creates a shell script that acts as a mock connector binary.
// It reads JSON from stdin and responds based on the command field.
func writeMockConnector(t *testing.T, dir string, script string) string {
	t.Helper()
	binPath := filepath.Join(dir, "connector")
	if err := os.WriteFile(binPath, []byte(script), 0o755); err != nil {
		t.Fatalf("writing mock connector: %v", err)
	}
	return binPath
}

func testManifest() *ConnectorManifest {
	return &ConnectorManifest{
		ID:   "mock",
		Name: "Mock Connector",
		Actions: []ManifestAction{
			{
				ActionType: "mock.do_thing",
				Name:       "Do Thing",
				RiskLevel:  "low",
			},
		},
		RequiredCredentials: []ManifestCredential{
			{Service: "mock", AuthType: "api_key"},
		},
	}
}

func TestExternalConnector_ID(t *testing.T) {
	c := NewExternalConnector(testManifest(), "/nonexistent")
	if got := c.ID(); got != "mock" {
		t.Errorf("ID() = %q, want %q", got, "mock")
	}
}

func TestExternalConnector_Actions(t *testing.T) {
	c := NewExternalConnector(testManifest(), "/nonexistent")
	actions := c.Actions()
	if _, ok := actions["mock.do_thing"]; !ok {
		t.Error("expected action mock.do_thing to be registered")
	}
	if len(actions) != 1 {
		t.Errorf("len(actions) = %d, want 1", len(actions))
	}
}

func TestExternalConnector_ExecuteSuccess(t *testing.T) {
	dir := t.TempDir()
	script := `#!/bin/sh
cat <<'EOF'
{"success": true, "data": {"result": "ok"}}
EOF
`
	binPath := writeMockConnector(t, dir, script)

	c := NewExternalConnector(testManifest(), binPath)
	action, ok := c.Actions()["mock.do_thing"]
	if !ok {
		t.Fatal("action not found")
	}

	result, err := action.Execute(context.Background(), ActionRequest{
		ActionType:  "mock.do_thing",
		Parameters:  json.RawMessage(`{"key": "value"}`),
		Credentials: NewCredentials(map[string]string{"api_key": "test-key"}),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]string
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("invalid result data: %v", err)
	}
	if data["result"] != "ok" {
		t.Errorf("result = %q, want %q", data["result"], "ok")
	}
}

func TestExternalConnector_ExecuteError(t *testing.T) {
	dir := t.TempDir()
	script := `#!/bin/sh
cat <<'EOF'
{"success": false, "error": {"type": "validation", "message": "missing required field"}}
EOF
`
	binPath := writeMockConnector(t, dir, script)

	c := NewExternalConnector(testManifest(), binPath)
	action, ok := c.Actions()["mock.do_thing"]
	if !ok {
		t.Fatal("action not found")
	}

	_, err := action.Execute(context.Background(), ActionRequest{
		ActionType:  "mock.do_thing",
		Parameters:  json.RawMessage(`{}`),
		Credentials: NewCredentials(map[string]string{"api_key": "test-key"}),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestExternalConnector_ExecuteAuthError(t *testing.T) {
	dir := t.TempDir()
	script := `#!/bin/sh
cat <<'EOF'
{"success": false, "error": {"type": "auth", "message": "invalid token"}}
EOF
`
	binPath := writeMockConnector(t, dir, script)

	c := NewExternalConnector(testManifest(), binPath)
	action := c.Actions()["mock.do_thing"]

	_, err := action.Execute(context.Background(), ActionRequest{
		ActionType:  "mock.do_thing",
		Credentials: NewCredentials(map[string]string{"api_key": "bad"}),
	})
	if !IsAuthError(err) {
		t.Errorf("expected AuthError, got %T: %v", err, err)
	}
}

func TestExternalConnector_ProcessFailure(t *testing.T) {
	dir := t.TempDir()
	script := `#!/bin/sh
echo "something went wrong" >&2
exit 1
`
	binPath := writeMockConnector(t, dir, script)

	c := NewExternalConnector(testManifest(), binPath)
	action := c.Actions()["mock.do_thing"]

	_, err := action.Execute(context.Background(), ActionRequest{
		ActionType:  "mock.do_thing",
		Credentials: NewCredentials(map[string]string{}),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsExternalError(err) {
		t.Errorf("expected ExternalError, got %T: %v", err, err)
	}
}

func TestExternalConnector_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	script := `#!/bin/sh
echo "not json"
`
	binPath := writeMockConnector(t, dir, script)

	c := NewExternalConnector(testManifest(), binPath)
	action := c.Actions()["mock.do_thing"]

	_, err := action.Execute(context.Background(), ActionRequest{
		ActionType:  "mock.do_thing",
		Credentials: NewCredentials(map[string]string{}),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsExternalError(err) {
		t.Errorf("expected ExternalError, got %T: %v", err, err)
	}
}

func TestExternalConnector_ValidateCredentials(t *testing.T) {
	dir := t.TempDir()
	script := `#!/bin/sh
cat <<'EOF'
{"success": true}
EOF
`
	binPath := writeMockConnector(t, dir, script)

	c := NewExternalConnector(testManifest(), binPath)
	err := c.ValidateCredentials(context.Background(), NewCredentials(map[string]string{"api_key": "test"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExternalConnector_ValidateCredentialsFailure(t *testing.T) {
	dir := t.TempDir()
	script := `#!/bin/sh
cat <<'EOF'
{"success": false, "error": {"type": "validation", "message": "api_key is required"}}
EOF
`
	binPath := writeMockConnector(t, dir, script)

	c := NewExternalConnector(testManifest(), binPath)
	err := c.ValidateCredentials(context.Background(), NewCredentials(map[string]string{}))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestExternalConnector_PassesInputToStdin(t *testing.T) {
	dir := t.TempDir()
	// This script reads stdin and echoes back the action_type as proof it received the input.
	// Uses sed instead of python3 to avoid an external runtime dependency in CI.
	script := `#!/bin/sh
INPUT=$(cat)
ACTION=$(printf '%s\n' "$INPUT" | sed -n 's/.*"action_type"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p')
printf '{"success": true, "data": {"received_action": "%s"}}' "$ACTION"
`
	binPath := writeMockConnector(t, dir, script)

	c := NewExternalConnector(testManifest(), binPath)
	action := c.Actions()["mock.do_thing"]

	result, err := action.Execute(context.Background(), ActionRequest{
		ActionType:  "mock.do_thing",
		Parameters:  json.RawMessage(`{}`),
		Credentials: NewCredentials(map[string]string{}),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]string
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("invalid result data: %v", err)
	}
	if data["received_action"] != "mock.do_thing" {
		t.Errorf("received_action = %q, want %q", data["received_action"], "mock.do_thing")
	}
}

func TestExternalConnector_ErrorWithNoDetails(t *testing.T) {
	dir := t.TempDir()
	script := `#!/bin/sh
echo '{"success": false}'
`
	binPath := writeMockConnector(t, dir, script)

	c := NewExternalConnector(testManifest(), binPath)
	action := c.Actions()["mock.do_thing"]

	_, err := action.Execute(context.Background(), ActionRequest{
		ActionType:  "mock.do_thing",
		Credentials: NewCredentials(map[string]string{}),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsExternalError(err) {
		t.Errorf("expected ExternalError for nil error details, got %T: %v", err, err)
	}
}

func TestMapSubprocessError(t *testing.T) {
	tests := []struct {
		errType string
		checker func(error) bool
	}{
		{"validation", IsValidationError},
		{"auth", IsAuthError},
		{"rate_limit", IsRateLimitError},
		{"timeout", IsTimeoutError},
		{"external", IsExternalError},
		{"unknown_type", IsExternalError},
	}

	for _, tt := range tests {
		t.Run(tt.errType, func(t *testing.T) {
			err := mapSubprocessError(&subprocessErr{Type: tt.errType, Message: "test"})
			if !tt.checker(err) {
				t.Errorf("mapSubprocessError(%q) returned %T, expected different type", tt.errType, err)
			}
		})
	}

	t.Run("nil error", func(t *testing.T) {
		err := mapSubprocessError(nil)
		if !IsExternalError(err) {
			t.Errorf("mapSubprocessError(nil) returned %T, expected ExternalError", err)
		}
	})
}
