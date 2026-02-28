package notify

import (
	"encoding/json"
	"fmt"
)

// AgentDisplayName returns a human-readable name for an agent, falling back
// to "Agent #<id>" when the name is empty. Used across all notification
// channels to keep the fallback consistent.
func AgentDisplayName(name string, id int64) string {
	if name != "" {
		return name
	}
	return fmt.Sprintf("Agent #%d", id)
}

// SummarizeAction extracts a human-readable summary from the action JSON.
// It looks for common keys: "description", "type", or "action_type".
// Falls back to "an action" if nothing useful is found.
//
// This is shared across channels so SMS, email, and web push all produce
// consistent action summaries.
func SummarizeAction(action json.RawMessage) string {
	if len(action) == 0 {
		return "an action"
	}

	var obj map[string]json.RawMessage
	if err := json.Unmarshal(action, &obj); err != nil {
		return "an action"
	}

	// Try common keys in order of preference.
	for _, key := range []string{"description", "type", "action_type"} {
		raw, ok := obj[key]
		if !ok {
			continue
		}
		var s string
		if err := json.Unmarshal(raw, &s); err == nil && s != "" {
			return TruncateUTF8(s, 60)
		}
	}

	return "an action"
}

// TruncateUTF8 truncates s to at most maxRunes runes, appending "..." if
// truncation occurred. This is safe for multi-byte UTF-8 characters — it
// never splits a character mid-byte.
//
// Edge cases: returns "" for maxRunes <= 0, and omits the ellipsis when
// maxRunes < 4 (not enough room for content + "...").
func TruncateUTF8(s string, maxRunes int) string {
	if maxRunes <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}
	// Not enough room for even one content rune plus "..." — just
	// return the first maxRunes runes without ellipsis.
	if maxRunes < 4 {
		return string(runes[:maxRunes])
	}
	return string(runes[:maxRunes-3]) + "..."
}
