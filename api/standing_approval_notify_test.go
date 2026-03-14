package api

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/db"
	"github.com/supersuit-tech/permission-slip-web/notify"
)

func TestNotifyStandingApprovalExecution_NilNotifier(t *testing.T) {
	t.Parallel()
	// Should not panic when Notifier is nil.
	deps := &Deps{Notifier: nil}
	exec := &db.StandingApprovalExecution{UserID: "u1", StandingApprovalID: "sa1"}
	agent := &db.Agent{AgentID: 1}
	NotifyStandingApprovalExecution(context.Background(), deps, exec, agent, "test.action", nil)
}

func TestNotifyStandingApprovalExecution_EmptyBaseURL(t *testing.T) {
	t.Parallel()
	// Should skip when BaseURL is empty (even with a notifier).
	deps := &Deps{
		Notifier: notify.NewDispatcher(nil, nil),
		BaseURL:  "",
	}
	exec := &db.StandingApprovalExecution{UserID: "u1", StandingApprovalID: "sa1"}
	agent := &db.Agent{AgentID: 1}
	NotifyStandingApprovalExecution(context.Background(), deps, exec, agent, "test.action", nil)
}

func TestBuildActionJSON_WithParameters(t *testing.T) {
	t.Parallel()
	params := json.RawMessage(`{"repo":"myrepo","title":"Bug fix"}`)
	result := buildActionJSON("github.issues.create", params)

	var parsed map[string]json.RawMessage
	if err := json.Unmarshal(result, &parsed); err != nil {
		t.Fatalf("failed to parse action JSON: %v", err)
	}

	var actionType string
	if err := json.Unmarshal(parsed["type"], &actionType); err != nil {
		t.Fatalf("failed to parse type: %v", err)
	}
	if actionType != "github.issues.create" {
		t.Errorf("expected type 'github.issues.create', got: %s", actionType)
	}

	if _, ok := parsed["parameters"]; !ok {
		t.Error("expected parameters key in action JSON")
	}
}

func TestBuildActionJSON_WithoutParameters(t *testing.T) {
	t.Parallel()
	result := buildActionJSON("slack.send", nil)

	var parsed map[string]json.RawMessage
	if err := json.Unmarshal(result, &parsed); err != nil {
		t.Fatalf("failed to parse action JSON: %v", err)
	}

	var actionType string
	if err := json.Unmarshal(parsed["type"], &actionType); err != nil {
		t.Fatalf("failed to parse type: %v", err)
	}
	if actionType != "slack.send" {
		t.Errorf("expected type 'slack.send', got: %s", actionType)
	}

	if _, ok := parsed["parameters"]; ok {
		t.Error("expected no parameters key when nil")
	}
}

func TestBuildActionJSON_EmptyParameters(t *testing.T) {
	t.Parallel()
	// Empty byte slice should be treated the same as nil.
	result := buildActionJSON("test.action", json.RawMessage{})

	var parsed map[string]json.RawMessage
	if err := json.Unmarshal(result, &parsed); err != nil {
		t.Fatalf("failed to parse action JSON: %v", err)
	}

	if _, ok := parsed["parameters"]; ok {
		t.Error("expected no parameters key for empty slice")
	}
}
