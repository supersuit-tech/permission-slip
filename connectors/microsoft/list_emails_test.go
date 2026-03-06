package microsoft

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestListEmails_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-access-token-123" {
			t.Errorf("expected Bearer token in Authorization header, got %q", got)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"value": []map[string]any{
				{
					"id":      "msg-1",
					"subject": "Hello",
					"from": map[string]any{
						"emailAddress": map[string]string{
							"name":    "Sender",
							"address": "sender@example.com",
						},
					},
					"toRecipients": []map[string]any{
						{
							"emailAddress": map[string]string{
								"name":    "Recipient",
								"address": "recipient@example.com",
							},
						},
					},
					"receivedDateTime": "2024-01-15T09:00:00Z",
					"isRead":           false,
					"bodyPreview":      "Preview text",
					"hasAttachments":   true,
				},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &listEmailsAction{conn: conn}

	params, _ := json.Marshal(listEmailsParams{})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.list_emails",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var summaries []emailSummary
	if err := json.Unmarshal(result.Data, &summaries); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if len(summaries) != 1 {
		t.Fatalf("expected 1 email, got %d", len(summaries))
	}
	if summaries[0].ID != "msg-1" {
		t.Errorf("expected id 'msg-1', got %q", summaries[0].ID)
	}
	if summaries[0].Subject != "Hello" {
		t.Errorf("expected subject 'Hello', got %q", summaries[0].Subject)
	}
	if summaries[0].From != "sender@example.com" {
		t.Errorf("expected from 'sender@example.com', got %q", summaries[0].From)
	}
	if !summaries[0].HasAttachments {
		t.Error("expected hasAttachments to be true")
	}
}

func TestListEmails_DefaultParams(t *testing.T) {
	t.Parallel()

	var params listEmailsParams
	params.defaults()
	if params.Folder != "inbox" {
		t.Errorf("expected default folder 'inbox', got %q", params.Folder)
	}
	if params.Top != 10 {
		t.Errorf("expected default top 10, got %d", params.Top)
	}
}

func TestListEmails_TopClamped(t *testing.T) {
	t.Parallel()

	params := listEmailsParams{Top: 100}
	params.defaults()
	if params.Top != 50 {
		t.Errorf("expected top clamped to 50, got %d", params.Top)
	}
}

func TestListEmails_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &listEmailsAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.list_emails",
		Parameters:  []byte(`{invalid`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}
