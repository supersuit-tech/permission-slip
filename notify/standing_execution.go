// Standing approval execution notification templates.
//
// When an agent executes an action via a standing approval, these templates
// generate informational notifications across all channels (email, SMS, push).
// The templates use a blue/informational tone (not red/amber) since no user
// action is needed — the execution was pre-authorized.
//
// # Context JSON contract
//
// The Approval.Context field may be empty or a JSON object with optional keys
// for future extensions. Execution quotas are not tracked in notifications.
//
// The Approval.Action field should contain the standard action JSON with a
// "type" key and optional "parameters" object. Parameter values are included
// in email notifications with sensitive keys automatically redacted.
//
// # Integration (Phase 2E)
//
// To dispatch a standing execution notification, construct a notify.Approval
// with Type set to NotificationTypeStandingExecution and call
// deps.Notifier.Dispatch(). See api/approval_notify.go for the existing
// pattern used by approval request notifications.
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

// standingExecutionInfo holds the fields extracted from the approval payload
// for a standing approval execution notification.
type standingExecutionInfo struct {
	AgentName  string // resolved display name
	ActionType string // e.g. "github.issues.create"
}

func extractStandingExecutionInfo(approval Approval) standingExecutionInfo {
	return standingExecutionInfo{
		AgentName:  AgentDisplayName(approval.AgentName, approval.AgentID),
		ActionType: extractActionType(approval.Action),
	}
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

	return PushContent{
		Title:      title,
		Body:       body,
		URL:        approval.ApprovalURL,
		ApprovalID: approval.ApprovalID,
	}
}

// sensitiveSubstrings are substrings that, when found in a lowercased
// parameter key, cause the value to be redacted. The substrings are chosen
// to catch compound keys like "aws_secret_access_key", "db_password",
// "oauth_token" while avoiding false positives on benign names like
// "author" or "hotkey".
var sensitiveSubstrings = []string{
	"secret", "password", "passwd", "token", "credential",
	"_key", "apikey", "access_key", "auth_token", "authz", "private",
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
// (see sensitiveSubstrings) are redacted to "***". Non-string primitives
// (numbers, booleans) are rendered as-is; complex objects show key-only.
// Output is sorted by key for deterministic ordering.
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
			// For non-string primitives (numbers, booleans) render the raw token;
			// for complex objects just show the key to avoid information overload.
			raw := params[key]
			if len(raw) > 0 && raw[0] != '{' && raw[0] != '[' {
				parts = append(parts, key+"="+string(raw))
			} else {
				parts = append(parts, key)
			}
		}
	}

	return strings.Join(parts, ", ")
}

// buildStandingExecutionSubject returns a subject like
// "Deploy Bot executed github.issues.create via standing approval".
func buildStandingExecutionSubject(approval Approval) string {
	info := extractStandingExecutionInfo(approval)
	if info.ActionType != "" {
		return fmt.Sprintf("%s executed %s via standing approval", info.AgentName, info.ActionType)
	}
	return fmt.Sprintf("%s executed an action via standing approval", info.AgentName)
}

// buildStandingExecutionPlainBody returns the plain-text email body with
// agent name, action type, parameter summary, and timestamp.
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

	b.WriteString(fmt.Sprintf("Time: %s\n", approval.CreatedAt.UTC().Format(time.RFC1123)))

	if approval.ApprovalURL != "" {
		b.WriteString(fmt.Sprintf("\nView activity:\n%s\n", approval.ApprovalURL))
	}

	b.WriteString("\n---\nThis action was auto-approved via a standing approval. Manage standing approvals in the dashboard.\n")

	return b.String()
}

// formatStandingExecutionSMS builds a concise SMS for a standing approval
// execution. Format: "[AgentName] ran [action_type]. View: [url]"
func formatStandingExecutionSMS(a Approval) string {
	info := extractStandingExecutionInfo(a)

	actionPart := info.ActionType
	if actionPart == "" {
		actionPart = "an action"
	}

	msg := fmt.Sprintf("[Permission Slip] %s ran %s", info.AgentName, actionPart)

	if a.ApprovalURL != "" {
		return fmt.Sprintf("%s. View: %s", msg, a.ApprovalURL)
	}

	return msg
}

// buildStandingExecutionHTMLBody returns the HTML email body with a blue
// accent (#2563eb), details table, "View Activity" CTA button, and a footer
// noting this was an auto-approved action.
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
