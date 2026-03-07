package slack

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// setTopicAction implements connectors.Action for slack.set_topic.
// It updates a channel's topic via POST /conversations.setTopic.
type setTopicAction struct {
	conn *SlackConnector
}

// setTopicParams is the user-facing parameter schema.
type setTopicParams struct {
	Channel string `json:"channel"`
	Topic   string `json:"topic"`
}

func (p *setTopicParams) validate() error {
	if err := validateChannelID(p.Channel); err != nil {
		return err
	}
	if p.Topic == "" {
		return &connectors.ValidationError{Message: "missing required parameter: topic"}
	}
	return nil
}

// setTopicRequest is the Slack API request body for conversations.setTopic.
type setTopicRequest struct {
	Channel string `json:"channel"`
	Topic   string `json:"topic"`
}

type setTopicResponse struct {
	slackResponse
	Channel *setTopicChannelInfo `json:"channel,omitempty"`
}

type setTopicChannelInfo struct {
	ID    string `json:"id"`
	Topic struct {
		Value string `json:"value"`
	} `json:"topic"`
}

// Execute updates a channel's topic and returns the updated topic.
func (a *setTopicAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params setTopicParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	body := setTopicRequest{
		Channel: params.Channel,
		Topic:   params.Topic,
	}

	var resp setTopicResponse
	if err := a.conn.doPost(ctx, "conversations.setTopic", req.Credentials, body, &resp); err != nil {
		return nil, err
	}

	if !resp.OK {
		return nil, mapSlackError(resp.Error)
	}

	topic := ""
	if resp.Channel != nil {
		topic = resp.Channel.Topic.Value
	}

	return connectors.JSONResult(map[string]string{
		"channel": params.Channel,
		"topic":   topic,
	})
}
