package notify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"sort"
	"strings"
	"time"
)

// standingExecutionInfo holds the fields extracted from the Context JSON
// for a standing approval execution notification.
type standingExecutionInfo struct {
	ExecutionCount int    // how many times the standing approval has been used
	MaxExecutions  int    // total allowed executions (0 = unlimited)
	AgentName      string // resolved display name
	ActionType     string // e.g. "github.issues.create"
}

// extractStandingExecutionInfo extracts execution metadata from the
// approval's Action and Context JSON. The Context is expected to contain
// "execution_count" and "max_executions" integers.
func extractStandingExecutionInfo(approval Approval) standingExecutionInfo {
	info := standingExecutionInfo{
		AgentName:  AgentDisplayName(approval.AgentName, approval.AgentID),
		ActionType: extractActionType(approval.Action),
	}

	if len(approval.Context) == 0 {
		return info
	}

	var ctx map[string]json.RawMessage
	if json.Unmarshal(approval.Context, &ctx) != nil {
		return info
	}

	if raw, ok := ctx["execution_count"]; ok {
		var n int
		if json.Unmarshal(raw, &n) == nil {
			info.ExecutionCount = n
		}
	}
	if raw, ok := ctx["max_executions"]; ok {
		var n int
		if json.Unmarshal(raw, &n) == nil {
			info.MaxExecutions = n
		}
	}

	return info
}

// executionCountLabel returns a human-readable execution count string
// like "3 of 10" or "3" (when unlimited).
func (info standingExecutionInfo) executionCountLabel() string {
	if info.MaxExecutions > 0 {
		return fmt.Sprintf("%d of %d", info.ExecutionCount, info.MaxExecutions)
	}
	if info.ExecutionCount > 0 {
		return fmt.Sprintf("%d", info.ExecutionCount)
	}
	return ""
}

// buildStandingExecutionPushContent constructs push notification content for
// standing approval executions. Used by both web push and mobile push senders.
func buildStandingExecutionPushContent(approval Approval) PushContent {
	info := extractStandingExecutionInfo(approval)

	title := fmt.Sprintf("%s auto-executed", info.AgentName)

	body := info.ActionType
	if body == "" {
		body = "an action"
	}
	if info.ExecutionCount > 0 {
		body = fmt.Sprintf("%s (#%d)", body, info.ExecutionCount)
	}

	return PushContent{
		Title:      title,
		Body:       body,
		URL:        approval.ApprovalURL,
		ApprovalID: approval.ApprovalID,
	}
}

// sensitiveSubstrings are substrings that, when found in a lowercased
// parameter key, cause the value to be redacted. Substring matching is
// safer than exact-match because it catches compound keys like
// "aws_secret_access_key", "db_password", "oauth_token", etc.
var sensitiveSubstrings = []string{
	"secret", "password", "passwd", "token", "credential",
	"key", "auth", "private",
}

// isSensitiveKey returns true if the parameter key looks like it holds
// a secret value. Uses substring matching on the lowercased key.
func isSensitiveKey(key string) bool {
	lower := strings.ToLower(key)
	for _, sub := range sensitiveSubstrings {
		if strings.Contains(lower, sub) {
			return true
		}
	}
	return false
}

// summarizeParameters extracts a human-readable parameter summary from the
// action JSON's "parameters" field. Values for keys that look sensitive
// (contain "secret", "password", "token", "key", etc.) are redacted to
// "***". Output is sorted by key for deterministic ordering.
// Returns "" if no parameters are present.
func summarizeParameters(action json.RawMessage) string {
	if len(action) == 0 {
		return ""
	}
	var obj map[string]json.RawMessage
	if json.Unmarshal(action, &obj) != nil {
		return ""
	}
	raw, ok := obj["parameters"]
	if !ok {
		return ""
	}
	var params map[string]json.RawMessage
	if json.Unmarshal(raw, &params) != nil {
		return ""
	}
	if len(params) == 0 {
		return ""
	}

	// Sort keys for deterministic output.
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var parts []string
	for _, key := range keys {
		if isSensitiveKey(key) {
			parts = append(parts, key+"=***")
			continue
		}
		var s string
		if json.Unmarshal(params[key], &s) == nil {
			parts = append(parts, key+"="+TruncateUTF8(s, 30))
		} else {
			// Non-string value — show the key only
			parts = append(parts, key)
		}
	}

	return strings.Join(parts, ", ")
}

func buildStandingExecutionSubject(approval Approval) string {
	info := extractStandingExecutionInfo(approval)
	if info.ActionType != "" {
		return fmt.Sprintf("%s executed %s via standing approval", info.AgentName, info.ActionType)
	}
	return fmt.Sprintf("%s executed an action via standing approval", info.AgentName)
}

