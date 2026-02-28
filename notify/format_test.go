package notify

import (
	"encoding/json"
	"strings"
	"testing"
	"unicode/utf8"
)

// ── SummarizeAction tests ───────────────────────────────────────────────────

func TestSummarizeAction_WithDescription(t *testing.T) {
	t.Parallel()
	action := json.RawMessage(`{"description":"Send welcome email","type":"email.send"}`)
	result := SummarizeAction(action)
	if result != "Send welcome email" {
		t.Errorf("expected description, got: %s", result)
	}
}

func TestSummarizeAction_WithType(t *testing.T) {
	t.Parallel()
	action := json.RawMessage(`{"type":"email.send"}`)
	result := SummarizeAction(action)
	if result != "email.send" {
		t.Errorf("expected type, got: %s", result)
	}
}

func TestSummarizeAction_WithActionType(t *testing.T) {
	t.Parallel()
	action := json.RawMessage(`{"action_type":"slack.post_message"}`)
	result := SummarizeAction(action)
	if result != "slack.post_message" {
		t.Errorf("expected action_type, got: %s", result)
	}
}

func TestSummarizeAction_EmptyJSON(t *testing.T) {
	t.Parallel()
	result := SummarizeAction(json.RawMessage(`{}`))
	if result != "an action" {
		t.Errorf("expected fallback, got: %s", result)
	}
}

func TestSummarizeAction_NilAction(t *testing.T) {
	t.Parallel()
	result := SummarizeAction(nil)
	if result != "an action" {
		t.Errorf("expected fallback, got: %s", result)
	}
}

func TestSummarizeAction_LongDescription(t *testing.T) {
	t.Parallel()
	long := strings.Repeat("a", 100)
	action := json.RawMessage(`{"description":"` + long + `"}`)
	result := SummarizeAction(action)
	if len([]rune(result)) > 60 {
		t.Errorf("expected truncation to 60 runes, got %d runes: %s", len([]rune(result)), result)
	}
	if !strings.HasSuffix(result, "...") {
		t.Errorf("expected truncated string to end with '...', got: %s", result)
	}
}

func TestSummarizeAction_InvalidJSON(t *testing.T) {
	t.Parallel()
	result := SummarizeAction(json.RawMessage(`not json`))
	if result != "an action" {
		t.Errorf("expected fallback for invalid JSON, got: %s", result)
	}
}

// ── TruncateUTF8 tests ─────────────────────────────────────────────────────

func TestTruncateUTF8_ShortString(t *testing.T) {
	t.Parallel()
	result := TruncateUTF8("hello", 60)
	if result != "hello" {
		t.Errorf("expected no truncation, got: %s", result)
	}
}

func TestTruncateUTF8_ExactLimit(t *testing.T) {
	t.Parallel()
	s := strings.Repeat("a", 60)
	result := TruncateUTF8(s, 60)
	if result != s {
		t.Errorf("expected no truncation at exact limit, got length %d", len(result))
	}
}

func TestTruncateUTF8_OverLimit(t *testing.T) {
	t.Parallel()
	s := strings.Repeat("a", 100)
	result := TruncateUTF8(s, 60)
	if len([]rune(result)) != 60 {
		t.Errorf("expected 60 runes, got %d", len([]rune(result)))
	}
	if !strings.HasSuffix(result, "...") {
		t.Errorf("expected '...' suffix, got: %s", result)
	}
}

func TestTruncateUTF8_MultiByte(t *testing.T) {
	t.Parallel()
	// 70 CJK characters — each is 3 bytes in UTF-8. Byte-based truncation
	// at position 57 would split a character. Rune-based must not.
	s := strings.Repeat("日", 70)
	result := TruncateUTF8(s, 60)

	if !utf8.ValidString(result) {
		t.Fatalf("result is not valid UTF-8: %q", result)
	}
	if len([]rune(result)) != 60 {
		t.Errorf("expected 60 runes, got %d", len([]rune(result)))
	}
	if !strings.HasSuffix(result, "...") {
		t.Errorf("expected '...' suffix, got: %s", result)
	}
}

func TestTruncateUTF8_Emoji(t *testing.T) {
	t.Parallel()
	// 65 emoji (each 4 bytes in UTF-8).
	s := strings.Repeat("🔒", 65)
	result := TruncateUTF8(s, 60)

	if !utf8.ValidString(result) {
		t.Fatalf("result is not valid UTF-8: %q", result)
	}
	if len([]rune(result)) != 60 {
		t.Errorf("expected 60 runes, got %d", len([]rune(result)))
	}
}

func TestTruncateUTF8_ZeroLimit(t *testing.T) {
	t.Parallel()
	result := TruncateUTF8("hello", 0)
	if result != "" {
		t.Errorf("expected empty string for maxRunes=0, got: %q", result)
	}
}

func TestTruncateUTF8_NegativeLimit(t *testing.T) {
	t.Parallel()
	result := TruncateUTF8("hello", -1)
	if result != "" {
		t.Errorf("expected empty string for maxRunes=-1, got: %q", result)
	}
}

func TestTruncateUTF8_SmallLimit(t *testing.T) {
	t.Parallel()
	// maxRunes=2 — not enough room for content + "...", should truncate
	// without ellipsis instead of panicking.
	result := TruncateUTF8("hello world", 2)
	if result != "he" {
		t.Errorf("expected %q for maxRunes=2, got: %q", "he", result)
	}
}

func TestTruncateUTF8_LimitThree(t *testing.T) {
	t.Parallel()
	result := TruncateUTF8("hello world", 3)
	if result != "hel" {
		t.Errorf("expected %q for maxRunes=3, got: %q", "hel", result)
	}
}

func TestTruncateUTF8_LimitFour(t *testing.T) {
	t.Parallel()
	// maxRunes=4 — just enough for 1 content rune + "..."
	result := TruncateUTF8("hello world", 4)
	if result != "h..." {
		t.Errorf("expected %q for maxRunes=4, got: %q", "h...", result)
	}
}

// ── AgentDisplayName tests ──────────────────────────────────────────────────

func TestAgentDisplayName_WithName(t *testing.T) {
	t.Parallel()
	if AgentDisplayName("My Bot", 42) != "My Bot" {
		t.Error("expected name to be returned as-is")
	}
}

func TestAgentDisplayName_EmptyName(t *testing.T) {
	t.Parallel()
	result := AgentDisplayName("", 99)
	if result != "Agent #99" {
		t.Errorf("expected fallback, got: %s", result)
	}
}
