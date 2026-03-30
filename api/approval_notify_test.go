package api

import (
	"encoding/json"
	"testing"

	"github.com/supersuit-tech/permission-slip/db"
)

func TestExtractAgentName_WithName(t *testing.T) {
	t.Parallel()
	agent := &db.Agent{
		AgentID:  42,
		Metadata: json.RawMessage(`{"name":"Deploy Bot","version":"1.0"}`),
	}
	name := extractAgentName(agent)
	if name != "Deploy Bot" {
		t.Errorf("expected 'Deploy Bot', got: %s", name)
	}
}

func TestExtractAgentName_WithoutName(t *testing.T) {
	t.Parallel()
	agent := &db.Agent{
		AgentID:  42,
		Metadata: json.RawMessage(`{"version":"1.0"}`),
	}
	name := extractAgentName(agent)
	if name != "Agent 42" {
		t.Errorf("expected 'Agent 42', got: %s", name)
	}
}

func TestExtractAgentName_EmptyMetadata(t *testing.T) {
	t.Parallel()
	agent := &db.Agent{AgentID: 99}
	name := extractAgentName(agent)
	if name != "Agent 99" {
		t.Errorf("expected 'Agent 99', got: %s", name)
	}
}

func TestExtractAgentName_EmptyName(t *testing.T) {
	t.Parallel()
	agent := &db.Agent{
		AgentID:  42,
		Metadata: json.RawMessage(`{"name":""}`),
	}
	name := extractAgentName(agent)
	if name != "Agent 42" {
		t.Errorf("expected 'Agent 42', got: %s", name)
	}
}

func TestExtractAgentName_InvalidJSON(t *testing.T) {
	t.Parallel()
	agent := &db.Agent{
		AgentID:  42,
		Metadata: json.RawMessage(`not-json`),
	}
	name := extractAgentName(agent)
	if name != "Agent 42" {
		t.Errorf("expected 'Agent 42', got: %s", name)
	}
}
