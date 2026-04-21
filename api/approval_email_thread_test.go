package api

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestMergeEmailThreadFromResourceDetailsIntoContext(t *testing.T) {
	ctxIn := []byte(`{"description":"x","details":{"foo":"bar"}}`)
	rd, _ := json.Marshal(map[string]any{
		"subject": "S",
		"email_thread": map[string]any{
			"subject": "Thr",
			"messages": []map[string]any{
				{"from": "a@b.com", "message_id": "1"},
			},
		},
	})
	out := mergeEmailThreadFromResourceDetailsIntoContext(ctxIn, rd)
	var parsed map[string]any
	if err := json.Unmarshal(out, &parsed); err != nil {
		t.Fatal(err)
	}
	details := parsed["details"].(map[string]any)
	et := details["email_thread"].(map[string]any)
	if et["subject"] != "Thr" {
		t.Fatalf("email_thread: %+v", et)
	}
}

func TestMergeEmailThreadFromResourceDetailsIntoContext_SizeCap(t *testing.T) {
	// Context stays under 65536; merged output would exceed the cap — merge must no-op.
	long := strings.Repeat("a", 65380)
	ctxIn := []byte(`{"description":"` + long + `","details":{}}`)
	if len(ctxIn) >= 65536 {
		t.Fatalf("setup: context len %d", len(ctxIn))
	}
	thread := connectors.EmailThreadPayload{
		Subject: "x",
		Messages: []connectors.EmailThreadMessage{
			{From: "a", MessageID: "1", BodyText: strings.Repeat("b", 400)},
		},
	}
	b, _ := json.Marshal(map[string]any{"email_thread": thread})
	out := mergeEmailThreadFromResourceDetailsIntoContext(ctxIn, b)
	if len(out) > 65536 {
		t.Fatalf("merged unexpectedly long: %d", len(out))
	}
	if string(out) != string(ctxIn) {
		t.Fatal("expected merge skipped when over size cap")
	}
}
