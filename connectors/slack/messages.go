package slack

// slackMessage is the Slack API representation of a single message.
// Used by both conversations.history and conversations.replies responses.
type slackMessage struct {
	Type       string            `json:"type"`
	User       string            `json:"user,omitempty"`
	BotID      string            `json:"bot_id,omitempty"`
	Text       slackNullableText `json:"text"`
	TS         string            `json:"ts"`
	ThreadTS   string            `json:"thread_ts,omitempty"`
	ReplyCount int               `json:"reply_count,omitempty"`
}

// messagesResponse is the shared Slack API response shape for endpoints
// that return a list of messages (conversations.history, conversations.replies).
type messagesResponse struct {
	slackResponse
	Messages []slackMessage  `json:"messages,omitempty"`
	HasMore  bool            `json:"has_more,omitempty"`
	Meta     *paginationMeta `json:"response_metadata,omitempty"`
}

// messageSummary is the connector-facing output for a single message.
// Omits the Slack-internal "type" field (always "message") for cleaner output.
type messageSummary struct {
	User       string `json:"user,omitempty"`
	BotID      string `json:"bot_id,omitempty"`
	Text       string `json:"text"`
	TS         string `json:"ts"`
	ThreadTS   string `json:"thread_ts,omitempty"`
	ReplyCount int    `json:"reply_count,omitempty"`
}

// messagesResult is the shared action output for message-list endpoints.
type messagesResult struct {
	Messages   []messageSummary `json:"messages"`
	HasMore    bool             `json:"has_more"`
	NextCursor string           `json:"next_cursor,omitempty"`
}

// toMessagesResult converts a messagesResponse into a messagesResult
// by mapping each slackMessage to a messageSummary and extracting the cursor.
func toMessagesResult(resp *messagesResponse) messagesResult {
	result := messagesResult{
		Messages: make([]messageSummary, 0, len(resp.Messages)),
		HasMore:  resp.HasMore,
	}
	for _, msg := range resp.Messages {
		result.Messages = append(result.Messages, messageSummary{
			User:       msg.User,
			BotID:      msg.BotID,
			Text:       msg.Text.String(),
			TS:         msg.TS,
			ThreadTS:   msg.ThreadTS,
			ReplyCount: msg.ReplyCount,
		})
	}
	if resp.Meta != nil {
		result.NextCursor = resp.Meta.NextCursor
	}
	return result
}
