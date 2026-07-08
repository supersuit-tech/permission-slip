package agentwake

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestBuildOpenClawDelivery_WakeNoSession(t *testing.T) {
	d, err := BuildOpenClawDelivery("http://127.0.0.1:18789/hooks", "secret", WakeRequest{
		ApprovalID: "appr_abc",
		Status:     "approved",
	})
	if err != nil {
		t.Fatal(err)
	}
	if d.URL != "http://127.0.0.1:18789/hooks/wake" {
		t.Fatalf("URL = %q", d.URL)
	}
	if d.Headers["Authorization"] != "Bearer secret" {
		t.Fatalf("Authorization = %q", d.Headers["Authorization"])
	}
	var body map[string]string
	if err := json.Unmarshal(d.Body, &body); err != nil {
		t.Fatal(err)
	}
	if body["mode"] != "now" {
		t.Fatalf("mode = %q", body["mode"])
	}
	if !strings.Contains(body["text"], "appr_abc") || !strings.Contains(body["text"], "approved") {
		t.Fatalf("text = %q", body["text"])
	}
}

func TestBuildOpenClawDelivery_AgentWithSessionKey(t *testing.T) {
	d, err := BuildOpenClawDelivery("http://100.64.0.5:18789/hooks/", "tok", WakeRequest{
		ApprovalID: "appr_x",
		Status:     "denied",
		SessionKey: "agent:main:imessage",
	})
	if err != nil {
		t.Fatal(err)
	}
	if d.URL != "http://100.64.0.5:18789/hooks/agent" {
		t.Fatalf("URL = %q", d.URL)
	}
	var body map[string]string
	if err := json.Unmarshal(d.Body, &body); err != nil {
		t.Fatal(err)
	}
	if body["wakeMode"] != "next-heartbeat" {
		t.Fatalf("wakeMode = %q", body["wakeMode"])
	}
	if body["sessionKey"] != "agent:main:imessage" {
		t.Fatalf("sessionKey = %q", body["sessionKey"])
	}
	if body["mode"] != "" {
		t.Fatalf("unexpected mode field on agent payload: %q", body["mode"])
	}
}

func TestSessionKeyFromApprovalContext(t *testing.T) {
	sk := SessionKeyFromApprovalContext([]byte(`{"session_key":"agent:main:slack","description":"x"}`))
	if sk != "agent:main:slack" {
		t.Fatalf("got %q", sk)
	}
	if SessionKeyFromApprovalContext([]byte(`{}`)) != "" {
		t.Fatal("expected empty")
	}
}

func TestWakeDeliveryFromStoredApprovalContext(t *testing.T) {
	ctxJSON := []byte(`{"description":"Send email","session_key":"agent:main:telegram:direct:8935627010"}`)
	sk := SessionKeyFromApprovalContext(ctxJSON)
	if sk != "agent:main:telegram:direct:8935627010" {
		t.Fatalf("session key = %q", sk)
	}
	d, err := BuildOpenClawDelivery("http://127.0.0.1:18789/hooks", "secret", WakeRequest{
		ApprovalID: "appr_ctx",
		Status:     "approved",
		SessionKey: sk,
	})
	if err != nil {
		t.Fatal(err)
	}
	if d.URL != "http://127.0.0.1:18789/hooks/agent" {
		t.Fatalf("URL = %q", d.URL)
	}
	var body map[string]string
	if err := json.Unmarshal(d.Body, &body); err != nil {
		t.Fatal(err)
	}
	if body["sessionKey"] != "agent:main:telegram:direct:8935627010" {
		t.Fatalf("sessionKey = %q", body["sessionKey"])
	}
	if body["wakeMode"] != "next-heartbeat" {
		t.Fatalf("wakeMode = %q", body["wakeMode"])
	}
}

func TestWakeMessage_Expired(t *testing.T) {
	msg := WakeMessage("appr_1", "expired")
	if !strings.Contains(msg, "expired unanswered") {
		t.Fatalf("msg = %q", msg)
	}
}
