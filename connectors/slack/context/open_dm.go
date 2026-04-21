package context

import (
	"context"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// OpenIMChannel opens (or reuses) a DM with the given user and returns the IM channel ID.
func OpenIMChannel(ctx context.Context, api SlackAPI, userID string, creds connectors.Credentials) (string, error) {
	if userID == "" {
		return "", &connectors.ValidationError{Message: "missing user id for DM open"}
	}
	openBody := conversationsOpenRequest{Users: userID}
	var openResp conversationsOpenResponse
	if err := api.Post(ctx, "conversations.open", creds, openBody, &openResp); err != nil {
		return "", err
	}
	if !openResp.OK {
		return "", mapSlackErr(openResp.Error)
	}
	return openResp.Channel.ID, nil
}
