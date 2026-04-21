package context

import (
	"context"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// FetchUserRef loads display fields for a Slack user ID (users.info).
func FetchUserRef(ctx context.Context, api SlackAPI, creds connectors.Credentials, userID string) (*UserRef, error) {
	return fetchUserRef(ctx, api, creds, userID)
}

func fetchUserRef(ctx context.Context, api SlackAPI, creds connectors.Credentials, userID string) (*UserRef, error) {
	if userID == "" {
		return nil, nil
	}
	var resp usersInfoResponse
	if err := api.Get(ctx, "users.info", creds, map[string]string{"user": userID}, &resp); err != nil {
		return nil, err
	}
	if !resp.OK {
		return nil, mapSlackErr(resp.Error)
	}
	if resp.User == nil {
		return nil, &connectors.ExternalError{StatusCode: 200, Message: "Slack users.info returned ok=true but no user"}
	}
	u := resp.User
	ref := &UserRef{
		ID:   u.ID,
		Name: u.Name,
	}
	if u.RealName != "" {
		ref.RealName = u.RealName
	} else if u.Profile.RealName != "" {
		ref.RealName = u.Profile.RealName
	}
	if u.Profile.Title != "" {
		ref.Title = u.Profile.Title
	}
	if u.Profile.ImageOriginal != "" {
		ref.AvatarURL = u.Profile.ImageOriginal
	} else if u.Profile.Image512 != "" {
		ref.AvatarURL = u.Profile.Image512
	}
	return ref, nil
}
