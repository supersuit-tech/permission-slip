package protonmail

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/supersuit-tech/permission-slip/connectors"
)

var angleAddrRe = regexp.MustCompile(`<([^>]+)>`)

// protonmailMetadataActions lists UID-targeted actions whose verified envelope
// metadata can be resolved for $meta constraint matching.
var protonmailMetadataActions = map[string]struct{}{
	"protonmail.read_email":          {},
	"protonmail.download_attachment": {},
	"protonmail.reply_email":         {},
	"protonmail.archive_email":       {},
	"protonmail.mark_read":           {},
	"protonmail.mark_unread":         {},
	"protonmail.flag":                {},
	"protonmail.unflag":              {},
	"protonmail.move_to_folder":      {},
	"protonmail.delete":              {},
	"protonmail.apply_label":         {},
	"protonmail.remove_label":        {},
}

// protonmailMetaConstraintFields are valid $meta keys for supported actions.
var protonmailMetaConstraintFields = []string{"from", "sender", "senders", "to", "cc", "bcc"}

// ResolveConstraintMetadata returns verified email metadata for constraint
// matching on UID-targeted actions (from/to/cc/bcc per message).
func (c *ProtonMailConnector) ResolveConstraintMetadata(ctx context.Context, actionType string, params json.RawMessage, creds connectors.Credentials) (map[string]any, error) {
	if _, ok := protonmailMetadataActions[actionType]; !ok {
		return nil, connectors.ErrConstraintMetadataUnavailable
	}
	return c.resolveMessageConstraintMetadata(ctx, actionType, params, creds)
}

// ConstraintMetadataActionSupport reports which $meta fields are valid per action.
func (c *ProtonMailConnector) ConstraintMetadataActionSupport(actionType string) ([]string, bool) {
	if _, ok := protonmailMetadataActions[actionType]; !ok {
		return nil, false
	}
	return protonmailMetaConstraintFields, true
}

type messageConstraintTarget struct {
	folder string
	uids   []uint32
}

func (c *ProtonMailConnector) resolveMessageConstraintMetadata(ctx context.Context, actionType string, params json.RawMessage, creds connectors.Credentials) (map[string]any, error) {
	target, err := parseMessageConstraintTarget(actionType, params)
	if err != nil {
		return nil, connectors.ErrConstraintMetadataUnavailable
	}

	uids := target.uids
	if shouldExpandThreadForMetadata(actionType, params) {
		expanded, expandErr := expandArchiveUIDsForApproval(ctx, c, creds, target.folder, target.uids)
		if expandErr != nil {
			return nil, fmt.Errorf("%w: thread expansion: %v", connectors.ErrConstraintMetadataUnavailable, expandErr)
		}
		uids = expanded
	}

	metaByUID, err := resolveMessageEnvelopes(ctx, c, creds, target.folder, uids, connectors.MailboxUIDValidityFromContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("%w: envelope fetch: %v", connectors.ErrConstraintMetadataUnavailable, err)
	}
	if len(metaByUID) == 0 {
		return nil, connectors.ErrConstraintMetadataUnavailable
	}

	entries := make([]messageMetaEntry, 0, len(uids))
	for _, uid := range uids {
		meta, ok := metaByUID[uid]
		if !ok {
			return nil, connectors.ErrConstraintMetadataUnavailable
		}
		entry, ok := envelopeToMessageMeta(meta)
		if !ok {
			return nil, connectors.ErrConstraintMetadataUnavailable
		}
		entries = append(entries, entry)
	}

	return buildConstraintMetadataResult(entries), nil
}

