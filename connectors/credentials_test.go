package connectors

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"testing"
)

func TestCredentials_Get(t *testing.T) {
	t.Parallel()
	creds := NewCredentials(map[string]string{
		"api_key":    "sk-secret-123",
		"api_secret": "super-secret",
	})

	val, ok := creds.Get("api_key")
	if !ok {
		t.Fatal("expected to find 'api_key'")
	}
	if val != "sk-secret-123" {
		t.Errorf("expected 'sk-secret-123', got %q", val)
	}

	_, ok = creds.Get("nonexistent")
	if ok {
		t.Error("expected ok=false for missing key")
	}
}

func TestCredentials_GetFromZeroValue(t *testing.T) {
	t.Parallel()
	var creds Credentials

	_, ok := creds.Get("anything")
	if ok {
		t.Error("expected ok=false for zero-value Credentials")
	}
}

func TestCredentials_Keys(t *testing.T) {
	t.Parallel()
	creds := NewCredentials(map[string]string{
		"api_key":    "secret1",
		"api_secret": "secret2",
	})

	keys := creds.Keys()
	sort.Strings(keys)

	if len(keys) != 2 {
		t.Fatalf("expected 2 keys, got %d", len(keys))
	}
	if keys[0] != "api_key" || keys[1] != "api_secret" {
		t.Errorf("unexpected keys: %v", keys)
	}
}

func TestCredentials_StringRedacts(t *testing.T) {
	t.Parallel()
	creds := NewCredentials(map[string]string{
		"api_key": "sk-secret-123",
	})

	s := creds.String()
	if s == "" {
		t.Fatal("expected non-empty string")
	}
	if strings.Contains(s, "sk-secret-123") {
		t.Errorf("String() must not contain credential values, got: %s", s)
	}
	if !strings.Contains(s, "api_key") {
		t.Errorf("String() should contain key names, got: %s", s)
	}
}

func TestCredentials_GoStringRedacts(t *testing.T) {
	t.Parallel()
	creds := NewCredentials(map[string]string{
		"token": "bearer-xyz",
	})

	s := fmt.Sprintf("%#v", creds)
	if strings.Contains(s, "bearer-xyz") {
		t.Errorf("GoString() must not contain credential values, got: %s", s)
	}
}

func TestCredentials_SprintfRedacts(t *testing.T) {
	t.Parallel()
	creds := NewCredentials(map[string]string{
		"password": "hunter2",
	})

	// %v (via String()) and %s formatting must both redact credential values
	for _, format := range []string{"%v", "%s"} {
		s := fmt.Sprintf(format, creds)
		if strings.Contains(s, "hunter2") {
			t.Errorf("fmt.Sprintf(%q) must not contain credential values, got: %s", format, s)
		}
	}
}

func TestCredentials_MarshalJSONRedacts(t *testing.T) {
	t.Parallel()
	creds := NewCredentials(map[string]string{
		"api_key": "sk-secret-123",
	})

	data, err := json.Marshal(creds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.Contains(string(data), "sk-secret-123") {
		t.Errorf("MarshalJSON must not contain credential values, got: %s", data)
	}
	if string(data) != `"[REDACTED]"` {
		t.Errorf("expected '\"[REDACTED]\"', got: %s", data)
	}
}

func TestCredentials_MarshalJSONInStruct(t *testing.T) {
	t.Parallel()
	req := ActionRequest{
		ActionType:  "github.create_issue",
		Credentials: NewCredentials(map[string]string{"token": "ghp_secret"}),
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.Contains(string(data), "ghp_secret") {
		t.Errorf("marshaling ActionRequest must not expose credentials, got: %s", data)
	}
}

func TestCredentials_DefensiveCopy(t *testing.T) {
	t.Parallel()
	original := map[string]string{
		"api_key": "original-value",
	}
	creds := NewCredentials(original)

	// Mutate the original map after construction
	original["api_key"] = "mutated-value"
	original["injected"] = "new-key"

	val, ok := creds.Get("api_key")
	if !ok {
		t.Fatal("expected to find 'api_key'")
	}
	if val != "original-value" {
		t.Errorf("expected 'original-value', got %q — caller mutation leaked in", val)
	}

	_, ok = creds.Get("injected")
	if ok {
		t.Error("expected ok=false for key injected after construction")
	}
}
