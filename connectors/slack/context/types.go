// Package context provides Slack approval-context helpers (thread/channel/DM
// surroundings) for the Slack connector. Types align with OpenAPI
// `SlackContext` under ActionContext.details.slack_context.
package context

// ContextScope matches `SlackContext.context_scope` in the API schema.
type ContextScope string

const (
	ScopeThread         ContextScope = "thread"
	ScopeRecentChannel  ContextScope = "recent_channel"
	ScopeRecentDM       ContextScope = "recent_dm"
	ScopeSelfDM         ContextScope = "self_dm"
	ScopeFirstContactDM ContextScope = "first_contact_dm"
	ScopeMetadataOnly   ContextScope = "metadata_only"
)

// SlackContext is the normalized payload for approvers (details.slack_context).
type SlackContext struct {
	Channel        *ChannelMeta     `json:"channel,omitempty"`
	Recipient      *UserRef         `json:"recipient,omitempty"`
	TargetMessage  *ContextMessage  `json:"target_message,omitempty"`
	Thread         *ThreadBlock     `json:"thread,omitempty"`
	RecentMessages []ContextMessage `json:"recent_messages,omitempty"`
	ContextScope   ContextScope     `json:"context_scope"`
	ContextWindow  *ContextWindow   `json:"context_window,omitempty"`
}

// ChannelMeta is channel or conversation metadata.
type ChannelMeta struct {
	ID             string `json:"id"`
	Name           string `json:"name,omitempty"`
	IsPrivate      bool   `json:"is_private,omitempty"`
	IsDM           bool   `json:"is_dm,omitempty"`
	Topic          string `json:"topic,omitempty"`
	Purpose        string `json:"purpose,omitempty"`
	MemberCount    int    `json:"member_count,omitempty"`
	LastActivityAt string `json:"last_activity_at,omitempty"`
	Permalink      string `json:"permalink"`
}

// UserRef is a minimal user display object.
type UserRef struct {
	ID        string `json:"id,omitempty"`
	Name      string `json:"name,omitempty"`
	RealName  string `json:"real_name,omitempty"`
	Title     string `json:"title,omitempty"`
	AvatarURL string `json:"avatar_url,omitempty"`
}

// ContextMessage is a single message in context (resolved text + permalink).
type ContextMessage struct {
	User      *UserRef   `json:"user,omitempty"`
	Text      string     `json:"text"`
	TS        string     `json:"ts"`
	Permalink string     `json:"permalink"`
	IsBot     bool       `json:"is_bot,omitempty"`
	Truncated bool       `json:"truncated,omitempty"`
	Files     []FileMeta `json:"files,omitempty"`
}

// FileMeta is attachment metadata only.
type FileMeta struct {
	Filename  string `json:"filename"`
	SizeBytes int64  `json:"size_bytes"`
}

// ThreadBlock groups a thread parent and replies.
type ThreadBlock struct {
	Parent    *ContextMessage  `json:"parent,omitempty"`
	Replies   []ContextMessage `json:"replies,omitempty"`
	Truncated bool             `json:"truncated,omitempty"`
}

// ContextWindow describes the recent-message window (channel or DM).
type ContextWindow struct {
	MessageCount int  `json:"message_count,omitempty"`
	Hours        int  `json:"hours,omitempty"`
	Truncated    bool `json:"truncated,omitempty"`
}

// DMHistorySentinel is returned by FetchDMHistory for special DM cases.
type DMHistorySentinel int

const (
	// SentinelNone means normal history should be fetched.
	SentinelNone DMHistorySentinel = iota
	// SentinelSelfDM is a DM with the authorizing user (note-to-self).
	SentinelSelfDM
	// SentinelFirstContact is a DM with no prior messages with the peer.
	SentinelFirstContact
)
