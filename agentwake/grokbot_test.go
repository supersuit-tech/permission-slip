package agentwake

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestValidateGrokBotWebhookURL_AllowsCursor(t *testing.T) {
	t.Parallel()
	ok := []string{
		"https://api2.cursor.sh/automations/webhook/abc123",
		"https://api2.cursor.sh/automations/webhook/abc123/",
		"https://API2.CURSOR.SH/automations/webhook/abc-def",
		"https://api2.cursor.sh:443/automations/webhook/x",
	}
	for _, raw := range ok {
		if err := ValidateGrokBotWebhookURL(raw); err != nil {
			t.Errorf("ValidateGrokBotWebhookURL(%q) = %v, want nil", raw, err)
		}
	}
}

func TestValidateGrokBotWebhookURL_Rejects(t *testing.T) {
	t.Parallel()
	cases := []string{
		"",
		"http://api2.cursor.sh/automations/webhook/abc",
		"https://evil.example/automations/webhook/abc",
		"https://api2.cursor.sh.evil.com/automations/webhook/abc",
		"https://api2.cursor.sh/hooks/wake",
		"https://api2.cursor.sh/automations/webhook/",
		"https://api2.cursor.sh/automations/webhook",
		"https://api2.cursor.sh/automations/webhook/abc/extra",
		"https://user:pass@api2.cursor.sh/automations/webhook/abc",
		"https://127.0.0.1/automations/webhook/abc",
		"http://100.64.0.5:18789/hooks",
		"https://api2.cursor.sh:8443/automations/webhook/abc",
	}
	for _, raw := range cases {
		if err := ValidateGrokBotWebhookURL(raw); err == nil {
			t.Errorf("ValidateGrokBotWebhookURL(%q) = nil, want error", raw)
		}
	}
}

func TestBuildGrokBotDelivery_PostsURLAsIs(t *testing.T) {
	t.Parallel()
	target := "https://api2.cursor.sh/automations/webhook/wh_abc"
	d, err := BuildGrokBotDelivery(target, "whsec_key", WakeRequest{
		ApprovalID: "appr_1",
		Status:     "approved",
		AgentID:    3,
	})
	if err != nil {
		t.Fatal(err)
	}
	if d.URL != target {
		t.Fatalf("URL = %q, want registered URL as-is (no /hooks/wake suffix)", d.URL)
	}
	if d.Headers["Authorization"] != "Bearer whsec_key" {
		t.Fatalf("Authorization = %q", d.Headers["Authorization"])
	}
	var body grokBotWakeBody
	if err := json.Unmarshal(d.Body, &body); err != nil {
		t.Fatal(err)
	}
	if body.Source != "permission-slip" || body.ApprovalID != "appr_1" || body.Status != "approved" || body.AgentID != 3 {
		t.Fatalf("body = %+v", body)
	}
	if !strings.Contains(body.Text, "appr_1") || !strings.Contains(body.Text, "approved") {
		t.Fatalf("text = %q", body.Text)
	}
}

func TestBuildGrokBotDelivery_PreservesAuthorizationScheme(t *testing.T) {
	t.Parallel()
	d, err := BuildGrokBotDelivery(
		"https://api2.cursor.sh/automations/webhook/wh_abc",
		"Bearer already-set",
		WakeRequest{ApprovalID: "appr_x", Status: "denied", AgentID: 1},
	)
	if err != nil {
		t.Fatal(err)
	}
	if d.Headers["Authorization"] != "Bearer already-set" {
		t.Fatalf("Authorization = %q", d.Headers["Authorization"])
	}
}

func TestBuildDelivery_DispatchesByProvider(t *testing.T) {
	t.Parallel()
	openclaw, err := BuildDelivery(ProviderOpenClaw, "http://127.0.0.1:18789/hooks", "tok", WakeRequest{
		ApprovalID: "appr_1",
		Status:     "approved",
	})
	if err != nil {
		t.Fatal(err)
	}
	if openclaw.URL != "http://127.0.0.1:18789/hooks/wake" {
		t.Fatalf("openclaw URL = %q", openclaw.URL)
	}

	grok, err := BuildDelivery(ProviderGrokBot, "https://api2.cursor.sh/automations/webhook/wh_1", "key", WakeRequest{
		ApprovalID: "appr_1",
		Status:     "cancelled",
		AgentID:    9,
	})
	if err != nil {
		t.Fatal(err)
	}
	if grok.URL != "https://api2.cursor.sh/automations/webhook/wh_1" {
		t.Fatalf("grokbot URL = %q", grok.URL)
	}
}

func TestNormalizeProvider(t *testing.T) {
	t.Parallel()
	got, err := NormalizeProvider("")
	if err != nil || got != ProviderOpenClaw {
		t.Fatalf("empty = %q %v", got, err)
	}
	got, err = NormalizeProvider("GrokBot")
	if err != nil || got != ProviderGrokBot {
		t.Fatalf("GrokBot = %q %v", got, err)
	}
	if _, err := NormalizeProvider("zapier"); err == nil {
		t.Fatal("expected error for unknown provider")
	}
}
