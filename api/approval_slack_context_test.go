package api

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestMergeSlackContextFromResourceDetailsIntoContext(t *testing.T) {
	ctxIn := []byte(`{"description":"x","details":{"foo":"bar"}}`)
	rd, _ := json.Marshal(map[string]any{
		"slack_context": map[string]any{
			"context_scope": "recent_channel",
			"channel": map[string]any{
				"id":        "C1",
				"permalink": "https://acme.slack.com/archives/C1",
			},
		},
	})
	out := mergeSlackContextFromResourceDetailsIntoContext(ctxIn, rd)
	var parsed map[string]any
	if err := json.Unmarshal(out, &parsed); err != nil {
		t.Fatal(err)
	}
	details := parsed["details"].(map[string]any)
	sc := details["slack_context"].(map[string]any)
	if sc["context_scope"] != "recent_channel" {
		t.Fatalf("slack_context: %+v", sc)
	}
}

func TestMergeSlackContextFromResourceDetailsIntoContext_SizeCap(t *testing.T) {
	long := strings.Repeat("a", 65380)
	ctxIn := []byte(`{"description":"` + long + `","details":{}}`)
	if len(ctxIn) >= 65536 {
		t.Fatalf("setup: context len %d", len(ctxIn))
	}
	sc := map[string]any{"context_scope": "metadata_only", "big": strings.Repeat("b", 500)}
	rd, _ := json.Marshal(map[string]any{"slack_context": sc})
	out := mergeSlackContextFromResourceDetailsIntoContext(ctxIn, rd)
	if len(out) > 65536 {
		t.Fatalf("merged unexpectedly long: %d", len(out))
	}
	if string(out) != string(ctxIn) {
		t.Fatal("expected merge skipped when over size cap")
	}
}
