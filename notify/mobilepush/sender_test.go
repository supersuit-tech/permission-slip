package mobilepush

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip-web/notify"
)

func TestBuildMessage(t *testing.T) {
	t.Parallel()
	t.Run("basic approval", func(t *testing.T) {
		t.Parallel()
		approval := notify.Approval{
			ApprovalID:  "appr_abc123",
			AgentName:   "code-reviewer",
			ApprovalURL: "https://app.test/approve/appr_abc123",
			ExpiresAt:   time.Now().Add(time.Hour),
		}

		msg := buildMessage(approval)
		if msg.Title != "code-reviewer" {
			t.Errorf("expected title 'code-reviewer', got %q", msg.Title)
		}
		if msg.ApprovalID != "appr_abc123" {
			t.Errorf("expected approval_id 'appr_abc123', got %q", msg.ApprovalID)
		}
		if msg.URL != "https://app.test/approve/appr_abc123" {
			t.Errorf("expected URL, got %q", msg.URL)
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

		msg := buildMessage(approval)
		if msg.Body != "Send invoice to client" {
			t.Errorf("expected summary body, got %q", msg.Body)
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

		msg := buildMessage(approval)
		if msg.Body != "github.merge_pr" {
			t.Errorf("expected action type as body, got %q", msg.Body)
		}
	})

	t.Run("no agent name uses default title", func(t *testing.T) {
		t.Parallel()
		approval := notify.Approval{}

		msg := buildMessage(approval)
		if msg.Title != "Approval Request" {
			t.Errorf("expected default title, got %q", msg.Title)
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

		msg := buildMessage(approval)
		if len(msg.Body) > 200 {
			t.Errorf("expected body to be truncated to 200 chars, got %d", len(msg.Body))
		}
		if msg.Body[len(msg.Body)-3:] != "..." {
			t.Errorf("expected truncated body to end with '...', got %q", msg.Body[len(msg.Body)-3:])
		}
	})

	t.Run("payment failed notification", func(t *testing.T) {
		t.Parallel()
		approval := notify.Approval{
			Type:        notify.NotificationTypePaymentFailed,
			ApprovalURL: "https://app.test/settings/billing",
			ApprovalID:  "appr_billing",
		}

		msg := buildMessage(approval)
		if msg.Title != "Payment Failed" {
			t.Errorf("expected 'Payment Failed', got %q", msg.Title)
		}
		if msg.URL != "https://app.test/settings/billing" {
			t.Errorf("expected billing URL, got %q", msg.URL)
		}
	})
}

func TestSenderName(t *testing.T) {
	t.Parallel()
	s := &Sender{}
	if s.Name() != "mobile-push" {
		t.Errorf("expected Name() to return 'mobile-push', got %q", s.Name())
	}
}
