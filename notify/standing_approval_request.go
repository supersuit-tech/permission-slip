package notify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"strings"
)

func buildStandingApprovalRequestSubject(approval Approval) string {
	actionType := extractActionType(approval.Action)
	if actionType != "" {
		return fmt.Sprintf("Rule proposal: auto-approve %s", actionType)
	}
	return "Rule proposal: new auto-approve rule"
}

func buildStandingApprovalRequestPlainBody(approval Approval) string {
	var b strings.Builder
	agentName := approval.AgentName
	if agentName == "" {
		agentName = fmt.Sprintf("Agent %d", approval.AgentID)
	}
	b.WriteString(fmt.Sprintf("%s proposed a new auto-approve rule.\n\n", agentName))
	actionType := extractActionType(approval.Action)
	if actionType != "" {
		b.WriteString(fmt.Sprintf("Action type: %s\n", actionType))
	}
	description := extractDescription(approval.Context)
	if description != "" {
		b.WriteString(fmt.Sprintf("Summary: %s\n", description))
	}
	if approval.ApprovalURL != "" {
		b.WriteString(fmt.Sprintf("\nReview and respond:\n%s\n", approval.ApprovalURL))
	}
	b.WriteString("\n---\nThis is an automated notification from Permission Slip.\n")
	return b.String()
}

func buildStandingApprovalRequestHTMLBody(approval Approval) string {
	agentName := approval.AgentName
	if agentName == "" {
		agentName = fmt.Sprintf("Agent %d", approval.AgentID)
	}
	actionType := extractActionType(approval.Action)
	description := extractDescription(approval.Context)

	var b bytes.Buffer
	b.WriteString(`<!DOCTYPE html><html><head><meta charset="UTF-8"></head>`)
	b.WriteString(`<body style="font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif;max-width:600px;margin:0 auto;padding:20px;color:#1a1a1a;">`)
	b.WriteString(`<div style="border-bottom:2px solid #e5e7eb;padding-bottom:16px;margin-bottom:20px;">`)
	b.WriteString(`<h2 style="margin:0 0 4px 0;font-size:20px;">Auto-Approve Rule Proposal</h2>`)
	b.WriteString(fmt.Sprintf(`<p style="margin:0;color:#6b7280;font-size:14px;">from %s</p>`, html.EscapeString(agentName)))
	b.WriteString(`</div>`)
	if actionType != "" {
		b.WriteString(fmt.Sprintf(`<p style="margin:0 0 8px 0;"><strong>Action:</strong> %s</p>`, html.EscapeString(actionType)))
	}
	if description != "" {
		b.WriteString(fmt.Sprintf(`<p style="margin:0 0 16px 0;color:#374151;">%s</p>`, html.EscapeString(description)))
	}
	if constraints := formatConstraintsForEmail(approval.Action); constraints != "" {
		b.WriteString(fmt.Sprintf(`<pre style="background:#f3f4f6;padding:12px;border-radius:6px;font-size:12px;overflow-x:auto;">%s</pre>`, html.EscapeString(constraints)))
	}
	if approval.ApprovalURL != "" {
		b.WriteString(fmt.Sprintf(`<p style="margin-top:20px;"><a href="%s" style="display:inline-block;background:#2563eb;color:#fff;padding:10px 20px;border-radius:6px;text-decoration:none;font-weight:600;">Review rule proposal</a></p>`, html.EscapeString(approval.ApprovalURL)))
	}
	b.WriteString(`</body></html>`)
	return b.String()
}

func formatConstraintsForEmail(action json.RawMessage) string {
	var obj map[string]json.RawMessage
	if json.Unmarshal(action, &obj) != nil {
		return ""
	}
	raw, ok := obj["constraints"]
	if !ok || len(raw) == 0 {
		return ""
	}
	var pretty bytes.Buffer
	if json.Indent(&pretty, raw, "", "  ") != nil {
		return string(raw)
	}
	return pretty.String()
}

func buildStandingApprovalRequestPushContent(approval Approval) PushContent {
	agentName := AgentDisplayName(approval.AgentName, approval.AgentID)
	actionType := extractActionType(approval.Action)
	body := "New auto-approve rule"
	if actionType != "" {
		body = actionType
	}
	return PushContent{
		Title:      fmt.Sprintf("%s proposed a rule", agentName),
		Body:       body,
		URL:        approval.ApprovalURL,
		ApprovalID: approval.ApprovalID,
	}
}
