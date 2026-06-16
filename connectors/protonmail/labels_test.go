package protonmail

import (
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestResolveLabelMailbox(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{"Work", "Labels/Work"},
		{"Labels/Work", "Labels/Work"},
		{"Labels/Parent/Child", "Labels/Parent/Child"},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()
			if got := resolveLabelMailbox(tc.input); got != tc.want {
				t.Errorf("resolveLabelMailbox(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestValidateLabelParam(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		label   string
		want    string
		wantErr bool
	}{
		{name: "short name", label: "Work", want: "Labels/Work"},
		{name: "full path", label: "Labels/Work", want: "Labels/Work"},
		{name: "empty", label: "", wantErr: true},
		{name: "folder path rejected", label: "Folders/Projects", wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := validateLabelParam(tc.label)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				if !connectors.IsValidationError(err) {
					t.Errorf("expected ValidationError, got %T", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestLabelDisplayName(t *testing.T) {
	t.Parallel()
	if got := labelDisplayName("Labels/Work"); got != "Work" {
		t.Errorf("got %q, want Work", got)
	}
	if got := labelDisplayName("Labels/Parent/Child"); got != "Parent/Child" {
		t.Errorf("got %q, want Parent/Child", got)
	}
}

func TestIsLabelMailbox(t *testing.T) {
	t.Parallel()
	if !isLabelMailbox("Labels/Work") {
		t.Error("expected Labels/Work to be a label mailbox")
	}
	if isLabelMailbox("INBOX") {
		t.Error("expected INBOX not to be a label mailbox")
	}
	if isLabelMailbox("Folders/Projects") {
		t.Error("expected Folders/Projects not to be a label mailbox")
	}
}
