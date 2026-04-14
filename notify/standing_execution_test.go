package notify

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestExtractStandingExecutionInfo_Full(t *testing.T) {
	t.Parallel()
	a := testStandingExecutionApproval()
	info := extractStandingExecutionInfo(a)

	if info.AgentName != "Deploy Bot" {
		t.Errorf("expected agent name 'Deploy Bot', got %q", info.AgentName)
	}
	if info.ActionType != "github.issues.create" {
		t.Errorf("expected action type 'github.issues.create', got %q", info.ActionType)
	}
}

func TestExtractStandingExecutionInfo_NoContext(t *testing.T) {
	t.Parallel()
	a := Approval{
		AgentID:   99,
		AgentName: "",
		Action:    json.RawMessage(`{"type":"test"}`),
		Type:      NotificationTypeStandingExecution,
	}
	info := extractStandingExecutionInfo(a)

	if info.AgentName != "Agent #99" {
		t.Errorf("expected fallback agent name, got %q", info.AgentName)
	}
}

// ── Email subject tests ─────────────────────────────────────────────────────

func TestBuildEmailSubject_StandingExecution(t *testing.T) {
	t.Parallel()
	a := testStandingExecutionApproval()
	subject := buildEmailSubject(a)
	expected := "Deploy Bot executed github.issues.create via standing approval"
	if subject != expected {
		t.Errorf("expected %q, got %q", expected, subject)
	}
}

func TestBuildEmailSubject_StandingExecution_NoActionType(t *testing.T) {
	t.Parallel()
	a := Approval{
		AgentName: "Bot",
		Action:    json.RawMessage(`{}`),
		Type:      NotificationTypeStandingExecution,
	}
	subject := buildEmailSubject(a)
	if !strings.Contains(subject, "executed an action") {
		t.Errorf("expected fallback subject, got %q", subject)
	}
}

// ── Email plain body tests ──────────────────────────────────────────────────

func TestBuildEmailPlainBody_StandingExecution(t *testing.T) {
	t.Parallel()
	a := testStandingExecutionApproval()
	body := buildEmailPlainBody(a)

	checks := []string{
		"Deploy Bot",
		"github.issues.create",
		"Parameters:",
		"auto-approved via a standing approval",
		"https://app.example.com/activity",
	}
	for _, check := range checks {
		if !strings.Contains(body, check) {
			t.Errorf("expected plain body to contain %q, body:\n%s", check, body)
		}
	}
}

func TestBuildEmailPlainBody_StandingExecution_NoSensitiveData(t *testing.T) {
	t.Parallel()
	a := Approval{
		AgentName: "Bot",
		Action:    json.RawMessage(`{"type":"email.send","parameters":{"api_key":"sk-secret123"}}`),
		CreatedAt: time.Now(),
		Type:      NotificationTypeStandingExecution,
	}
	body := buildEmailPlainBody(a)
	if strings.Contains(body, "sk-secret123") {
		t.Error("plain body should not contain sensitive parameters")
	}
}

// ── Email HTML body tests ───────────────────────────────────────────────────

func TestBuildEmailHTMLBody_StandingExecution(t *testing.T) {
	t.Parallel()
	a := testStandingExecutionApproval()
	h := buildEmailHTMLBody(a)

	checks := []string{
		"#2563eb",                             // blue accent
		"Auto-Executed",                       // header
		"Deploy Bot",                          // agent name
		"github.issues.create",                // action type
		"Parameters",                          // parameter summary row
		"View Activity",                       // CTA button
		"https://app.example.com/activity",    // URL
		"auto-approved via a standing approval", // footer
	}
	for _, check := range checks {
		if !strings.Contains(h, check) {
			t.Errorf("expected HTML to contain %q", check)
		}
	}
}

func TestBuildEmailHTMLBody_StandingExecution_EscapesHTML(t *testing.T) {
	t.Parallel()
	a := Approval{
		AgentName: `<script>alert("xss")</script>`,
		Action:    json.RawMessage(`{"type":"test"}`),
		CreatedAt: time.Now(),
		Type:      NotificationTypeStandingExecution,
	}
	h := buildEmailHTMLBody(a)
	if strings.Contains(h, "<script>") {
		t.Error("HTML body should escape script tags")
	}
	if !strings.Contains(h, "&lt;script&gt;") {
		t.Error("HTML body should contain escaped script tag")
	}
}

