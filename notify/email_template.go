package notify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"strings"
	"time"
)

// buildEmailSubject returns the email subject line for an approval notification.
func buildEmailSubject(approval Approval) string {
	actionType := extractActionType(approval.Action)
	if actionType != "" {
		return fmt.Sprintf("Approval needed: %s", actionType)
	}
	return "Approval needed"
}

// buildEmailPlainBody returns the plain-text email body for an approval notification.
func buildEmailPlainBody(approval Approval) string {
	var b strings.Builder

	agentName := approval.AgentName
	if agentName == "" {
		agentName = fmt.Sprintf("Agent %d", approval.AgentID)
	}

	b.WriteString(fmt.Sprintf("%s is requesting approval.\n\n", agentName))

	actionType := extractActionType(approval.Action)
	if actionType != "" {
		b.WriteString(fmt.Sprintf("Action: %s\n", actionType))
	}

	riskLevel := extractRiskLevel(approval.Context)
	if riskLevel != "" {
		b.WriteString(fmt.Sprintf("Risk: %s\n", riskLevel))
	}

	description := extractDescription(approval.Context)
	if description != "" {
		b.WriteString(fmt.Sprintf("Summary: %s\n", description))
	}

	b.WriteString(fmt.Sprintf("Expires: %s\n", approval.ExpiresAt.UTC().Format(time.RFC1123)))
	if approval.ApprovalURL != "" {
		b.WriteString(fmt.Sprintf("\nReview and respond:\n%s\n", approval.ApprovalURL))
	}
	b.WriteString("\n---\nThis is an automated notification from Permission Slip.\n")
	b.WriteString("Do not reply to this email. View full details in the dashboard.\n")

	return b.String()
}

// buildEmailHTMLBody returns the HTML email body for an approval notification.
// Uses inline styles only — no external CSS, no images.
func buildEmailHTMLBody(approval Approval) string {
	agentName := approval.AgentName
	if agentName == "" {
		agentName = fmt.Sprintf("Agent %d", approval.AgentID)
	}

	actionType := extractActionType(approval.Action)
	riskLevel := extractRiskLevel(approval.Context)
	description := extractDescription(approval.Context)

	var b bytes.Buffer
	b.WriteString(`<!DOCTYPE html><html><head><meta charset="UTF-8"></head>`)
	b.WriteString(`<body style="font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif;max-width:600px;margin:0 auto;padding:20px;color:#1a1a1a;">`)

	// Header
	b.WriteString(`<div style="border-bottom:2px solid #e5e7eb;padding-bottom:16px;margin-bottom:20px;">`)
	b.WriteString(`<h2 style="margin:0 0 4px 0;font-size:20px;">Approval Request</h2>`)
	b.WriteString(fmt.Sprintf(`<p style="margin:0;color:#6b7280;font-size:14px;">from %s</p>`, html.EscapeString(agentName)))
	b.WriteString(`</div>`)

	// Details table
	b.WriteString(`<table style="width:100%%;border-collapse:collapse;margin-bottom:20px;">`)
	if actionType != "" {
		b.WriteString(emailDetailRow("Action", html.EscapeString(actionType)))
	}
	if riskLevel != "" {
		b.WriteString(emailDetailRow("Risk", emailRiskBadge(riskLevel)))
	}
	if description != "" {
		b.WriteString(emailDetailRow("Summary", html.EscapeString(description)))
	}
	b.WriteString(emailDetailRow("Expires", html.EscapeString(approval.ExpiresAt.UTC().Format(time.RFC1123))))
	b.WriteString(`</table>`)

	// CTA button — only rendered when an approval URL is available.
	if approval.ApprovalURL != "" {
		b.WriteString(fmt.Sprintf(
			`<div style="text-align:center;margin:24px 0;">
		<a href="%s" style="display:inline-block;background-color:#2563eb;color:#ffffff;padding:12px 32px;border-radius:6px;text-decoration:none;font-weight:600;font-size:16px;">Review Request</a>
		</div>`,
			html.EscapeString(approval.ApprovalURL),
		))
	}

	// Footer
	b.WriteString(`<div style="border-top:1px solid #e5e7eb;padding-top:12px;margin-top:20px;font-size:12px;color:#9ca3af;">`)
	b.WriteString(`<p style="margin:0;">This is an automated notification from Permission Slip. Do not reply to this email.</p>`)
	b.WriteString(`<p style="margin:4px 0 0 0;">View full details in the dashboard — sensitive parameters are not included in this email.</p>`)
	b.WriteString(`</div>`)

	b.WriteString(`</body></html>`)
	return b.String()
}

func emailDetailRow(label, value string) string {
	return fmt.Sprintf(
		`<tr><td style="padding:6px 12px 6px 0;color:#6b7280;vertical-align:top;white-space:nowrap;">%s</td><td style="padding:6px 0;">%s</td></tr>`,
		label, value,
	)
}

func emailRiskBadge(level string) string {
	color := "#6b7280" // default gray
	switch strings.ToLower(level) {
	case "high", "critical":
		color = "#dc2626"
	case "medium":
		color = "#d97706"
	case "low":
		color = "#059669"
	}
	return fmt.Sprintf(
		`<span style="display:inline-block;padding:2px 8px;border-radius:4px;font-size:12px;font-weight:600;color:#ffffff;background-color:%s;">%s</span>`,
		color, html.EscapeString(strings.ToUpper(level)),
	)
}

// extractActionType pulls "type" from the action JSONB.
func extractActionType(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var m map[string]json.RawMessage
	if json.Unmarshal(raw, &m) != nil {
		return ""
	}
	if v, ok := m["type"]; ok {
		var s string
		if json.Unmarshal(v, &s) == nil {
			return s
		}
	}
	return ""
}

// extractDescription pulls "description" from the context JSONB.
func extractDescription(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var m map[string]json.RawMessage
	if json.Unmarshal(raw, &m) != nil {
		return ""
	}
	if v, ok := m["description"]; ok {
		var s string
		if json.Unmarshal(v, &s) == nil {
			return s
		}
	}
	return ""
}

// extractRiskLevel pulls "risk_level" from the context JSONB.
func extractRiskLevel(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var m map[string]json.RawMessage
	if json.Unmarshal(raw, &m) != nil {
		return ""
	}
	if v, ok := m["risk_level"]; ok {
		var s string
		if json.Unmarshal(v, &s) == nil {
			return s
		}
	}
	return ""
}
