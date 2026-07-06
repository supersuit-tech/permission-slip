package api

import (
	"encoding/json"
	"testing"
)

func TestMergeStandingApprovalFallthroughIntoContext_MetadataUnavailable(t *testing.T) {
	t.Parallel()
	in := json.RawMessage(`{"description":"read email","details":{"folder":"INBOX"}}`)
	out := mergeStandingApprovalFallthroughIntoContext(in, standingApprovalFallthroughMetadataUnavailable)

	var parsed map[string]json.RawMessage
	if err := json.Unmarshal(out, &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	var details map[string]any
	if err := json.Unmarshal(parsed["details"], &details); err != nil {
		t.Fatalf("unmarshal details: %v", err)
	}
	ft, ok := details["standing_approval_fallthrough"].(map[string]any)
	if !ok {
		t.Fatalf("expected standing_approval_fallthrough, got %#v", details["standing_approval_fallthrough"])
	}
	if ft["reason"] != standingApprovalFallthroughMetadataUnavailable {
		t.Fatalf("reason = %#v", ft["reason"])
	}
	if ft["message"] == "" {
		t.Fatal("expected non-empty message")
	}
}

func TestMergeStandingApprovalFallthroughIntoContext_EmptyReason(t *testing.T) {
	t.Parallel()
	in := json.RawMessage(`{"description":"x"}`)
	out := mergeStandingApprovalFallthroughIntoContext(in, "")
	if string(out) != string(in) {
		t.Fatalf("expected unchanged context, got %s", out)
	}
}
