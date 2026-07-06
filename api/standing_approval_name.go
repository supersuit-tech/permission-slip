package api

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/supersuit-tech/permission-slip/db"
)

const standingApprovalAutoRuleDescription = "Standing auto-approve rule"

// deriveStandingApprovalNameFromRequest builds a human-readable name when a
// standing approval request is approved, using the action display name and the
// most meaningful non-wildcard constraint values.
func deriveStandingApprovalNameFromRequest(actionName string, constraints json.RawMessage) string {
	actionName = strings.TrimSpace(actionName)
	if actionName == "" {
		actionName = "Action"
	}

	summary := summarizeStandingApprovalConstraints(constraints)
	if summary == "" {
		return truncateStandingApprovalName(fmt.Sprintf("%s auto-approve", actionName))
	}
	return truncateStandingApprovalName(fmt.Sprintf("%s — %s", actionName, summary))
}

func summarizeStandingApprovalConstraints(constraints json.RawMessage) string {
	if len(constraints) == 0 {
		return ""
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(constraints, &obj); err != nil || len(obj) == 0 {
		return ""
	}

	if metaRaw, ok := obj[db.MetaNamespaceKey]; ok {
		if summary := summarizeMetaConstraints(metaRaw); summary != "" {
			return summary
		}
	}

	parts := make([]string, 0, len(obj))
	for key, raw := range obj {
		if key == db.MetaNamespaceKey || key == db.DataWindowNamespaceKey {
			continue
		}
		if val := constraintDisplayValue(raw); val != "" && val != "*" {
			parts = append(parts, fmt.Sprintf("%s: %s", key, val))
		}
	}
	if len(parts) == 0 {
		return ""
	}
	if len(parts) == 1 {
		return parts[0]
	}
	return strings.Join(parts[:min(2, len(parts))], ", ")
}

func summarizeMetaConstraints(metaRaw json.RawMessage) string {
	var meta map[string]json.RawMessage
	if err := json.Unmarshal(metaRaw, &meta); err != nil || len(meta) == 0 {
		return ""
	}

	priority := []struct {
		key   string
		label string
	}{
		{"from", "from"},
		{"sender", "from"},
		{"senders", "from"},
		{"to", "to"},
		{"cc", "cc"},
		{"bcc", "bcc"},
	}
	for _, item := range priority {
		raw, ok := meta[item.key]
		if !ok {
			continue
		}
		if val := constraintDisplayValue(raw); val != "" && val != "*" {
			return fmt.Sprintf("%s %s", item.label, val)
		}
	}

	for key, raw := range meta {
		if val := constraintDisplayValue(raw); val != "" && val != "*" {
			return fmt.Sprintf("%s: %s", key, val)
		}
	}
	return ""
}

func constraintDisplayValue(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return strings.TrimSpace(s)
	}
	var pattern map[string]json.RawMessage
	if err := json.Unmarshal(raw, &pattern); err == nil {
		if p, ok := pattern[db.PatternKey]; ok {
			return constraintDisplayValue(p)
		}
	}
	return ""
}

func truncateStandingApprovalName(name string) string {
	if len(name) <= maxStandingApprovalNameLength {
		return name
	}
	return name[:maxStandingApprovalNameLength-1] + "…"
}
