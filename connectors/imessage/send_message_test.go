package imessage

import (
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestSendMessageParams_Validate(t *testing.T) {
	t.Parallel()
	p := sendMessageParams{
		To:            []Handle{{Type: "phone", Value: "+15551234567"}},
		Text:          "hello",
		Service:       "imessage",
		NoSMSFallback: boolPtr(true),
	}
	if err := p.validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}

	missingTarget := sendMessageParams{Text: "hi"}
	if err := missingTarget.validate(); err == nil {
		t.Fatal("expected error for missing target")
	}

	bothTargets := sendMessageParams{
		ChatID: 1,
		To:     []Handle{{Type: "phone", Value: "+15551234567"}},
		Text:   "hi",
	}
	if err := bothTargets.validate(); err == nil {
		t.Fatal("expected error for both chat and to")
	}

	multiTo := sendMessageParams{
		To:   []Handle{{Type: "phone", Value: "+15551234567"}, {Type: "phone", Value: "+15559876543"}},
		Text: "hi",
	}
	if err := multiTo.validate(); err == nil {
		t.Fatal("expected error for multiple to handles")
	}
}

func TestBuildSendRPCParams(t *testing.T) {
	t.Parallel()
	params := buildSendRPCParams(sendMessageParams{
		To:            []Handle{{Type: "phone", Value: "+15551234567"}},
		Text:          "hello",
		Service:       "imessage",
		NoSMSFallback: boolPtr(true),
	})
	if params["to"] != "+15551234567" {
		t.Fatalf("to = %v", params["to"])
	}
	if params["no_sms_fallback"] != true {
		t.Fatalf("no_sms_fallback = %v", params["no_sms_fallback"])
	}
}

func TestSendPreviewTarget(t *testing.T) {
	t.Parallel()
	if got := sendPreviewTarget(sendMessageParams{ChatID: 42}); got != "chat:42" {
		t.Fatalf("got %q", got)
	}
}
func TestSendMessageParams_InvalidService(t *testing.T) {
	t.Parallel()
	p := sendMessageParams{
		To:      []Handle{{Type: "phone", Value: "+15551234567"}},
		Text:    "hi",
		Service: "rcs",
	}
	err := p.validate()
	if err == nil || !connectors.IsValidationError(err) {
		t.Fatalf("got %v", err)
	}
}

func boolPtr(b bool) *bool { return &b }
