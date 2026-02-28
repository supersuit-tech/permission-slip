package notify

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestBuildEmailSubject_WithActionType(t *testing.T) {
	t.Parallel()
	approval := sendGridTestApproval()
	subject := buildEmailSubject(approval)
	if subject != "Approval needed: email.send" {
		t.Errorf("expected 'Approval needed: email.send', got: %s", subject)
	}
}

func TestBuildEmailSubject_WithoutActionType(t *testing.T) {
	t.Parallel()
	approval := Approval{
		ApprovalID: "appr_test",
		Action:     json.RawMessage(`{}`),
	}
	subject := buildEmailSubject(approval)
	if subject != "Approval needed" {
		t.Errorf("expected 'Approval needed', got: %s", subject)
	}
}

func TestBuildEmailSubject_NilAction(t *testing.T) {
	t.Parallel()
	approval := Approval{ApprovalID: "appr_test"}
	subject := buildEmailSubject(approval)
	if subject != "Approval needed" {
		t.Errorf("expected 'Approval needed', got: %s", subject)
	}
}

func TestBuildEmailPlainBody_IncludesAllFields(t *testing.T) {
	t.Parallel()
	approval := sendGridTestApproval()
	body := buildEmailPlainBody(approval)

	checks := []string{
		"Deploy Bot",
		"email.send",
		"low",
		"Send deployment notification",
		"https://app.example.com/approve/appr_test123",
		"Permission Slip",
	}
	for _, check := range checks {
		if !strings.Contains(body, check) {
			t.Errorf("expected plain body to contain %q", check)
		}
	}
}

func TestBuildEmailPlainBody_FallbackAgentName(t *testing.T) {
	t.Parallel()
	approval := Approval{
		ApprovalID: "appr_test",
		AgentID:    99,
		AgentName:  "",
		Action:     json.RawMessage(`{"type":"test"}`),
		Context:    json.RawMessage(`{}`),
		ExpiresAt:  time.Now().Add(5 * time.Minute),
	}
	body := buildEmailPlainBody(approval)
	if !strings.Contains(body, "Agent 99") {
		t.Error("expected fallback 'Agent 99' in plain body")
	}
}

func TestBuildEmailPlainBody_NoSensitiveData(t *testing.T) {
	t.Parallel()
	approval := Approval{
		ApprovalID: "appr_test",
		AgentID:    42,
		AgentName:  "Bot",
		Action:     json.RawMessage(`{"type":"email.send","parameters":{"to":"secret@example.com","api_key":"sk-secret123"}}`),
		Context:    json.RawMessage(`{"description":"test"}`),
		ExpiresAt:  time.Now().Add(5 * time.Minute),
	}
	body := buildEmailPlainBody(approval)

	// Parameters should NOT appear in the email body.
	if strings.Contains(body, "secret@example.com") {
		t.Error("plain body should not contain action parameters (email address)")
	}
	if strings.Contains(body, "sk-secret123") {
		t.Error("plain body should not contain action parameters (API key)")
	}
}

func TestBuildEmailHTMLBody_ContainsApprovalURL(t *testing.T) {
	t.Parallel()
	approval := sendGridTestApproval()
	html := buildEmailHTMLBody(approval)

	if !strings.Contains(html, "https://app.example.com/approve/appr_test123") {
		t.Error("expected HTML body to contain approval URL")
	}
	if !strings.Contains(html, "Review Request") {
		t.Error("expected HTML body to contain CTA button text")
	}
}

func TestBuildEmailHTMLBody_RiskBadge(t *testing.T) {
	t.Parallel()

	tests := []struct {
		riskLevel string
		color     string
	}{
		{"high", "#dc2626"},
		{"low", "#059669"},
		{"medium", "#d97706"},
	}

	for _, tt := range tests {
		approval := Approval{
			ApprovalID: "appr_test",
			AgentID:    42,
			AgentName:  "Bot",
			Action:     json.RawMessage(`{"type":"test"}`),
			Context:    json.RawMessage(`{"risk_level":"` + tt.riskLevel + `"}`),
			ExpiresAt:  time.Now().Add(5 * time.Minute),
		}
		html := buildEmailHTMLBody(approval)
		if !strings.Contains(html, tt.color) {
			t.Errorf("risk_level=%s: expected color %s in HTML", tt.riskLevel, tt.color)
		}
	}
}

func TestBuildEmailHTMLBody_EscapesHTML(t *testing.T) {
	t.Parallel()
	approval := Approval{
		ApprovalID: "appr_test",
		AgentID:    42,
		AgentName:  `<script>alert("xss")</script>`,
		Action:     json.RawMessage(`{"type":"test"}`),
		Context:    json.RawMessage(`{"description":"<b>bold</b>"}`),
		ExpiresAt:  time.Now().Add(5 * time.Minute),
	}
	html := buildEmailHTMLBody(approval)

	if strings.Contains(html, "<script>") {
		t.Error("HTML body should escape script tags")
	}
	if !strings.Contains(html, "&lt;script&gt;") {
		t.Error("HTML body should contain escaped script tag")
	}
}

func TestExtractActionType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		raw      json.RawMessage
		expected string
	}{
		{"valid", json.RawMessage(`{"type":"email.send"}`), "email.send"},
		{"empty object", json.RawMessage(`{}`), ""},
		{"nil", nil, ""},
		{"invalid json", json.RawMessage(`not-json`), ""},
		{"type is number", json.RawMessage(`{"type":42}`), ""},
	}
	for _, tt := range tests {
		result := extractActionType(tt.raw)
		if result != tt.expected {
			t.Errorf("%s: expected %q, got %q", tt.name, tt.expected, result)
		}
	}
}

func TestExtractRiskLevel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		raw      json.RawMessage
		expected string
	}{
		{"valid", json.RawMessage(`{"risk_level":"high"}`), "high"},
		{"missing", json.RawMessage(`{}`), ""},
		{"nil", nil, ""},
	}
	for _, tt := range tests {
		result := extractRiskLevel(tt.raw)
		if result != tt.expected {
			t.Errorf("%s: expected %q, got %q", tt.name, tt.expected, result)
		}
	}
}

func TestExtractDescription(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		raw      json.RawMessage
		expected string
	}{
		{"valid", json.RawMessage(`{"description":"Send email"}`), "Send email"},
		{"missing", json.RawMessage(`{}`), ""},
		{"nil", nil, ""},
	}
	for _, tt := range tests {
		result := extractDescription(tt.raw)
		if result != tt.expected {
			t.Errorf("%s: expected %q, got %q", tt.name, tt.expected, result)
		}
	}
}
