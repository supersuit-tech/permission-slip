package slack

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Integration-style coverage for issue #981 PR 3: ResolveResourceDetails returns
// slack_context for all secondary Slack actions (mocked Slack API).
func TestResolveResourceDetails_SecondaryActions_SlackContext(t *testing.T) {
	t.Parallel()

	tsMsg := "100.000001"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/auth.test":
			json.NewEncoder(w).Encode(map[string]any{"ok": true, "url": "https://acme.slack.com/", "user_id": "U_SELF"})
		case "/conversations.info":
			json.NewEncoder(w).Encode(map[string]any{
				"ok": true,
				"channel": map[string]any{
					"id":          "C01234567",
					"name":        "general",
					"num_members": 2,
				},
			})
		case "/conversations.history":
			var body struct {
				Latest    string `json:"latest"`
				Inclusive bool   `json:"inclusive"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body.Latest != "" && body.Inclusive {
				json.NewEncoder(w).Encode(map[string]any{
					"ok": true,
					"messages": []map[string]any{
						{"user": "U1", "text": "hi", "ts": tsMsg},
					},
				})
				return
			}
			json.NewEncoder(w).Encode(map[string]any{
				"ok": true,
				"messages": []map[string]any{
					{"user": "U0", "text": "before", "ts": "99.000001"},
					{"user": "U1", "text": "hi", "ts": tsMsg},
				},
			})
		case "/users.info":
			json.NewEncoder(w).Encode(map[string]any{
				"ok": true,
				"user": map[string]any{
					"id":   "U111",
					"name": "invitee",
				},
			})
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	ctx := context.Background()
	creds := validCreds()

	cases := []struct {
		action string
		params string
	}{
		{"slack.add_reaction", `{"channel":"C01234567","timestamp":"` + tsMsg + `","name":"thumbsup"}`},
		{"slack.remove_reaction", `{"channel":"C01234567","timestamp":"` + tsMsg + `","name":"thumbsup"}`},
		{"slack.pin_message", `{"channel":"C01234567","ts":"` + tsMsg + `"}`},
		{"slack.unpin_message", `{"channel":"C01234567","ts":"` + tsMsg + `"}`},
		{"slack.archive_channel", `{"channel":"C01234567"}`},
		{"slack.invite_to_channel", `{"channel":"C01234567","users":"U111"}`},
		{"slack.remove_from_channel", `{"channel":"C01234567","user":"U111"}`},
	}

	for _, tc := range cases {
		t.Run(tc.action, func(t *testing.T) {
			t.Parallel()
			details, err := conn.ResolveResourceDetails(ctx, tc.action, json.RawMessage(tc.params), creds)
			if err != nil {
				t.Fatalf("ResolveResourceDetails: %v", err)
			}
			sc, ok := details["slack_context"].(map[string]any)
			if !ok || sc == nil {
				t.Fatalf("missing slack_context in %+v", details)
			}
			if _, ok := sc["context_scope"]; !ok {
				t.Fatalf("context_scope missing: %+v", sc)
			}
		})
	}
}
