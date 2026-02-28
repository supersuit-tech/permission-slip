package api

import (
	"errors"
	"strings"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/db"
)

func TestConnectorIDFromActionType(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input string
		want  *string
	}{
		{"github.create_issue", strPtr("github")},
		{"slack.send_message", strPtr("slack")},
		{"email.send", strPtr("email")},
		{"connector.deeply.nested.action", strPtr("connector")},
		{"malformed_type", nil},
		{".missing_prefix", nil},
		{"", nil},
		{"trailing.", strPtr("trailing")},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := connectorIDFromActionType(tt.input)
			if tt.want == nil {
				if got != nil {
					t.Errorf("connectorIDFromActionType(%q) = %q, want nil", tt.input, *got)
				}
			} else {
				if got == nil {
					t.Errorf("connectorIDFromActionType(%q) = nil, want %q", tt.input, *tt.want)
				} else if *got != *tt.want {
					t.Errorf("connectorIDFromActionType(%q) = %q, want %q", tt.input, *got, *tt.want)
				}
			}
		})
	}
}

func TestActionTypeFromJSON(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input []byte
		want  string
	}{
		{"valid", []byte(`{"type":"email.send","version":"1"}`), "email.send"},
		{"type only", []byte(`{"type":"github.create_issue"}`), "github.create_issue"},
		{"empty type", []byte(`{"type":""}`), ""},
		{"no type field", []byte(`{"action":"something"}`), ""},
		{"empty json", []byte(`{}`), ""},
		{"nil input", nil, ""},
		{"empty bytes", []byte{}, ""},
		{"invalid json", []byte(`not json`), ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := actionTypeFromJSON(tt.input); got != tt.want {
				t.Errorf("actionTypeFromJSON(%s) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestRedactActionToType(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		input   []byte
		want    string
		wantNil bool
	}{
		{"valid", []byte(`{"type":"email.send","parameters":{"to":"alice@example.com"}}`), `{"type":"email.send"}`, false},
		{"type only", []byte(`{"type":"github.create_issue"}`), `{"type":"github.create_issue"}`, false},
		{"empty type", []byte(`{"type":""}`), "", true},
		{"no type", []byte(`{"action":"something"}`), "", true},
		{"nil", nil, "", true},
		{"empty", []byte{}, "", true},
		{"invalid json", []byte(`not json`), "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := redactActionToType(tt.input)
			if tt.wantNil {
				if got != nil {
					t.Errorf("redactActionToType(%s) = %s, want nil", tt.input, got)
				}
			} else {
				if string(got) != tt.want {
					t.Errorf("redactActionToType(%s) = %s, want %s", tt.input, got, tt.want)
				}
			}
		})
	}
}

func TestResolveExecResult(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		status, errMsg := resolveExecResult(nil)
		if status != db.ExecStatusSuccess {
			t.Errorf("status = %q, want %q", status, db.ExecStatusSuccess)
		}
		if errMsg != nil {
			t.Errorf("errMsg = %q, want nil", *errMsg)
		}
	})

	t.Run("failure", func(t *testing.T) {
		status, errMsg := resolveExecResult(errors.New("connection refused"))
		if status != db.ExecStatusFailure {
			t.Errorf("status = %q, want %q", status, db.ExecStatusFailure)
		}
		if errMsg == nil {
			t.Fatal("errMsg = nil, want non-nil")
		}
		if *errMsg != "connection refused" {
			t.Errorf("errMsg = %q, want %q", *errMsg, "connection refused")
		}
	})

	t.Run("truncation", func(t *testing.T) {
		longErr := errors.New(strings.Repeat("x", 1000))
		status, errMsg := resolveExecResult(longErr)
		if status != db.ExecStatusFailure {
			t.Errorf("status = %q, want %q", status, db.ExecStatusFailure)
		}
		if errMsg == nil {
			t.Fatal("errMsg = nil, want non-nil")
		}
		if len(*errMsg) != 512 {
			t.Errorf("len(errMsg) = %d, want 512", len(*errMsg))
		}
	})
}

func strPtr(s string) *string { return &s }
