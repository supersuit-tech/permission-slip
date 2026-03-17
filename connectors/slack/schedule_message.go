package slack

import (
	"context"
	"fmt"
	"time"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// scheduleMessageAction implements connectors.Action for slack.schedule_message.
// It schedules a message for future delivery via POST /chat.scheduleMessage.
type scheduleMessageAction struct {
	conn *SlackConnector
}

// scheduleMessageParams is the user-facing parameter schema.
type scheduleMessageParams struct {
	Channel string `json:"channel"`
	Message string `json:"message"`
	PostAt  string `json:"post_at"`
}

func (p *scheduleMessageParams) validate() error {
	if p.Channel == "" {
		return &connectors.ValidationError{Message: "missing required parameter: channel"}
	}
	if p.Message == "" {
		return &connectors.ValidationError{Message: "missing required parameter: message"}
	}
	if p.PostAt == "" {
		return &connectors.ValidationError{Message: "missing required parameter: post_at"}
	}
	return nil
}

// postAtUnix parses the post_at field as RFC 3339 and returns the Unix
// timestamp. Returns a ValidationError if the value is unparseable or in the
// past.
func (p *scheduleMessageParams) postAtUnix() (int64, error) {
	// Try RFC 3339 first; also accept "datetime-local" format (no seconds, no TZ)
	// emitted by HTML <input type="datetime-local">, treating it as UTC.
	t, err := time.Parse(time.RFC3339Nano, p.PostAt)
	if err != nil {
		// Also accept "datetime-local" formats (no TZ) emitted by HTML
		// <input type="datetime-local">, treating them as UTC.
		for _, layout := range []string{"2006-01-02T15:04:05", "2006-01-02T15:04"} {
			if t2, err2 := time.Parse(layout, p.PostAt); err2 == nil {
				t = t2.UTC()
				err = nil
				break
			}
		}
		if err != nil {
			return 0, &connectors.ValidationError{
				Message: fmt.Sprintf("post_at must be a valid RFC 3339 datetime (e.g. 2026-03-20T09:00:00Z), got %q", p.PostAt),
			}
		}
	}
	unix := t.Unix()
	if unix <= time.Now().Unix() {
		return 0, &connectors.ValidationError{
			Message: fmt.Sprintf("post_at must be in the future (got %s)", p.PostAt),
		}
	}
	return unix, nil
}

// scheduleMessageRequest is the Slack API request body for chat.scheduleMessage.
type scheduleMessageRequest struct {
	Channel string `json:"channel"`
	Text    string `json:"text"`
	PostAt  int64  `json:"post_at"`
}

type scheduleMessageResponse struct {
	slackResponse
	ScheduledMessageID string `json:"scheduled_message_id,omitempty"`
	PostAt             int64  `json:"post_at,omitempty"`
	Channel            string `json:"channel,omitempty"`
}

// Execute schedules a message for future delivery and returns the scheduled message metadata.
func (a *scheduleMessageAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params scheduleMessageParams
	if err := parseAndValidate(req.Parameters, &params); err != nil {
		return nil, err
	}

	postAtUnix, err := params.postAtUnix()
	if err != nil {
		return nil, err
	}

	body := scheduleMessageRequest{
		Channel: params.Channel,
		Text:    params.Message,
		PostAt:  postAtUnix,
	}

	var resp scheduleMessageResponse
	if err := a.conn.doPost(ctx, "chat.scheduleMessage", req.Credentials, body, &resp); err != nil {
		return nil, err
	}

	if !resp.OK {
		return nil, mapSlackError(resp.Error)
	}

	return connectors.JSONResult(map[string]any{
		"scheduled_message_id": resp.ScheduledMessageID,
		"post_at":              resp.PostAt,
		"channel":              resp.Channel,
	})
}
