package connectors

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestTruncateEmailThreadBodies(t *testing.T) {
	long := strings.Repeat("a", MaxEmailThreadBodyRunes+100)
	m := EmailThreadMessage{BodyHTML: long, BodyText: "short"}
	TruncateEmailThreadBodies(&m)
	if utf8.RuneCountInString(m.BodyHTML) != MaxEmailThreadBodyRunes {
		t.Fatalf("expected html truncated to %d runes, got %d", MaxEmailThreadBodyRunes, utf8.RuneCountInString(m.BodyHTML))
	}
	if !m.Truncated {
		t.Fatal("expected truncated flag")
	}
	if m.BodyText != "short" {
		t.Errorf("short text should be unchanged, got len %d", utf8.RuneCountInString(m.BodyText))
	}
}
