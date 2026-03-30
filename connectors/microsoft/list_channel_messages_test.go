package microsoft

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestListChannelMessages_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-access-token-123" {
			t.Errorf("expected Bearer token, got %q", got)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"value": []map[string]any{
				{
					"id":              "msg-1",
					"createdDateTime": "2024-01-15T09:00:00Z",
					"from": map[string]any{
						"user": map[string]any{
							"displayName": "Alice",
							"id":          "user-1",
						},
					},
					"body": map[string]any{
						"contentType": "text",
						"content":     "Hello team!",
					},
				},
				{
					"id":              "msg-2",
					"createdDateTime": "2024-01-15T09:05:00Z",
					"from": map[string]any{
						"user": map[string]any{
							"displayName": "Bob",
							"id":          "user-2",
						},
					},
					"body": map[string]any{
						"contentType": "text",
						"content":     "Hey Alice!",
					},
				},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &listChannelMessagesAction{conn: conn}

	params, _ := json.Marshal(listChannelMessagesParams{
		TeamID:    "team-1",
		ChannelID: "channel-1",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.list_channel_messages",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var summaries []channelMessageSummary
	if err := json.Unmarshal(result.Data, &summaries); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if len(summaries) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(summaries))
	}
	if summaries[0].From != "Alice" {
		t.Errorf("expected from 'Alice', got %q", summaries[0].From)
	}
	if summaries[0].Content != "Hello team!" {
		t.Errorf("expected content 'Hello team!', got %q", summaries[0].Content)
	}
}

func TestListChannelMessages_MissingTeamID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &listChannelMessagesAction{conn: conn}

	params, _ := json.Marshal(map[string]string{"channel_id": "ch-1"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.list_channel_messages",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing team_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestListChannelMessages_MissingChannelID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &listChannelMessagesAction{conn: conn}

	params, _ := json.Marshal(map[string]string{"team_id": "team-1"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.list_channel_messages",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing channel_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestListChannelMessages_DefaultParams(t *testing.T) {
	t.Parallel()

	params := listChannelMessagesParams{TeamID: "t", ChannelID: "c"}
	params.defaults()
	if params.Top != 20 {
		t.Errorf("expected default top 20, got %d", params.Top)
	}
}

func TestListChannelMessages_TopClamped(t *testing.T) {
	t.Parallel()

	params := listChannelMessagesParams{TeamID: "t", ChannelID: "c", Top: 100}
	params.defaults()
	if params.Top != 50 {
		t.Errorf("expected top clamped to 50, got %d", params.Top)
	}
}

func TestListChannelMessages_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &listChannelMessagesAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.list_channel_messages",
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
