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

// PushContent holds the display content for a push notification. Both the
// web push and mobile push channels use this to build their transport-specific
// payloads — keeping content generation in one place.
type PushContent struct {
	Title      string
	Body       string
	URL        string
	ApprovalID string
}

// BuildPushContent constructs the push notification display content from
// approval data. Used by both webpush and mobilepush senders to ensure
// consistent messaging across push channels.
func BuildPushContent(approval Approval) PushContent {
	if approval.Type == NotificationTypePaymentFailed {
		return PushContent{
			Title:      "Payment Failed",
			Body:       "Your subscription payment could not be processed. Update your payment method to avoid losing access.",
			URL:        approval.ApprovalURL,
			ApprovalID: approval.ApprovalID,
		}
	}

	if approval.Type == NotificationTypeCardExpiring {
		info := extractCardExpiringInfo(approval.Context)
		title := "Card Expiring Soon"
		body := fmt.Sprintf("Your %s card ending in %s expires %s. Add a replacement to avoid disruptions.",
			info.Brand, info.Last4, formatCardExpiry(info.ExpMonth, info.ExpYear))
		if info.Expired {
			title = "Card Expired"
			body = fmt.Sprintf("Your %s card ending in %s has expired. Update your payment methods.",
				info.Brand, info.Last4)
		}
		return PushContent{
			Title:      title,
			Body:       body,
			URL:        approval.ApprovalURL,
			ApprovalID: approval.ApprovalID,
		}
	}

	title := "Approval Request"
	if approval.AgentName != "" {
		title = approval.AgentName
	}

	body := "Action requires your approval"
	if len(approval.Action) > 0 {
		var action struct {
			Type    string `json:"type"`
			Summary string `json:"summary"`
		}
		if json.Unmarshal(approval.Action, &action) == nil && action.Summary != "" {
			body = TruncateUTF8(action.Summary, 200)
		} else if action.Type != "" {
			body = action.Type
		}
	}

	return PushContent{
		Title:      title,
		Body:       body,
		URL:        approval.ApprovalURL,
		ApprovalID: approval.ApprovalID,
	}
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