func TestBuildEmailHTMLBody_StandingExecution_NoURL(t *testing.T) {
	t.Parallel()
	a := Approval{
		AgentName: "Bot",
		Action:    json.RawMessage(`{"type":"test"}`),
		CreatedAt: time.Now(),
		Type:      NotificationTypeStandingExecution,
	}
	h := buildEmailHTMLBody(a)
	if strings.Contains(h, "View Activity") {
		t.Error("HTML body should not contain CTA button when URL is empty")
	}
}

// ── SMS tests ───────────────────────────────────────────────────────────────

func TestFormatSMSBody_StandingExecution(t *testing.T) {
	t.Parallel()
	a := testStandingExecutionApproval()
	body := formatSMSBody(a)

	checks := []string{
		"Deploy Bot",
		"github.issues.create",
		"View:",
		"https://app.example.com/activity",
	}
	for _, check := range checks {
		if !strings.Contains(body, check) {
			t.Errorf("expected SMS body to contain %q, got: %s", check, body)
		}
	}
}

func TestFormatSMSBody_StandingExecution_NoURL(t *testing.T) {
	t.Parallel()
	a := Approval{
		AgentName: "Bot",
		Action:    json.RawMessage(`{"type":"test"}`),
		Type:      NotificationTypeStandingExecution,
	}
	body := formatSMSBody(a)
	if strings.Contains(body, "View:") {
		t.Error("expected no View URL")
	}
}

func TestFormatSMSBody_StandingExecution_WithURL(t *testing.T) {
	t.Parallel()
	a := Approval{
		AgentName:   "Bot",
		Action:      json.RawMessage(`{"type":"test"}`),
		ApprovalURL: "https://example.com/activity",
		Type:        NotificationTypeStandingExecution,
	}
	body := formatSMSBody(a)
	if !strings.Contains(body, "View: https://example.com/activity") {
		t.Errorf("expected View URL in SMS, got: %s", body)
	}
}

// ── Push content tests ──────────────────────────────────────────────────────

func TestBuildPushContent_StandingExecution(t *testing.T) {
	t.Parallel()
	a := testStandingExecutionApproval()
	c := BuildPushContent(a)

	if c.Title != "Deploy Bot auto-executed" {
		t.Errorf("expected title 'Deploy Bot auto-executed', got %q", c.Title)
	}
	if c.Body != "github.issues.create" {
		t.Errorf("expected body 'github.issues.create', got %q", c.Body)
	}
	if c.URL != "https://app.example.com/activity" {
		t.Errorf("expected activity URL, got %q", c.URL)
	}
}

func TestBuildPushContent_StandingExecution_NoCount(t *testing.T) {
	t.Parallel()
	a := Approval{
		AgentName: "Bot",
		Action:    json.RawMessage(`{"type":"deploy"}`),
		Type:      NotificationTypeStandingExecution,
	}
	c := BuildPushContent(a)
	if c.Title != "Bot auto-executed" {
		t.Errorf("expected title, got %q", c.Title)
	}
	if c.Body != "deploy" {
		t.Errorf("expected body 'deploy', got %q", c.Body)
	}
}

// ── summarizeParameters tests ────────────────────────────────────────────────

func TestSummarizeParameters_WithParams(t *testing.T) {
	t.Parallel()
	action := json.RawMessage(`{"type":"test","parameters":{"repo":"acme/app","title":"Deploy v2.1"}}`)
	summary := summarizeParameters(action)
	// Output is sorted by key: repo before title.
	expected := "repo=acme/app, title=Deploy v2.1"
	if summary != expected {
		t.Errorf("expected %q, got %q", expected, summary)
	}
}

func TestSummarizeParameters_RedactsSensitive(t *testing.T) {
	t.Parallel()
	action := json.RawMessage(`{"type":"test","parameters":{"repo":"acme/app","api_key":"sk-secret123","token":"tok-abc"}}`)
	summary := summarizeParameters(action)
	if strings.Contains(summary, "sk-secret123") {
		t.Error("summary should redact api_key value")
	}
	if strings.Contains(summary, "tok-abc") {
		t.Error("summary should redact token value")
	}
	if !strings.Contains(summary, "api_key=***") {
		t.Errorf("expected redacted api_key, got: %s", summary)
	}
	if !strings.Contains(summary, "token=***") {
		t.Errorf("expected redacted token, got: %s", summary)
	}
}

