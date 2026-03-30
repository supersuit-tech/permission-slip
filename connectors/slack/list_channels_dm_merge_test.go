package slack

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestListChannels_MergesHumanDMNotVisibleToBot(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/users.lookupByEmail":
			json.NewEncoder(w).Encode(map[string]any{
				"ok":   true,
				"user": map[string]string{"id": "U_CALLER"},
			})
		case "/users.conversations":
			if got := r.Header.Get("Authorization"); got != "Bearer xoxp-user" {
				t.Errorf("users.conversations: expected user token, got %q", got)
			}
			json.NewEncoder(w).Encode(map[string]any{
				"ok": true,
				"channels": []map[string]any{
					{
						"id":          "D_HUMAN",
						"user":        "U_OTHER",
						"is_private":  true,
						"num_members": 0,
					},
				},
			})
		case "/conversations.list":
			if got := r.Header.Get("Authorization"); got != "Bearer xoxp-user" {
				t.Errorf("conversations.list: expected user token, got %q", got)
			}
			// User token list may omit a human-to-human DM — only public channel here.
			json.NewEncoder(w).Encode(map[string]any{
				"ok": true,
				"channels": []map[string]any{
					{"id": "C001", "name": "general", "is_private": false, "num_members": 10},
				},
			})
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &listChannelsAction{conn: conn}
	params, _ := json.Marshal(listChannelsParams{Types: "public_channel,im"})
	creds := connectors.NewCredentials(map[string]string{
		"access_token": "xoxp-user",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.list_channels",
		Parameters:  params,
		Credentials: creds,
		UserEmail:   "caller@example.com",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data listChannelsResult
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(data.Channels) != 2 {
		t.Fatalf("expected merged public + DM (2 channels), got %d", len(data.Channels))
	}
	var sawC, sawD bool
	for _, ch := range data.Channels {
		switch ch.ID {
		case "C001":
			sawC = true
		case "D_HUMAN":
			sawD = true
			if ch.User != "U_OTHER" {
				t.Errorf("expected DM user U_OTHER, got %q", ch.User)
			}
		}
	}
	if !sawC || !sawD {
		t.Errorf("expected C001 and D_HUMAN in result, sawC=%v sawD=%v", sawC, sawD)
	}
}
