package microsoft

import (
	"context"
	"encoding/json"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// ResolveResourceDetails fetches human-readable metadata for Microsoft action parameters.
func (c *MicrosoftConnector) ResolveResourceDetails(ctx context.Context, actionType string, params json.RawMessage, creds connectors.Credentials) (map[string]any, error) {
	switch actionType {
	case "microsoft.send_email_reply":
		return c.resolveSendEmailReply(ctx, creds, params)
	default:
		return nil, nil
	}
}

func (c *MicrosoftConnector) resolveSendEmailReply(ctx context.Context, creds connectors.Credentials, params json.RawMessage) (map[string]any, error) {
	var p struct {
		MessageID string `json:"message_id"`
	}
	if err := json.Unmarshal(params, &p); err != nil || p.MessageID == "" {
		return nil, nil
	}
	msg, err := c.fetchGraphMessage(ctx, creds, p.MessageID)
	if err != nil {
		return nil, err
	}
	details := map[string]any{
		"subject": stripGraphHeaderNewlines(msg.Subject),
	}
	if msg.From != nil {
		details["from"] = stripGraphHeaderNewlines(msg.From.EmailAddress.Address)
	}
	thread, err := c.buildMicrosoftEmailThread(ctx, creds, p.MessageID)
	if err != nil {
		return details, nil
	}
	extra := connectors.EmailThreadDetailsMap(thread)
	if extra == nil {
		return details, nil
	}
	for k, v := range extra {
		details[k] = v
	}
	return details, nil
}
