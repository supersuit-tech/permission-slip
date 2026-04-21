package context

import (
	"context"
	"encoding/json"
	"testing"
)

func TestResolveMentions(t *testing.T) {
	t.Parallel()
	api := newMockAPI()
	api.getHandlers["users.info"] = func(params map[string]string) (any, error) {
		if params["user"] == "U1" {
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
				}{
					ID:   "U1",
					Name: "alice",
				},
			}, nil
		}
		return usersInfoResponse{slackResponse: slackResponse{OK: false, Error: "user_not_found"}}, nil
	}
	api.postHandlers["conversations.info"] = func(body json.RawMessage) (any, error) {
		var req conversationsInfoRequest
		_ = json.Unmarshal(body, &req)
		if req.Channel == "C9" {
			return conversationsInfoResponse{
				slackResponse: slackResponse{OK: true},
				Channel: &struct {
					ID         string `json:"id"`
					Name       string `json:"name"`
					IsPrivate  bool   `json:"is_private"`
					IsIM       bool   `json:"is_im"`
					IsMPIM     bool   `json:"is_mpim"`
					NumMembers int    `json:"num_members"`
					Topic      struct {
						Value string `json:"value"`
					} `json:"topic"`
					Purpose struct {
						Value string `json:"value"`
					} `json:"purpose"`
				}{ID: "C9", Name: "general"},
			}, nil
		}
		return conversationsInfoResponse{slackResponse: slackResponse{OK: false, Error: "channel_not_found"}}, nil
	}

	in := "Hi <@U1> in <#C9|ignored> <!here>"
	out, err := ResolveMentions(context.Background(), in, api, testSlackCredentials(), &MentionCache{})
	if err != nil {
		t.Fatal(err)
	}
	if out != "Hi @alice in #ignored @here" {
		t.Fatalf("got %q", out)
	}
}
