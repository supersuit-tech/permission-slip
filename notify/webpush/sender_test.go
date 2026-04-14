package webpush

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip/notify"
)

func TestBuildBody(t *testing.T) {
	t.Parallel()
	t.Run("basic approval", func(t *testing.T) {
		t.Parallel()
		approval := notify.Approval{
			ApprovalID:  "appr_abc123",
			AgentName:   "code-reviewer",
			ApprovalURL: "https://app.test/approve/appr_abc123",
			ExpiresAt:   time.Now().Add(time.Hour),
		}

		body := buildBody(approval)
		if body.Title != "code-reviewer" {
			t.Errorf("expected title 'code-reviewer', got %q", body.Title)
		}
		if body.ApprovalID != "appr_abc123" {
			t.Errorf("expected approval_id 'appr_abc123', got %q", body.ApprovalID)
		}
		if body.URL != "https://app.test/approve/appr_abc123" {
			t.Errorf("expected URL, got %q", body.URL)
		}
	})

	t.Run("action with summary", func(t *testing.T) {
		t.Parallel()
		action := map[string]string{"type": "email.send", "summary": "Send invoice to client"}
		actionJSON, _ := json.Marshal(action)

		approval := notify.Approval{
			AgentName: "billing-bot",
			Action:    actionJSON,
		}

		body := buildBody(approval)
		if body.Body != "Send invoice to client" {
			t.Errorf("expected summary body, got %q", body.Body)
		}
	})

	t.Run("action with only type", func(t *testing.T) {
		t.Parallel()
		action := map[string]string{"type": "github.merge_pr"}
		actionJSON, _ := json.Marshal(action)

		approval := notify.Approval{
			AgentName: "deploy-bot",
			Action:    actionJSON,
		}

		body := buildBody(approval)
		if body.Body != "github.merge_pr" {
			t.Errorf("expected action type as body, got %q", body.Body)
		}
	})

	t.Run("no agent name uses default title", func(t *testing.T) {
		t.Parallel()
		approval := notify.Approval{}

		body := buildBody(approval)
		if body.Title != "Approval Request" {
			t.Errorf("expected default title, got %q", body.Title)
		}
	})

	t.Run("long summary is truncated", func(t *testing.T) {
		t.Parallel()
		longSummary := ""
		for i := 0; i < 250; i++ {
			longSummary += "x"
		}
		action := map[string]string{"summary": longSummary}
		actionJSON, _ := json.Marshal(action)

		approval := notify.Approval{
			AgentName: "test",
			Action:    actionJSON,
		}

		body := buildBody(approval)
		if len(body.Body) > 200 {
			t.Errorf("expected body to be truncated to 200 chars, got %d", len(body.Body))
		}
		if body.Body[len(body.Body)-3:] != "..." {
			t.Errorf("expected truncated body to end with '...', got %q", body.Body[len(body.Body)-3:])
		}
	})
}

func TestBuildBody_StandingExecution(t *testing.T) {
	t.Parallel()
	approval := notify.Approval{
		ApprovalID:  "appr_standing_001",
		AgentName:   "Deploy Bot",
		Action:      json.RawMessage(`{"type":"github.issues.create"}`),
		Context:     json.RawMessage(`{}`),
		ApprovalURL: "https://app.test/activity",
		Type:        notify.NotificationTypeStandingExecution,
	}

	body := buildBody(approval)
	if body.Title != "Deploy Bot auto-executed" {
		t.Errorf("expected title 'Deploy Bot auto-executed', got %q", body.Title)
	}
	if body.Body != "github.issues.create" {
		t.Errorf("expected body 'github.issues.create', got %q", body.Body)
	}
	if body.URL != "https://app.test/activity" {
		t.Errorf("expected activity URL, got %q", body.URL)
	}
}

func TestSenderName(t *testing.T) {
	t.Parallel()
	s := &Sender{}
	if s.Name() != "web-push" {
		t.Errorf("expected Name() to return 'web-push', got %q", s.Name())
	}
}
