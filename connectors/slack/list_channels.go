package slack

import (
	"context"
	"strings"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// listChannelsAction implements connectors.Action for slack.list_channels.
// It lists channels visible to the bot via POST /conversations.list.
type listChannelsAction struct {
	conn *SlackConnector
}

// listChannelsParams defines the user-facing parameter schema.
type listChannelsParams struct {
	// Types filters by channel type. Comma-separated list of:
	// public_channel, private_channel, mpim, im.
	// Defaults to all types: public_channel,private_channel,mpim,im.
	Types string `json:"types,omitempty"`
	// Limit is the max number of channels to return (1-1000, default 100).
	Limit int `json:"limit,omitempty"`
	// Cursor is a pagination cursor from a previous response.
	Cursor string `json:"cursor,omitempty"`
	// ExcludeArchived filters out archived channels. Defaults to true.
	ExcludeArchived *bool `json:"exclude_archived,omitempty"`
}

func (p *listChannelsParams) validate() error {
	return validateLimit(p.Limit)
}

// listChannelsRequest is the Slack API request body for conversations.list.
type listChannelsRequest struct {
	Types           string `json:"types,omitempty"`
	Limit           int    `json:"limit,omitempty"`
	Cursor          string `json:"cursor,omitempty"`
	ExcludeArchived bool   `json:"exclude_archived"`
}

type listChannelsResponse struct {
	slackResponse
	Channels []listChannelEntry `json:"channels,omitempty"`
	Meta     *paginationMeta    `json:"response_metadata,omitempty"`
}

// listChannelEntry maps the Slack API response for a single channel from
// conversations.list. IM channels (DMs) omit Name and instead populate User
// with the other participant's Slack user ID.
type listChannelEntry struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	User       string `json:"user,omitempty"`
	IsPrivate  bool   `json:"is_private"`
	IsIM       bool   `json:"is_im"`
	IsMPIM     bool   `json:"is_mpim"`
	IsArchived bool   `json:"is_archived"`
	NumMembers int    `json:"num_members"`
	Topic      struct {
		Value string `json:"value"`
	} `json:"topic"`
	Purpose struct {
		Value string `json:"value"`
	} `json:"purpose"`
}

// listChannelsResult is the action output.
type listChannelsResult struct {
	Channels   []listChannelSummary `json:"channels"`
	NextCursor string               `json:"next_cursor,omitempty"`
}

// listChannelSummary is the user-facing output for a single channel. For IM
// channels, Name is empty and User contains the other participant's Slack user
// ID. Both fields use omitempty so the JSON output only includes relevant fields.
type listChannelSummary struct {
	ID         string `json:"id"`
	Name       string `json:"name,omitempty"`
	User       string `json:"user,omitempty"`
	IsPrivate  bool   `json:"is_private"`
	IsIM       bool   `json:"is_im,omitempty"`
	IsMPIM     bool   `json:"is_mpim,omitempty"`
	Topic      string `json:"topic,omitempty"`
	Purpose    string `json:"purpose,omitempty"`
	NumMembers int    `json:"num_members"`
}

// Execute lists Slack channels for the authorizing user's OAuth token via
// conversations.list, with users.conversations merged in to fill gaps (e.g.
// human DMs missing from conversations.list). Results are filtered to the
// requested types even when Slack mis-honors the types parameter (#1028).
func (a *listChannelsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params listChannelsParams
	if err := parseAndValidate(req.Parameters, &params); err != nil {
		return nil, err
	}

	result, err := a.conn.listChannelsMerged(ctx, req.Credentials, req.UserEmail, params)
	if err != nil {
		return nil, err
	}
	return connectors.JSONResult(*result)
}

// listChannelEntryMatchesTypes returns whether ch should be included for the
// given comma-separated conversations.list types string.
func listChannelEntryMatchesTypes(types string, ch listChannelEntry) bool {
	for _, raw := range strings.Split(types, ",") {
		t := strings.TrimSpace(raw)
		switch t {
		case "im":
			if ch.IsIM || (len(ch.ID) > 0 && ch.ID[0] == 'D') {
				return true
			}
		case "mpim":
			if ch.IsMPIM {
				return true
			}
			// Fallback if API omits is_mpim: treat G-prefix non-DM, non-private-channel
			// as mpim. Pre-2020 workspaces used G-prefix for private channels too, so
			// we exclude those by checking !ch.IsPrivate (real MPIMs aren't marked private).
			if len(ch.ID) > 0 && ch.ID[0] == 'G' && !ch.IsIM && !ch.IsPrivate {
				return true
			}
		case "private_channel":
			// Private channels (C or legacy G) but not DMs or MPIMs — avoids treating
			// is_private DMs as private_channel when only private_channel was requested.
			if ch.IsIM || ch.IsMPIM {
				break
			}
			if ch.IsPrivate && len(ch.ID) > 0 && (ch.ID[0] == 'C' || ch.ID[0] == 'G') {
				return true
			}
		case "public_channel":
			// Public C-channels only — exclude DMs/MPIMs (which may carry is_private)
			// and exclude private channels (C or legacy G with is_private=true).
			if ch.IsIM || ch.IsMPIM {
				break
			}
			if !ch.IsPrivate && len(ch.ID) > 0 && ch.ID[0] == 'C' {
				return true
			}
		}
	}
	return false
}
