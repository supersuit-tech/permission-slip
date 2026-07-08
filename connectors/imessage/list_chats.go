package imessage

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/supersuit-tech/permission-slip/connectors"
)

type listChatsAction struct {
	conn *IMessageConnector
}

type listChatsParams struct {
	Limit      int    `json:"limit"`
	UnreadOnly bool   `json:"unread_only"`
	Since      string `json:"since"`
	Before     string `json:"before"`
	OrderBy    string `json:"order_by"`
	Sort       string `json:"sort"`
}

func (p *listChatsParams) validate() error {
	if p.Limit <= 0 {
		p.Limit = 20
	}
	if p.Limit > 100 {
		return &connectors.ValidationError{Message: "limit must be at most 100"}
	}

	var sinceAt, beforeAt time.Time
	var err error
	if p.Since != "" {
		sinceAt, err = parseRFC3339Timestamp(p.Since)
		if err != nil {
			return &connectors.ValidationError{Message: fmt.Sprintf("since must be RFC 3339 format: %v", err)}
		}
	}
	if p.Before != "" {
		beforeAt, err = parseRFC3339Timestamp(p.Before)
		if err != nil {
			return &connectors.ValidationError{Message: fmt.Sprintf("before must be RFC 3339 format: %v", err)}
		}
	}
	if p.Since != "" && p.Before != "" && !beforeAt.After(sinceAt) {
		return &connectors.ValidationError{Message: "before must be after since"}
	}

	if p.OrderBy == "" {
		p.OrderBy = "last_activity"
	}
	switch p.OrderBy {
	case "last_activity", "contact_name":
	default:
		return &connectors.ValidationError{Message: fmt.Sprintf("invalid order_by: %q; must be one of: last_activity, contact_name", p.OrderBy)}
	}

	if p.Sort == "" {
		p.Sort = "desc"
	}
	switch p.Sort {
	case "asc", "desc":
	default:
		return &connectors.ValidationError{Message: fmt.Sprintf("invalid sort: %q; must be one of: asc, desc", p.Sort)}
	}

	return nil
}

func (a *listChatsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params listChatsParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: "invalid parameters: " + err.Error()}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	actionCtx, cancel := a.conn.actionTimeout(ctx)
	defer cancel()

	fetchLimit := params.Limit
	if params.UnreadOnly || params.Since != "" || params.Before != "" || params.OrderBy != "" {
		// Over-fetch when sorting or filtering client-side (imsg chats.list has no native
		// sort/date filters) so we can return the correct top-N after post-processing.
		fetchLimit = params.Limit * 5
		if fetchLimit > 100 {
			fetchLimit = 100
		}
	}

	rpcParams := map[string]any{
		"limit": fetchLimit,
	}
	if params.UnreadOnly {
		rpcParams["unread_only"] = true
	}

	var result chatsListResult
	if err := a.conn.client.rpcCall(actionCtx, req.Credentials, "chats.list", rpcParams, &result); err != nil {
		return nil, err
	}
	chats := result.Chats
	if chats == nil {
		chats = []chat{}
	}
	sortChats(chats, params.OrderBy, params.Sort)
	if params.Since != "" || params.Before != "" {
		chats = filterChatsByActivity(chats, params.Since, params.Before, params.Limit)
	}
	if params.UnreadOnly {
		chats = filterUnreadChats(chats, params.Limit)
	}
	if params.Since == "" && params.Before == "" && !params.UnreadOnly {
		chats = trimChats(chats, params.Limit)
	}
	return connectors.JSONResult(map[string]any{"chats": chats})
}

func sortChats(chats []chat, orderBy, sortDir string) {
	desc := sortDir != "asc"
	sort.SliceStable(chats, func(i, j int) bool {
		switch orderBy {
		case "contact_name":
			left := chatContactSortKey(&chats[i])
			right := chatContactSortKey(&chats[j])
			if desc {
				return left > right
			}
			return left < right
		default:
			leftAt, leftOK := chatLastActivityTime(&chats[i])
			rightAt, rightOK := chatLastActivityTime(&chats[j])
			if !leftOK && !rightOK {
				return false
			}
			if !leftOK {
				return !desc
			}
			if !rightOK {
				return desc
			}
			if desc {
				return leftAt.After(rightAt)
			}
			return leftAt.Before(rightAt)
		}
	})
}

func chatContactSortKey(ch *chat) string {
	if ch == nil {
		return ""
	}
	for _, name := range []string{ch.ContactName, ch.DisplayName, ch.Name, ch.Identifier} {
		if trimmed := strings.TrimSpace(name); trimmed != "" {
			return strings.ToLower(trimmed)
		}
	}
	return ""
}

func chatLastActivityTime(ch *chat) (time.Time, bool) {
	if ch == nil || ch.LastMessageAt == "" {
		return time.Time{}, false
	}
	at, err := parseRFC3339Timestamp(ch.LastMessageAt)
	if err != nil {
		return time.Time{}, false
	}
	return at, true
}

func trimChats(chats []chat, limit int) []chat {
	if limit <= 0 || len(chats) <= limit {
		return chats
	}
	return chats[:limit]
}

// filterChatsByActivity keeps chats whose last_message_at falls within the optional
// since/before window, capped at limit. Chats without last_message_at are excluded
// because their activity date cannot be verified.
func filterChatsByActivity(chats []chat, since, before string, limit int) []chat {
	var sinceAt, beforeAt *time.Time
	if since != "" {
		t, err := parseRFC3339Timestamp(since)
		if err != nil {
			return []chat{}
		}
		sinceAt = &t
	}
	if before != "" {
		t, err := parseRFC3339Timestamp(before)
		if err != nil {
			return []chat{}
		}
		beforeAt = &t
	}

	out := make([]chat, 0, limit)
	for _, c := range chats {
		if c.LastMessageAt == "" {
			continue
		}
		at, err := parseRFC3339Timestamp(c.LastMessageAt)
		if err != nil {
			continue
		}
		if sinceAt != nil && at.Before(*sinceAt) {
			continue
		}
		if beforeAt != nil && !at.Before(*beforeAt) {
			continue
		}
		out = append(out, c)
		if len(out) >= limit {
			break
		}
	}
	return out
}

// filterUnreadChats keeps chats with unread_count > 0, capped at limit.
// When imsg omits unread_count (older builds), the zero value is treated as no unreads.
func filterUnreadChats(chats []chat, limit int) []chat {
	out := make([]chat, 0, limit)
	for _, c := range chats {
		if c.UnreadCount <= 0 {
			continue
		}
		out = append(out, c)
		if len(out) >= limit {
			break
		}
	}
	return out
}

func parseRFC3339Timestamp(s string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t, err = time.Parse(time.RFC3339Nano, s)
	}
	return t, err
}