func buildStandingExecutionPlainBody(approval Approval) string {
	info := extractStandingExecutionInfo(approval)

	var b strings.Builder

	b.WriteString(fmt.Sprintf("%s auto-executed an action via a standing approval.\n\n", info.AgentName))

	if info.ActionType != "" {
		b.WriteString(fmt.Sprintf("Action: %s\n", info.ActionType))
	}

	if paramSummary := summarizeParameters(approval.Action); paramSummary != "" {
		b.WriteString(fmt.Sprintf("Parameters: %s\n", paramSummary))
	}

	if countLabel := info.executionCountLabel(); countLabel != "" {
		b.WriteString(fmt.Sprintf("Executions: %s\n", countLabel))
	}

	b.WriteString(fmt.Sprintf("Time: %s\n", approval.CreatedAt.UTC().Format(time.RFC1123)))

	if approval.ApprovalURL != "" {
		b.WriteString(fmt.Sprintf("\nView activity:\n%s\n", approval.ApprovalURL))
	}

	b.WriteString("\n---\nThis action was auto-approved via a standing approval. Manage standing approvals in the dashboard.\n")

	return b.String()
}

// formatStandingExecutionSMS builds a concise SMS for a standing approval
// execution. Format: "[AgentName] ran [action_type] (3 of 10 uses). View: [url]"
func formatStandingExecutionSMS(a Approval) string {
	info := extractStandingExecutionInfo(a)

	actionPart := info.ActionType
	if actionPart == "" {
		actionPart = "an action"
	}

	msg := fmt.Sprintf("[Permission Slip] %s ran %s", info.AgentName, actionPart)

	if countLabel := info.executionCountLabel(); countLabel != "" {
		msg += fmt.Sprintf(" (%s uses)", countLabel)
	}

	if a.ApprovalURL != "" {
		return fmt.Sprintf("%s. View: %s", msg, a.ApprovalURL)
	}

	return msg
}

func buildStandingExecutionHTMLBody(approval Approval) string {
	info := extractStandingExecutionInfo(approval)

	var b bytes.Buffer
	b.WriteString(`<!DOCTYPE html><html><head><meta charset="UTF-8"></head>`)
	b.WriteString(`<body style="font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif;max-width:600px;margin:0 auto;padding:20px;color:#1a1a1a;">`)

	// Header — blue accent (informational, not urgent)
	b.WriteString(`<div style="border-bottom:2px solid #2563eb;padding-bottom:16px;margin-bottom:20px;">`)
	b.WriteString(`<h2 style="margin:0 0 4px 0;font-size:20px;">Action Auto-Executed</h2>`)
	b.WriteString(fmt.Sprintf(`<p style="margin:0;color:#6b7280;font-size:14px;">by %s via standing approval</p>`, html.EscapeString(info.AgentName)))
	b.WriteString(`</div>`)

	// Details table
	b.WriteString(`<table style="width:100%;border-collapse:collapse;margin-bottom:20px;">`)
	if info.ActionType != "" {
		b.WriteString(emailDetailRow("Action", html.EscapeString(info.ActionType)))
	}
	if paramSummary := summarizeParameters(approval.Action); paramSummary != "" {
		b.WriteString(emailDetailRow("Parameters", html.EscapeString(paramSummary)))
	}
	if countLabel := info.executionCountLabel(); countLabel != "" {
		b.WriteString(emailDetailRow("Executions", html.EscapeString(countLabel)))
	}
	b.WriteString(emailDetailRow("Time", html.EscapeString(approval.CreatedAt.UTC().Format(time.RFC1123))))
	b.WriteString(`</table>`)

	// CTA button — blue to match informational tone
	if approval.ApprovalURL != "" {
		b.WriteString(fmt.Sprintf(
			`<div style="text-align:center;margin:24px 0;">
		<a href="%s" style="display:inline-block;background-color:#2563eb;color:#ffffff;padding:12px 32px;border-radius:6px;text-decoration:none;font-weight:600;font-size:16px;">View Activity</a>
		</div>`,
			html.EscapeString(approval.ApprovalURL),
		))
	}

	// Footer — standing-approval-specific messaging
	b.WriteString(`<div style="border-top:1px solid #e5e7eb;padding-top:12px;margin-top:20px;font-size:12px;color:#9ca3af;">`)
	b.WriteString(`<p style="margin:0;">This action was auto-approved via a standing approval. Manage standing approvals in the dashboard.</p>`)
	b.WriteString(`</div>`)

	b.WriteString(`</body></html>`)
	return b.String()
}
