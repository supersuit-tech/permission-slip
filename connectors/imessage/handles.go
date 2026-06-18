package imessage

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/supersuit-tech/permission-slip/connectors"
)

const (
	handleTypePhone = "phone"
	handleTypeEmail = "email"
)

// e164Pattern matches phone numbers in E.164 format.
var e164Pattern = regexp.MustCompile(`^\+[1-9]\d{1,14}$`)

// Handle is a normalized iMessage recipient or sender identity.
type Handle struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// NormalizeHandle canonicalizes a handle for reliable permission matching.
// Phones are normalized to E.164; emails are lowercased.
func NormalizeHandle(h Handle) (Handle, error) {
	typ := strings.TrimSpace(strings.ToLower(h.Type))
	value := strings.TrimSpace(h.Value)
	if typ == "" || value == "" {
		return Handle{}, &connectors.ValidationError{Message: "handle type and value are required"}
	}
	switch typ {
	case handleTypePhone:
		value = normalizePhoneValue(value)
		if !e164Pattern.MatchString(value) {
			return Handle{}, &connectors.ValidationError{
				Message: fmt.Sprintf("phone handle must be E.164 (e.g. +15551234567), got %q", h.Value),
			}
		}
	case handleTypeEmail:
		value = strings.ToLower(value)
		if !strings.Contains(value, "@") {
			return Handle{}, &connectors.ValidationError{
				Message: fmt.Sprintf("email handle must contain @, got %q", h.Value),
			}
		}
	default:
		return Handle{}, &connectors.ValidationError{
			Message: fmt.Sprintf("handle type must be %q or %q, got %q", handleTypePhone, handleTypeEmail, h.Type),
		}
	}
	return Handle{Type: typ, Value: value}, nil
}

func normalizePhoneValue(value string) string {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "+") {
		return value
	}
	digits := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, value)
	if digits == "" {
		return value
	}
	if strings.HasPrefix(digits, "1") && len(digits) == 11 {
		return "+" + digits
	}
	if len(digits) == 10 {
		return "+1" + digits
	}
	return "+" + digits
}

// NormalizeHandles canonicalizes a slice of handles, preserving order.
func NormalizeHandles(handles []Handle) ([]Handle, error) {
	if len(handles) == 0 {
		return nil, &connectors.ValidationError{Message: "at least one handle is required"}
	}
	out := make([]Handle, 0, len(handles))
	seen := make(map[string]struct{}, len(handles))
	for _, h := range handles {
		normalized, err := NormalizeHandle(h)
		if err != nil {
			return nil, err
		}
		key := normalized.Type + ":" + normalized.Value
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, normalized)
	}
	return out, nil
}

// InferHandleType guesses phone vs email from a raw imsg handle string.
func InferHandleType(raw string) string {
	raw = strings.TrimSpace(raw)
	if strings.Contains(raw, "@") {
		return handleTypeEmail
	}
	return handleTypePhone
}

// HandleFromRaw builds a normalized handle from an imsg sender/participant string.
func HandleFromRaw(raw string) (Handle, error) {
	typ := InferHandleType(raw)
	return NormalizeHandle(Handle{Type: typ, Value: raw})
}

// HandlesFromRaws normalizes a list of raw handle strings.
func HandlesFromRaws(raws []string) ([]Handle, error) {
	handles := make([]Handle, 0, len(raws))
	for _, raw := range raws {
		if strings.TrimSpace(raw) == "" {
			continue
		}
		h, err := HandleFromRaw(raw)
		if err != nil {
			return nil, err
		}
		handles = append(handles, h)
	}
	return handles, nil
}
