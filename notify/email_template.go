package notify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"strings"
	"time"
)

// buildEmailSubject returns the email subject line for a notification.
func buildEmailSubject(approval Approval) string {
	if approval.Type == NotificationTypePaymentFailed {
		return "Action required: Your payment failed"
	}
	if approval.Type == NotificationTypeCardExpiring {
		return buildCardExpiringSubject(approval)
	}
	if approval.Type == NotificationTypeStandingExecution {
		return buildStandingExecutionSubject(approval)
	}
	actionType := extractActionType(approval.Action)
	if actionType != "" {
		return fmt.Sprintf("Approval needed: %s", actionType)
	}
	return "Approval needed"
}

// buildEmailPlainBody returns the plain-text email body for a notification.
func buildEmailPlainBody(approval Approval) string {
	if approval.Type == NotificationTypePaymentFailed {
		return buildPaymentFailedPlainBody(approval)
	}
	if approval.Type == NotificationTypeCardExpiring {
		return buildCardExpiringPlainBody(approval)
	}
	if approval.Type == NotificationTypeStandingExecution {
		return buildStandingExecutionPlainBody(approval)
	}

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

// buildPaymentFailedPlainBody returns the plain-text email body for a payment failure.
func buildPaymentFailedPlainBody(approval Approval) string {
	var b strings.Builder
	b.WriteString("Your subscription payment could not be processed.\n\n")
	b.WriteString("Please update your payment method to avoid losing access to paid features. ")
	b.WriteString("Stripe will automatically retry the payment over the next few days. ")
	b.WriteString("If all retries fail, your subscription will be cancelled and you'll be downgraded to the free plan.\n")
	if approval.ApprovalURL != "" {
		b.WriteString(fmt.Sprintf("\nUpdate payment method:\n%s\n", approval.ApprovalURL))
	}
	b.WriteString("\n---\nThis is an automated notification from Permission Slip.\n")
	return b.String()
}

// buildEmailHTMLBody returns the HTML email body for a notification.
// Uses inline styles only — no external CSS, no images.
func buildEmailHTMLBody(approval Approval) string {
	if approval.Type == NotificationTypePaymentFailed {
		return buildPaymentFailedHTMLBody(approval)
	}
	if approval.Type == NotificationTypeCardExpiring {
		return buildCardExpiringHTMLBody(approval)
	}
	if approval.Type == NotificationTypeStandingExecution {
		return buildStandingExecutionHTMLBody(approval)
	}

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
	b.WriteString(`<table style="width:100%;border-collapse:collapse;margin-bottom:20px;">`)
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

// buildPaymentFailedHTMLBody returns the HTML email body for a payment failure.
func buildPaymentFailedHTMLBody(approval Approval) string {
	var b bytes.Buffer
	b.WriteString(`<!DOCTYPE html><html><head><meta charset="UTF-8"></head>`)
	b.WriteString(`<body style="font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif;max-width:600px;margin:0 auto;padding:20px;color:#1a1a1a;">`)

	// Header — red accent to indicate urgency
	b.WriteString(`<div style="border-bottom:2px solid #dc2626;padding-bottom:16px;margin-bottom:20px;">`)
	b.WriteString(`<h2 style="margin:0 0 4px 0;font-size:20px;color:#dc2626;">Payment Failed</h2>`)
	b.WriteString(`<p style="margin:0;color:#6b7280;font-size:14px;">Your subscription payment could not be processed</p>`)
	b.WriteString(`</div>`)

	// Body
	b.WriteString(`<p style="margin:0 0 16px 0;line-height:1.6;">`)
	b.WriteString(`Please update your payment method to avoid losing access to paid features. `)
	b.WriteString(`Stripe will automatically retry the payment over the next few days.`)
	b.WriteString(`</p>`)
	b.WriteString(`<p style="margin:0 0 16px 0;line-height:1.6;color:#6b7280;">`)
	b.WriteString(`If all retries fail, your subscription will be cancelled and you&#39;ll be downgraded to the free plan.`)
	b.WriteString(`</p>`)

	// CTA button — red to match urgency
	if approval.ApprovalURL != "" {
		b.WriteString(fmt.Sprintf(
			`<div style="text-align:center;margin:24px 0;">
		<a href="%s" style="display:inline-block;background-color:#dc2626;color:#ffffff;padding:12px 32px;border-radius:6px;text-decoration:none;font-weight:600;font-size:16px;">Update Payment Method</a>
		</div>`,
			html.EscapeString(approval.ApprovalURL),
		))
	}

	// Footer
	b.WriteString(`<div style="border-top:1px solid #e5e7eb;padding-top:12px;margin-top:20px;font-size:12px;color:#9ca3af;">`)
	b.WriteString(`<p style="margin:0;">This is an automated notification from Permission Slip.</p>`)
	b.WriteString(`</div>`)

	b.WriteString(`</body></html>`)
	return b.String()
}

func buildCardExpiringSubject(approval Approval) string {
	info := extractCardExpiringInfo(approval.Context)
	if info.Expired {
		return fmt.Sprintf("Your %s has expired", info.CardIdentifier())
	}
	return fmt.Sprintf("Your %s is expiring soon", info.CardIdentifier())
}

func buildCardExpiringPlainBody(approval Approval) string {
	info := extractCardExpiringInfo(approval.Context)
	var b strings.Builder
	if info.Expired {
		b.WriteString(fmt.Sprintf("Your %s (expires %s) has expired.\n\n",
			info.CardIdentifier(), formatCardExpiry(info.ExpMonth, info.ExpYear)))
		b.WriteString("Any connector actions that use this card will fail until you replace it.\n")
	} else {
		b.WriteString(fmt.Sprintf("Your %s expires %s.\n\n",
			info.CardIdentifier(), formatCardExpiry(info.ExpMonth, info.ExpYear)))
		b.WriteString("Please add a replacement card before it expires to avoid disruptions to connector actions.\n")
	}
	if approval.ApprovalURL != "" {
		b.WriteString(fmt.Sprintf("\nUpdate payment methods:\n%s\n", approval.ApprovalURL))
	}
	b.WriteString("\n---\nThis is an automated notification from Permission Slip.\n")
	return b.String()
}

func buildCardExpiringHTMLBody(approval Approval) string {
	info := extractCardExpiringInfo(approval.Context)
	isExpired := info.Expired

	var b bytes.Buffer
	b.WriteString(`<!DOCTYPE html><html><head><meta charset="UTF-8"></head>`)
	b.WriteString(`<body style="font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif;max-width:600px;margin:0 auto;padding:20px;color:#1a1a1a;">`)

	// Header — amber for expiring soon, red for expired
	accentColor := "#d97706"
	headerTitle := "Card Expiring Soon"
	if isExpired {
		accentColor = "#dc2626"
		headerTitle = "Card Expired"
	}
	b.WriteString(fmt.Sprintf(`<div style="border-bottom:2px solid %s;padding-bottom:16px;margin-bottom:20px;">`, accentColor))
	b.WriteString(fmt.Sprintf(`<h2 style="margin:0 0 4px 0;font-size:20px;color:%s;">%s</h2>`, accentColor, headerTitle))

	subtitle := fmt.Sprintf("%s &middot; expires %s",
		html.EscapeString(info.CardIdentifier()),
		html.EscapeString(formatCardExpiry(info.ExpMonth, info.ExpYear)))
	b.WriteString(fmt.Sprintf(`<p style="margin:0;color:#6b7280;font-size:14px;">%s</p>`, subtitle))
	b.WriteString(`</div>`)

	// Body
	b.WriteString(`<p style="margin:0 0 16px 0;line-height:1.6;">`)
	if isExpired {
		b.WriteString(`This card has expired. Any connector actions that try to use it will fail. `)
		b.WriteString(`Please add a replacement card in your payment settings.`)
	} else {
		b.WriteString(`This card is expiring soon. Please add a replacement card before it expires `)
		b.WriteString(`to avoid disruptions to connector actions.`)
	}
	b.WriteString(`</p>`)

	// CTA button
	if approval.ApprovalURL != "" {
		b.WriteString(fmt.Sprintf(
			`<div style="text-align:center;margin:24px 0;">
		<a href="%s" style="display:inline-block;background-color:%s;color:#ffffff;padding:12px 32px;border-radius:6px;text-decoration:none;font-weight:600;font-size:16px;">Update Payment Methods</a>
		</div>`,
			html.EscapeString(approval.ApprovalURL), accentColor,
		))
	}

	// Footer
	b.WriteString(`<div style="border-top:1px solid #e5e7eb;padding-top:12px;margin-top:20px;font-size:12px;color:#9ca3af;">`)
	b.WriteString(`<p style="margin:0;">This is an automated notification from Permission Slip.</p>`)
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

// extractJSONString pulls a string value from a top-level key in a JSON
// object. Returns "" if the key is missing, not a string, or the JSON is
// invalid. Used to extract fields like "type", "description", and
// "risk_level" from action/context JSONB without defining full structs.
func extractJSONString(raw json.RawMessage, key string) string {
	if len(raw) == 0 {
		return ""
	}
	var m map[string]json.RawMessage
	if json.Unmarshal(raw, &m) != nil {
		return ""
	}
	if v, ok := m[key]; ok {
		var s string
		if json.Unmarshal(v, &s) == nil {
			return s
		}
	}
	return ""
}

func extractActionType(raw json.RawMessage) string  { return extractJSONString(raw, "type") }
func extractDescription(raw json.RawMessage) string { return extractJSONString(raw, "description") }
func extractRiskLevel(raw json.RawMessage) string   { return extractJSONString(raw, "risk_level") }
