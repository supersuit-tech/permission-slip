package connectors

import (
	"context"
	"testing"
)

// stubAction is a minimal Action for testing.
type stubAction struct {
	name string
}

func (a *stubAction) Execute(_ context.Context, _ ActionRequest) (*ActionResult, error) {
	return &ActionResult{}, nil
}

// stubConnector is a minimal Connector for testing.
type stubConnector struct {
	id      string
	actions map[string]Action
}

func (c *stubConnector) ID() string                  { return c.id }
func (c *stubConnector) Actions() map[string]Action   { return c.actions }
func (c *stubConnector) ValidateCredentials(_ context.Context, _ Credentials) error {
	return nil
}

func newStubConnector(id string, actionTypes ...string) *stubConnector {
	actions := make(map[string]Action, len(actionTypes))
	for _, at := range actionTypes {
		actions[at] = &stubAction{name: at}
	}
	return &stubConnector{id: id, actions: actions}
}

func TestNewRegistry(t *testing.T) {
	t.Parallel()
	r := NewRegistry()
	if r == nil {
		t.Fatal("expected non-nil registry")
	}
}

func TestRegistry_RegisterAndGet(t *testing.T) {
	t.Parallel()
	r := NewRegistry()
	c := newStubConnector("github", "github.create_issue")

	r.Register(c)

	got, ok := r.Get("github")
	if !ok {
		t.Fatal("expected to find connector 'github'")
	}
	if got.ID() != "github" {
		t.Errorf("expected ID 'github', got %q", got.ID())
	}
}

func TestRegistry_GetMissing(t *testing.T) {
	t.Parallel()
	r := NewRegistry()

	_, ok := r.Get("nonexistent")
	if ok {
		t.Error("expected ok=false for missing connector")
	}
}

func TestRegistry_RegisterReplacesExisting(t *testing.T) {
	t.Parallel()
	r := NewRegistry()
	c1 := newStubConnector("github", "github.create_issue")
	c2 := newStubConnector("github", "github.create_issue", "github.merge_pr")

	r.Register(c1)
	r.Register(c2)

	got, ok := r.Get("github")
	if !ok {
		t.Fatal("expected to find connector 'github'")
	}
	if len(got.Actions()) != 2 {
		t.Errorf("expected 2 actions after replacement, got %d", len(got.Actions()))
	}
}

func TestRegistry_MultipleConnectors(t *testing.T) {
	t.Parallel()
	r := NewRegistry()
	r.Register(newStubConnector("github", "github.create_issue"))
	r.Register(newStubConnector("slack", "slack.send_message"))

	if _, ok := r.Get("github"); !ok {
		t.Error("expected to find connector 'github'")
	}
	if _, ok := r.Get("slack"); !ok {
		t.Error("expected to find connector 'slack'")
	}
}

func TestRegistry_GetAction(t *testing.T) {
	t.Parallel()
	r := NewRegistry()
	r.Register(newStubConnector("github", "github.create_issue", "github.merge_pr"))
	r.Register(newStubConnector("slack", "slack.send_message"))

	action, ok := r.GetAction("github.create_issue")
	if !ok {
		t.Fatal("expected to find action 'github.create_issue'")
	}
	if action == nil {
		t.Fatal("expected non-nil action")
	}

	action, ok = r.GetAction("slack.send_message")
	if !ok {
		t.Fatal("expected to find action 'slack.send_message'")
	}
	if action == nil {
		t.Fatal("expected non-nil action")
	}
}

func TestRegistry_GetActionMissingConnector(t *testing.T) {
	t.Parallel()
	r := NewRegistry()

	_, ok := r.GetAction("nonexistent.some_action")
	if ok {
		t.Error("expected ok=false for action with missing connector")
	}
}

func TestRegistry_GetActionMissingAction(t *testing.T) {
	t.Parallel()
	r := NewRegistry()
	r.Register(newStubConnector("github", "github.create_issue"))

	_, ok := r.GetAction("github.nonexistent_action")
	if ok {
		t.Error("expected ok=false for missing action on existing connector")
	}
}

func TestRegistry_GetActionNoDot(t *testing.T) {
	t.Parallel()
	r := NewRegistry()
	r.Register(newStubConnector("github", "github.create_issue"))

	_, ok := r.GetAction("github_create_issue")
	if ok {
		t.Error("expected ok=false for action type without dot separator")
	}
}

func TestRegistry_GetActionEmptyString(t *testing.T) {
	t.Parallel()
	r := NewRegistry()

	_, ok := r.GetAction("")
	if ok {
		t.Error("expected ok=false for empty action type")
	}
}

func TestRegistry_IDsEmpty(t *testing.T) {
	t.Parallel()
	r := NewRegistry()
	ids := r.IDs()
	if len(ids) != 0 {
		t.Errorf("expected 0 IDs from empty registry, got %d", len(ids))
	}
}

func TestRegistry_IDsReturnsAll(t *testing.T) {
	t.Parallel()
	r := NewRegistry()
	r.Register(newStubConnector("github", "github.create_issue"))
	r.Register(newStubConnector("slack", "slack.send_message"))

	ids := r.IDs()
	if len(ids) != 2 {
		t.Fatalf("expected 2 IDs, got %d", len(ids))
	}
	idSet := map[string]bool{}
	for _, id := range ids {
		idSet[id] = true
	}
	if !idSet["github"] || !idSet["slack"] {
		t.Errorf("expected IDs to contain github and slack, got %v", ids)
	}
}

func TestRegistry_GetActionMultipleDots(t *testing.T) {
	t.Parallel()
	// Action types with multiple dots should split on the first dot only.
	// The connector ID is "github" and the action type is "github.repos.create".
	r := NewRegistry()
	r.Register(newStubConnector("github", "github.repos.create"))

	action, ok := r.GetAction("github.repos.create")
	if !ok {
		t.Fatal("expected to find action 'github.repos.create'")
	}
	if action == nil {
		t.Fatal("expected non-nil action")
	}
}
