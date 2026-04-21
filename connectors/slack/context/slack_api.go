package context

// slackResponse is the common envelope for Slack Web API responses.
type slackResponse struct {
	OK     bool   `json:"ok"`
	Error  string `json:"error,omitempty"`
	Needed string `json:"needed,omitempty"`
}

type paginationMeta struct {
	NextCursor string `json:"next_cursor"`
}

// slackMessage is the subset of a Slack message object used for context.
type slackMessage struct {
	Type     string      `json:"type"`
	User     string      `json:"user,omitempty"`
	BotID    string      `json:"bot_id,omitempty"`
	Text     string      `json:"text"`
	TS       string      `json:"ts"`
	ThreadTS string      `json:"thread_ts,omitempty"`
	Files    []slackFile `json:"files,omitempty"`
}

type slackFile struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
}

type messagesResponse struct {
	slackResponse
	Messages []slackMessage  `json:"messages,omitempty"`
	HasMore  bool            `json:"has_more,omitempty"`
	Meta     *paginationMeta `json:"response_metadata,omitempty"`
}

type authTestResponse struct {
	slackResponse
	URL    string `json:"url"`
	UserID string `json:"user_id"`
}

type conversationsInfoRequest struct {
	Channel string `json:"channel"`
}

type conversationsInfoResponse struct {
	slackResponse
	Channel *struct {
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
	} `json:"channel,omitempty"`
}

type usersInfoResponse struct {
	slackResponse
	User *struct {
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
	} `json:"user,omitempty"`
}

type conversationsOpenRequest struct {
	Users string `json:"users"`
}

type conversationsOpenResponse struct {
	slackResponse
	Channel struct {
		ID string `json:"id"`
	} `json:"channel"`
}

type readThreadRequest struct {
	Channel string `json:"channel"`
	TS      string `json:"ts"`
	Limit   int    `json:"limit,omitempty"`
}

type readChannelHistoryRequest struct {
	Channel   string `json:"channel"`
	Limit     int    `json:"limit,omitempty"`
	Oldest    string `json:"oldest,omitempty"`
	Latest    string `json:"latest,omitempty"`
	Inclusive bool   `json:"inclusive,omitempty"`
	Cursor    string `json:"cursor,omitempty"`
}

// ExtractFiles returns filename and size_bytes for each attached file on a message.
func ExtractFiles(msg slackMessage) []FileMeta {
	if len(msg.Files) == 0 {
		return nil
	}
	out := make([]FileMeta, 0, len(msg.Files))
	for _, f := range msg.Files {
		name := f.Name
		if name == "" {
			name = "file"
		}
		out = append(out, FileMeta{Filename: name, SizeBytes: f.Size})
	}
	return out
}
