package context

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestFetchRecentMessages_WindowAndAnchor(t *testing.T) {
	t.Parallel()
	api := newMockAPI()
	now := time.Now().UTC()
	tsOld := fmt.Sprintf("%d.%06d", now.Add(-25*time.Hour).Unix(), 0)
	tsMid := fmt.Sprintf("%d.%06d", now.Add(-1*time.Hour).Unix(), 0)
	tsNew := fmt.Sprintf("%d.%06d", now.Add(-30*time.Minute).Unix(), 0)

	api.postHandlers["conversations.history"] = func(body json.RawMessage) (any, error) {
		return messagesResponse{
			slackResponse: slackResponse{OK: true},
			Messages: []slackMessage{
				{User: "U1", Text: "old", TS: tsOld},
				{User: "U1", Text: "mid", TS: tsMid},
				{User: "U1", Text: "new", TS: tsNew},
			},
		}, nil
	}
	api.getHandlers["users.info"] = func(params map[string]string) (any, error) {
		return usersInfoResponse{
			slackResponse: slackResponse{OK: true},
			User: &struct {
				ID       string `json:"id"`
				Name     string `json:"name"`
				RealName string `json:"real_name"`
				Profile  struct {
					DisplayNameNormalized string `json:"display_name_normalized"`
					RealName              string `json:"real_name"`
					Title                 string `json:"title"`
					ImageOriginal         string `json:"image_original"`
					Image512              string `json:"image_512"`
				} `json:"profile"`
			}{ID: params["user"], Name: "alice"},
		}, nil
	}

	msgs, err := FetchRecentMessages(context.Background(), api, "C1", testSlackCredentials(), nil, RecentMessagesOpts{AnchorTS: tsMid}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 2 {
		t.Fatalf("len=%d want 2 (old filtered)", len(msgs))
	}
	if msgs[0].Text != "mid" || msgs[1].Text != "new" {
		t.Fatalf("order/content: %#v", msgs)
	}
	if msgs[0].Permalink == "" {
		t.Fatal("expected permalink")
	}
}

func TestFetchThread_Truncation(t *testing.T) {
	t.Parallel()
	api := newMockAPI()
	var list []slackMessage
	list = append(list, slackMessage{User: "U1", Text: "parent", TS: "10.0"})
	for i := 1; i <= 25; i++ {
		list = append(list, slackMessage{User: "U1", Text: fmt.Sprintf("r%d", i), TS: fmt.Sprintf("10.%06d", i)})
	}
	api.postHandlers["conversations.replies"] = func(body json.RawMessage) (any, error) {
		return messagesResponse{slackResponse: slackResponse{OK: true}, Messages: list}, nil
	}
	api.getHandlers["users.info"] = func(params map[string]string) (any, error) {
		return usersInfoResponse{
			slackResponse: slackResponse{OK: true},
			User: &struct {
				ID       string `json:"id"`
				Name     string `json:"name"`
				RealName string `json:"real_name"`
				Profile  struct {
					DisplayNameNormalized string `json:"display_name_normalized"`
					RealName              string `json:"real_name"`
					Title                 string `json:"title"`
					ImageOriginal         string `json:"image_original"`
					Image512              string `json:"image_512"`
				} `json:"profile"`
			}{ID: params["user"], Name: "alice"},
		}, nil
	}
	parent, replies, truncated, err := FetchThread(context.Background(), api, "C1", "10.0", testSlackCredentials(), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if parent.Text != "parent" {
		t.Fatalf("parent %q", parent.Text)
	}
	if !truncated || len(replies) != 19 {
		t.Fatalf("truncated=%v len=%d", truncated, len(replies))
	}
}

func TestFetchDMHistory_SelfAndFirstContact(t *testing.T) {
	t.Parallel()
	api := newMockAPI()
	api.authTest.UserID = "U_PEER"

	sent, msgs, ch, err := FetchDMHistory(context.Background(), api, "U_PEER", testSlackCredentials(), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if sent != SentinelSelfDM || msgs != nil || ch != "" {
		t.Fatalf("self-dm: sent=%v msgs=%v ch=%q", sent, msgs, ch)
	}

	api.authTest.UserID = "U_SELF"
	api.postHandlers["conversations.open"] = func(body json.RawMessage) (any, error) {
		return conversationsOpenResponse{slackResponse: slackResponse{OK: true}, Channel: struct {
			ID string `json:"id"`
		}{ID: "D1"}}, nil
	}
	api.postHandlers["conversations.history"] = func(body json.RawMessage) (any, error) {
		return messagesResponse{slackResponse: slackResponse{OK: true}, Messages: nil}, nil
	}
	sent, msgs, ch, err = FetchDMHistory(context.Background(), api, "U_PEER", testSlackCredentials(), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if sent != SentinelFirstContact || len(msgs) != 0 || ch != "D1" {
		t.Fatalf("first contact: sent=%v len(msgs)=%d ch=%q", sent, len(msgs), ch)
	}
}

func TestHandleRateLimit_FromFetch(t *testing.T) {
	t.Parallel()
	api := newMockAPI()
	api.postHandlers["conversations.history"] = func(body json.RawMessage) (any, error) {
		return nil, &connectors.RateLimitError{Message: "nope"}
	}
	_, err := FetchRecentMessages(context.Background(), api, "C1", testSlackCredentials(), nil, RecentMessagesOpts{}, nil)
	sc, ok := HandleRateLimit(err)
	if !ok || sc.ContextScope != ScopeMetadataOnly {
		t.Fatalf("HandleRateLimit after fetch: %v %v", sc, ok)
	}
}
