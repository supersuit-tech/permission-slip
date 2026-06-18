package imessage

import (
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestNormalizeHandle_Phone(t *testing.T) {
	t.Parallel()
	h, err := NormalizeHandle(Handle{Type: "phone", Value: "+15551234567"})
	if err != nil {
		t.Fatalf("NormalizeHandle: %v", err)
	}
	if h.Value != "+15551234567" {
		t.Fatalf("got %q", h.Value)
	}
}

func TestNormalizeHandle_PhoneWithoutPlus(t *testing.T) {
	t.Parallel()
	h, err := NormalizeHandle(Handle{Type: "phone", Value: "5551234567"})
	if err != nil {
		t.Fatalf("NormalizeHandle: %v", err)
	}
	if h.Value != "+15551234567" {
		t.Fatalf("got %q", h.Value)
	}
}

func TestNormalizeHandle_Email(t *testing.T) {
	t.Parallel()
	h, err := NormalizeHandle(Handle{Type: "email", Value: "Me@iCloud.com"})
	if err != nil {
		t.Fatalf("NormalizeHandle: %v", err)
	}
	if h.Value != "me@icloud.com" {
		t.Fatalf("got %q", h.Value)
	}
}

func TestNormalizeHandle_InvalidType(t *testing.T) {
	t.Parallel()
	_, err := NormalizeHandle(Handle{Type: "fax", Value: "x"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !connectors.IsValidationError(err) {
		t.Fatalf("expected ValidationError, got %T", err)
	}
}

func TestInferHandleType(t *testing.T) {
	t.Parallel()
	if got := InferHandleType("a@b.com"); got != handleTypeEmail {
		t.Fatalf("got %q", got)
	}
	if got := InferHandleType("+1555"); got != handleTypePhone {
		t.Fatalf("got %q", got)
	}
}

func TestHandlesFromRaws_Dedupes(t *testing.T) {
	t.Parallel()
	handles, err := HandlesFromRaws([]string{"+15551234567", "+15551234567"})
	if err != nil {
		t.Fatalf("HandlesFromRaws: %v", err)
	}
	if len(handles) != 1 {
		t.Fatalf("got %d handles", len(handles))
	}
}