func parseMessageConstraintTarget(actionType string, params json.RawMessage) (*messageConstraintTarget, error) {
	switch actionType {
	case "protonmail.read_email":
		var p readEmailParams
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		if err := p.validate(); err != nil {
			return nil, err
		}
		return &messageConstraintTarget{folder: p.Folder, uids: []uint32{p.MessageID}}, nil

	case "protonmail.download_attachment":
		var p downloadAttachmentParams
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		if err := p.validate(); err != nil {
			return nil, err
		}
		return &messageConstraintTarget{folder: p.Folder, uids: []uint32{p.MessageID}}, nil

	case "protonmail.reply_email":
		var p struct {
			InReplyToMessageID uint32 `json:"in_reply_to_message_id"`
			Folder             string `json:"folder"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		if p.InReplyToMessageID == 0 {
			return nil, fmt.Errorf("missing in_reply_to_message_id")
		}
		if p.Folder == "" {
			p.Folder = "INBOX"
		}
		return &messageConstraintTarget{folder: p.Folder, uids: []uint32{p.InReplyToMessageID}}, nil

	case "protonmail.archive_email":
		archiveParams, err := parseArchiveParams(params)
		if err != nil {
			return nil, err
		}
		if err := validateArchiveParams(archiveParams); err != nil {
			return nil, err
		}
		return &messageConstraintTarget{folder: archiveParams.Folder, uids: archiveParams.MessageIDs}, nil

	case "protonmail.apply_label", "protonmail.remove_label":
		labelParams, err := parseLabelMessageParams(params)
		if err != nil {
			return nil, err
		}
		if err := labelParams.validate(); err != nil {
			return nil, err
		}
		return &messageConstraintTarget{folder: labelParams.Folder, uids: labelParams.MessageIDs}, nil

	default:
		uidParams, err := parseUIDMessageParams(params)
		if err != nil {
			return nil, err
		}
		if err := uidParams.validate(); err != nil {
			return nil, err
		}
		return &messageConstraintTarget{folder: uidParams.Folder, uids: uidParams.MessageIDs}, nil
	}
}

func shouldExpandThreadForMetadata(actionType string, params json.RawMessage) bool {
	switch actionType {
	case "protonmail.archive_email":
		archiveParams, err := parseArchiveParams(params)
		if err != nil {
			return false
		}
		return includeThreadEnabled(archiveParams.IncludeThread)
	case "protonmail.apply_label", "protonmail.remove_label":
		labelParams, err := parseLabelMessageParams(params)
		if err != nil {
			return false
		}
		return includeThreadEnabled(labelParams.IncludeThread)
	default:
		return false
	}
}

type messageMetaEntry struct {
	From string
	To   []string
	Cc   []string
	Bcc  []string
}

func envelopeToMessageMeta(meta emailEnvelopeMetadata) (messageMetaEntry, bool) {
	from, ok := primarySenderAddress(meta.From)
	if !ok {
		return messageMetaEntry{}, false
	}
	return messageMetaEntry{
		From: from,
		To:   normalizeAddressList(meta.To),
		Cc:   normalizeAddressList(meta.Cc),
		Bcc:  normalizeAddressList(meta.Bcc),
	}, true
}

func normalizeAddressList(addrs []string) []string {
	if len(addrs) == 0 {
		return nil
	}
	out := make([]string, 0, len(addrs))
	for _, raw := range addrs {
		if normalized, ok := normalizeEmailAddress(raw); ok {
			out = append(out, normalized)
		}
	}
	return out
}

func buildConstraintMetadataResult(entries []messageMetaEntry) map[string]any {
	messages := make([]map[string]any, len(entries))
	senders := make([]string, len(entries))
	for i, entry := range entries {
		messages[i] = map[string]any{
			"from": entry.From,
			"to":   entry.To,
			"cc":   entry.Cc,
			"bcc":  entry.Bcc,
		}
		senders[i] = entry.From
	}

	result := map[string]any{
		"messages": messages,
		"senders":  senders,
	}
	if len(senders) == 1 {
		result["sender"] = senders[0]
	}
	return result
}

// primarySenderAddress extracts the first From address as a bare email suitable
// for pattern matching (strips display names and angle-bracket formatting).
func primarySenderAddress(from []string) (string, bool) {
	if len(from) == 0 {
		return "", false
	}
	return normalizeEmailAddress(from[0])
}

func normalizeEmailAddress(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", false
	}
	if matches := angleAddrRe.FindStringSubmatch(raw); len(matches) == 2 {
		return strings.TrimSpace(matches[1]), true
	}
	return raw, true
}

var _ connectors.ConstraintMetadataResolver = (*ProtonMailConnector)(nil)
var _ connectors.ConstraintMetadataCapabilities = (*ProtonMailConnector)(nil)
