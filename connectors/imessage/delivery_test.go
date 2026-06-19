package imessage

import "testing"

func TestResolveDeliveryDisclosure_SMSChatAuto(t *testing.T) {
	t.Parallel()
	service, disclosure := resolveDeliveryDisclosure(sendMessageParams{Service: "auto"}, &chat{Service: "SMS"})
	if service != "sms" {
		t.Fatalf("service = %q", service)
	}
	if disclosure != "Will send as SMS via relay" {
		t.Fatalf("disclosure = %q", disclosure)
	}
}

func TestResolveDeliveryDisclosure_iMessageChatAuto(t *testing.T) {
	t.Parallel()
	service, disclosure := resolveDeliveryDisclosure(sendMessageParams{Service: "auto"}, &chat{Service: "iMessage"})
	if service != "imessage" {
		t.Fatalf("service = %q", service)
	}
	if disclosure != "Will send as iMessage" {
		t.Fatalf("disclosure = %q", disclosure)
	}
}

func TestResolveDeliveryDisclosure_NewRecipientAuto(t *testing.T) {
	t.Parallel()
	service, disclosure := resolveDeliveryDisclosure(sendMessageParams{Service: "auto"}, nil)
	if service != "auto" {
		t.Fatalf("service = %q", service)
	}
	if disclosure == "" {
		t.Fatal("expected disclosure")
	}
}

func TestResolveDeliveryDisclosure_StrictIMessage(t *testing.T) {
	t.Parallel()
	noFallback := true
	_, disclosure := resolveDeliveryDisclosure(sendMessageParams{
		Service:       "auto",
		NoSMSFallback: &noFallback,
	}, nil)
	if disclosure != "Will send as iMessage (no SMS fallback)" {
		t.Fatalf("disclosure = %q", disclosure)
	}
}

func TestBuildSendRPCParams_DefaultAutoNoFallbackFlag(t *testing.T) {
	t.Parallel()
	params := buildSendRPCParams(sendMessageParams{
		To:   []Handle{{Type: "phone", Value: "+15551234567"}},
		Text: "hello",
	})
	if params["service"] != "auto" {
		t.Fatalf("service = %v", params["service"])
	}
	if _, ok := params["no_sms_fallback"]; ok {
		t.Fatalf("no_sms_fallback should be omitted by default, got %v", params["no_sms_fallback"])
	}
}

func TestBuildSendRPCParams_OptInNoSMSFallback(t *testing.T) {
	t.Parallel()
	noFallback := true
	params := buildSendRPCParams(sendMessageParams{
		To:            []Handle{{Type: "phone", Value: "+15551234567"}},
		Text:          "hello",
		NoSMSFallback: &noFallback,
	})
	if params["no_sms_fallback"] != true {
		t.Fatalf("no_sms_fallback = %v", params["no_sms_fallback"])
	}
}

func TestIsTerminalSendState(t *testing.T) {
	t.Parallel()
	if !isTerminalSendState("delivered") {
		t.Fatal("delivered should be terminal")
	}
	if isTerminalSendState("pending") {
		t.Fatal("pending should not be terminal")
	}
}
