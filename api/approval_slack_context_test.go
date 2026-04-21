package api

import (
	"encoding/json"
	"testing"
)

func TestMergeSlackContextFromResourceDetailsIntoContext(t *testing.T) {
	ctxIn := []byte(`{"description":"post","details":{"foo":"bar"}}`)
	rd, _ := json.Marshal(map[string]any{
		"slack_context": map[string]any{
			"context_scope": "recent_channel",
			"channel": map[string]any{
				"id": "C1",
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
