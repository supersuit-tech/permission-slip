package api

import (
	"encoding/json"
	"testing"
)

func TestDeriveStandingApprovalNameFromRequest_MetaFrom(t *testing.T) {
	t.Parallel()
	name := deriveStandingApprovalNameFromRequest("Read Email", json.RawMessage(`{
		"message_id":"*",
		"folder":"*",
		"$meta":{"from":"automated@airbnb.com"}
	}`))
	if name != "Read Email — from automated@airbnb.com" {
		t.Fatalf("name = %q", name)
	}
}

func TestDeriveStandingApprovalNameFromRequest_ParamConstraint(t *testing.T) {
	t.Parallel()
	name := deriveStandingApprovalNameFromRequest("Send Email", json.RawMessage(`{"to":"*@example.com"}`))
	if name != "Send Email — to: *@example.com" {
		t.Fatalf("name = %q", name)
	}
}

func TestDeriveStandingApprovalNameFromRequest_AllWildcard(t *testing.T) {
	t.Parallel()
	name := deriveStandingApprovalNameFromRequest("Read Email", json.RawMessage(`{"message_id":"*","folder":"*"}`))
	if name != "Read Email auto-approve" {
		t.Fatalf("name = %q", name)
	}
}

func TestDeriveStandingApprovalNameFromRequest_TruncatesLongName(t *testing.T) {
	t.Parallel()
	long := make([]byte, 300)
	for i := range long {
		long[i] = 'a'
	}
	name := deriveStandingApprovalNameFromRequest("Read Email", json.RawMessage(`{"to":"`+string(long)+`"}`))
	if len(name) > maxStandingApprovalNameLength {
		t.Fatalf("name length = %d, want <= %d", len(name), maxStandingApprovalNameLength)
	}
}
