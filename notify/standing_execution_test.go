package notify

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func standingExecutionApproval() Approval {
	return Approval{
		ApprovalID:  "appr_standing_001",
		AgentID:     42,
		AgentName:   "Deploy Bot",
		Action:      json.RawMessage(`{"type":"github.issues.create"}`),
		Context:     json.RawMessage(`{"execution_count":3,"max_executions":10}`),
		ApprovalURL: "https://app.example.com/activity",
		CreatedAt:   time.Date(2026, 3, 14, 12, 0, 0, 0, time.UTC),
		Type:        NotificationTypeStandingExecution,
	}
}

// ── extractStandingExecutionInfo tests ──────────────────────────────────────

func TestExtractStandingExecutionInfo_Full(t *testing.T) {
	t.Parallel()
	a := standingExecutionApproval()
	info := extractStandingExecutionInfo(a)

	if info.AgentName != "Deploy Bot" {
		t.Errorf("expected agent name 'Deploy Bot', got %q", info.AgentName)
	}
	if info.ActionType != "github.issues.create" {
		t.Errorf("expected action type 'github.issues.create', got %q", info.ActionType)
	}
	if info.ExecutionCount != 3 {
		t.Errorf("expected execution count 3, got %d", info.ExecutionCount)
	}
	if info.MaxExecutions != 10 {
		t.Errorf("expected max executions 10, got %d", info.MaxExecutions)
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
	if info.ExecutionCount != 0 {
		t.Errorf("expected 0 execution count, got %d", info.ExecutionCount)
	}
}

func TestExtractStandingExecutionInfo_Unlimited(t *testing.T) {
	t.Parallel()
	a := Approval{
		AgentName: "Bot",
		Action:    json.RawMessage(`{"type":"test"}`),
		Context:   json.RawMessage(`{"execution_count":5,"max_executions":0}`),
		Type:      NotificationTypeStandingExecution,
	}
	info := extractStandingExecutionInfo(a)
	if info.executionCountLabel() != "5" {
		t.Errorf("expected '5' for unlimited, got %q", info.executionCountLabel())
	}
}

func TestExecutionCountLabel_WithMax(t *testing.T) {
	t.Parallel()
	info := standingExecutionInfo{ExecutionCount: 3, MaxExecutions: 10}
	if info.executionCountLabel() != "3 of 10" {
		t.Errorf("expected '3 of 10', got %q", info.executionCountLabel())
	}
}

func TestExecutionCountLabel_NoCount(t *testing.T) {
	t.Parallel()
	info := standingExecutionInfo{}
	if info.executionCountLabel() != "" {
		t.Errorf("expected empty label, got %q", info.executionCountLabel())
	}
}

// ── Email subject tests ─────────────────────────────────────────────────────

func TestBuildEmailSubject_StandingExecution(t *testing.T) {
	t.Parallel()
	a := standingExecutionApproval()
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
	a := standingExecutionApproval()
	body := buildEmailPlainBody(a)

	checks := []string{
		"Deploy Bot",
		"github.issues.create",
		"3 of 10",
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
		Context:   json.RawMessage(`{"execution_count":1,"max_executions":5}`),
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
	a := standingExecutionApproval()
	h := buildEmailHTMLBody(a)

	checks := []string{
		"#2563eb",                             // blue accent
		"Auto-Executed",                       // header
		"Deploy Bot",                          // agent name
		"github.issues.create",                // action type
		"3 of 10",                             // execution count
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
		Context:   json.RawMessage(`{"execution_count":1}`),
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
		Context:   json.RawMessage(`{"execution_count":1}`),
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
	a := standingExecutionApproval()
	body := formatSMSBody(a)

	checks := []string{
		"Deploy Bot",
		"github.issues.create",
		"3 of 10 uses",
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
		Context:   json.RawMessage(`{"execution_count":2,"max_executions":5}`),
		Type:      NotificationTypeStandingExecution,
	}
	body := formatSMSBody(a)
	if strings.Contains(body, "View:") {
		t.Error("expected no View URL")
	}
	if !strings.Contains(body, "2 of 5 uses") {
		t.Errorf("expected execution count, got: %s", body)
	}
}

func TestFormatSMSBody_StandingExecution_Unlimited(t *testing.T) {
	t.Parallel()
	a := Approval{
		AgentName:   "Bot",
		Action:      json.RawMessage(`{"type":"test"}`),
		Context:     json.RawMessage(`{"execution_count":7}`),
		ApprovalURL: "https://example.com/activity",
		Type:        NotificationTypeStandingExecution,
	}
	body := formatSMSBody(a)
	if !strings.Contains(body, "7 uses") {
		t.Errorf("expected execution count, got: %s", body)
	}
}

// ── Push content tests ──────────────────────────────────────────────────────

func TestBuildPushContent_StandingExecution(t *testing.T) {
	t.Parallel()
	a := standingExecutionApproval()
	c := BuildPushContent(a)

	if c.Title != "Deploy Bot auto-executed" {
		t.Errorf("expected title 'Deploy Bot auto-executed', got %q", c.Title)
	}
	if c.Body != "github.issues.create (#3)" {
		t.Errorf("expected body 'github.issues.create (#3)', got %q", c.Body)
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

func TestBuildPushContent_StandingExecution_NoAction(t *testing.T) {
	t.Parallel()
	a := Approval{
		AgentName: "Bot",
		Context:   json.RawMessage(`{"execution_count":1}`),
		Type:      NotificationTypeStandingExecution,
	}
	c := BuildPushContent(a)
	if c.Body != "an action (#1)" {
		t.Errorf("expected fallback body, got %q", c.Body)
	}
}
