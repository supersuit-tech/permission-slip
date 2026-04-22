package slack

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestSlackNullableText_UnmarshalJSON(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		raw  string
		want string
	}{
		{`null`, ""},
		{`""`, ""},
		{`"hello"`, "hello"},
	} {
		var got slackNullableText
		if err := json.Unmarshal([]byte(tc.raw), &got); err != nil {
			t.Fatalf("Unmarshal(%q): %v", tc.raw, err)
		}
		if got.String() != tc.want {
			t.Errorf("Unmarshal(%q) = %q, want %q", tc.raw, got.String(), tc.want)
		}
	}
}

func TestReadChannelMessages_NullMessageText(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path != "/conversations.history" {
			t.Errorf("unexpected path %s", r.URL.Path)
			return
		}
		_, _ = w.Write([]byte(`{
			"ok": true,
			"messages": [
				{"type":"message","user":"U1","text":null,"ts":"1.1"}
			]
		}`))
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &readChannelMessagesAction{conn: conn}
	params, _ := json.Marshal(readChannelMessagesParams{Channel: "C01234567"})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.read_channel_messages",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var data messagesResult
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if len(data.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(data.Messages))
	}
	if data.Messages[0].Text != "" {
		t.Errorf("expected empty text for null slack text, got %q", data.Messages[0].Text)
	}
}

func TestSearchMessages_NullMatchText(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path != "/search.messages" {
			t.Errorf("unexpected path %s", r.URL.Path)
			return
		}
		_, _ = w.Write([]byte(`{
			"ok": true,
			"messages": {
				"matches": [
					{
						"channel": {"id": "C001", "name": "general"},
						"user": "U001",
						"text": null,
						"ts": "1234567890.123456"
					}
				],
				"paging": {"count": 20, "total": 1, "page": 1, "pages": 1}
			}
		}`))
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &searchMessagesAction{conn: conn}
	params, _ := json.Marshal(searchMessagesParams{Query: "blocks only"})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.search_messages",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var data searchMessagesResult
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if len(data.Matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(data.Matches))
	}
	if data.Matches[0].Text != "" {
		t.Errorf("expected empty text, got %q", data.Matches[0].Text)
	}
}
