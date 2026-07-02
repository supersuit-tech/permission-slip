package imessage

// chat mirrors the imsg Chat JSON object.
type chat struct {
	ID                  int      `json:"id"`
	Name                string   `json:"name,omitempty"`
	Identifier          string   `json:"identifier,omitempty"`
	GUID                string   `json:"guid,omitempty"`
	Service             string   `json:"service,omitempty"`
	LastMessageAt       string   `json:"last_message_at,omitempty"`
	DisplayName         string   `json:"display_name,omitempty"`
	ContactName         string   `json:"contact_name,omitempty"`
	IsGroup             bool     `json:"is_group,omitempty"`
	Participants        []string `json:"participants,omitempty"`
	AccountID           string   `json:"account_id,omitempty"`
	AccountLogin        string   `json:"account_login,omitempty"`
	LastAddressedHandle string   `json:"last_addressed_handle,omitempty"`
	UnreadCount         int      `json:"unread_count,omitempty"`
}

// message mirrors the imsg Message JSON object.
type message struct {
	ID                   int          `json:"id"`
	ChatID               int          `json:"chat_id"`
	ChatIdentifier       string       `json:"chat_identifier,omitempty"`
	ChatGUID             string       `json:"chat_guid,omitempty"`
	ChatName             string       `json:"chat_name,omitempty"`
	Participants         []string     `json:"participants,omitempty"`
	IsGroup              bool         `json:"is_group,omitempty"`
	GUID                 string       `json:"guid,omitempty"`
	ReplyToGUID          string       `json:"reply_to_guid,omitempty"`
	ThreadOriginatorGUID string       `json:"thread_originator_guid,omitempty"`
	Sender               string       `json:"sender,omitempty"`
	SenderName           string       `json:"sender_name,omitempty"`
	IsFromMe             bool         `json:"is_from_me,omitempty"`
	Text                 string       `json:"text,omitempty"`
	CreatedAt            string       `json:"created_at,omitempty"`
	Attachments          []attachment `json:"attachments,omitempty"`
	Reactions            []reaction   `json:"reactions,omitempty"`
	AccountID            string       `json:"account_id,omitempty"`
	AccountLogin         string       `json:"account_login,omitempty"`
	IsRead               bool         `json:"is_read,omitempty"`
	DateRead             string       `json:"date_read,omitempty"`
}

type attachment struct {
	Filename          string `json:"filename,omitempty"`
	TransferName      string `json:"transfer_name,omitempty"`
	UTI               string `json:"uti,omitempty"`
	MimeType          string `json:"mime_type,omitempty"`
	TotalBytes        int64  `json:"total_bytes,omitempty"`
	Missing           bool   `json:"missing,omitempty"`
	OriginalPath      string `json:"original_path,omitempty"`
	ConvertedPath     string `json:"converted_path,omitempty"`
	ConvertedMimeType string `json:"converted_mime_type,omitempty"`
}

type reaction struct {
	Type      string `json:"type,omitempty"`
	Emoji     string `json:"emoji,omitempty"`
	Sender    string `json:"sender,omitempty"`
	IsFromMe  bool   `json:"is_from_me,omitempty"`
	ReactedTo string `json:"reacted_to_guid,omitempty"`
}

type messagesHistoryResult struct {
	Messages []message `json:"messages"`
}

type sendResult struct {
	OK   bool   `json:"ok"`
	ID   int    `json:"id,omitempty"`
	GUID string `json:"guid,omitempty"`
}

type accountInfo struct {
	AccountID    string   `json:"account_id,omitempty"`
	AccountLogin string   `json:"account_login,omitempty"`
	Aliases      []string `json:"aliases,omitempty"`
}