func TestSummarizeParameters_RedactsCompoundSensitiveKeys(t *testing.T) {
	t.Parallel()
	// Compound key names like "aws_secret_access_key" and "db_password"
	// must also be redacted via substring matching.
	action := json.RawMessage(`{"type":"test","parameters":{"aws_secret_access_key":"AKIA...","db_password":"hunter2","oauth_token":"ghp_abc"}}`)
	summary := summarizeParameters(action)
	if strings.Contains(summary, "AKIA") {
		t.Error("summary should redact aws_secret_access_key value")
	}
	if strings.Contains(summary, "hunter2") {
		t.Error("summary should redact db_password value")
	}
	if strings.Contains(summary, "ghp_abc") {
		t.Error("summary should redact oauth_token value")
	}
	if !strings.Contains(summary, "aws_secret_access_key=***") {
		t.Errorf("expected redacted aws_secret_access_key, got: %s", summary)
	}
	if !strings.Contains(summary, "db_password=***") {
		t.Errorf("expected redacted db_password, got: %s", summary)
	}
}

func TestSummarizeParameters_NoFalsePositives(t *testing.T) {
	t.Parallel()
	// "author" and "hotkey" should NOT be redacted — they are benign parameter names.
	action := json.RawMessage(`{"type":"test","parameters":{"author":"alice","hotkey":"ctrl+s","keyboard":"us"}}`)
	summary := summarizeParameters(action)
	if !strings.Contains(summary, "author=alice") {
		t.Errorf("expected author to not be redacted, got: %s", summary)
	}
	if !strings.Contains(summary, "hotkey=ctrl+s") {
		t.Errorf("expected hotkey to not be redacted, got: %s", summary)
	}
	if !strings.Contains(summary, "keyboard=us") {
		t.Errorf("expected keyboard to not be redacted, got: %s", summary)
	}
}

func TestSummarizeParameters_NonStringPrimitives(t *testing.T) {
	t.Parallel()
	action := json.RawMessage(`{"type":"test","parameters":{"count":42,"enabled":true,"repo":"acme/app","nested":{"a":1}}}`)
	summary := summarizeParameters(action)
	if !strings.Contains(summary, "count=42") {
		t.Errorf("expected number value to be rendered, got: %s", summary)
	}
	if !strings.Contains(summary, "enabled=true") {
		t.Errorf("expected boolean value to be rendered, got: %s", summary)
	}
	if !strings.Contains(summary, "repo=acme/app") {
		t.Errorf("expected string value to be rendered, got: %s", summary)
	}
	// Nested objects should show key only
	if strings.Contains(summary, "nested=") {
		t.Errorf("expected nested object to show key only, got: %s", summary)
	}
	if !strings.Contains(summary, "nested") {
		t.Errorf("expected nested key to be present, got: %s", summary)
	}
}

func TestSummarizeParameters_NoParams(t *testing.T) {
	t.Parallel()
	action := json.RawMessage(`{"type":"test"}`)
	if summary := summarizeParameters(action); summary != "" {
		t.Errorf("expected empty summary, got: %s", summary)
	}
}

func TestSummarizeParameters_EmptyParams(t *testing.T) {
	t.Parallel()
	action := json.RawMessage(`{"type":"test","parameters":{}}`)
	if summary := summarizeParameters(action); summary != "" {
		t.Errorf("expected empty summary, got: %s", summary)
	}
}

func TestSummarizeParameters_LongValue(t *testing.T) {
	t.Parallel()
	longVal := strings.Repeat("x", 50)
	action := json.RawMessage(`{"type":"test","parameters":{"desc":"` + longVal + `"}}`)
	summary := summarizeParameters(action)
	if !strings.Contains(summary, "...") {
		t.Errorf("expected long value to be truncated, got: %s", summary)
	}
}

func TestSummarizeParameters_NilAction(t *testing.T) {
	t.Parallel()
	if summary := summarizeParameters(nil); summary != "" {
		t.Errorf("expected empty summary for nil, got: %s", summary)
	}
}

func TestBuildPushContent_StandingExecution_NoAction(t *testing.T) {
	t.Parallel()
	a := Approval{
		AgentName: "Bot",
		Type:      NotificationTypeStandingExecution,
	}
	c := BuildPushContent(a)
	if c.Body != "an action" {
		t.Errorf("expected fallback body, got %q", c.Body)
	}
}
