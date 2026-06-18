package imessage

import (
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestSendMessageParams_Validate(t *testing.T) {
	t.Parallel()
	p := sendMessageParams{
		To:      []Handle{{Type: "phone", Value: "+15551234567"}},
		Text:    "hello",
		Service: "auto",
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
		To:      []Handle{{Type: "phone", Value: "+15551234567"}},
		Text:    "hello",
		Service: "auto",
	})
	if params["to"] != "+15551234567" {
		t.Fatalf("to = %v", params["to"])
	}
	if params["service"] != "auto" {
		t.Fatalf("service = %v", params["service"])
	}
	if _, ok := params["no_sms_fallback"]; ok {
		t.Fatalf("no_sms_fallback should be omitted by default")
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

func TestSendMessageParams_DefaultServiceAuto(t *testing.T) {
	t.Parallel()
	p := sendMessageParams{
		To:   []Handle{{Type: "phone", Value: "+15551234567"}},
		Text: "hi",
	}
	if err := p.validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}
	if p.Service != "auto" {
		t.Fatalf("service = %q", p.Service)
	}
}
