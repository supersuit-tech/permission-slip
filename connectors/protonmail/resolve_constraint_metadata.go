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

// ResolveConstraintMetadata returns verified email metadata for constraint
// matching. Currently supports archive_email with batch and thread expansion.
func (c *ProtonMailConnector) ResolveConstraintMetadata(ctx context.Context, actionType string, params json.RawMessage, creds connectors.Credentials) (map[string]any, error) {
	switch actionType {
	case "protonmail.archive_email":
		return c.resolveArchiveConstraintMetadata(ctx, params, creds)
	default:
		return nil, connectors.ErrConstraintMetadataUnavailable
	}
}

func (c *ProtonMailConnector) resolveArchiveConstraintMetadata(ctx context.Context, params json.RawMessage, creds connectors.Credentials) (map[string]any, error) {
	archiveParams, err := parseArchiveParams(params)
	if err != nil {
		return nil, connectors.ErrConstraintMetadataUnavailable
	}
	if err := validateArchiveParams(archiveParams); err != nil {
		return nil, connectors.ErrConstraintMetadataUnavailable
	}

	uids := archiveParams.MessageIDs
	if includeThreadEnabled(archiveParams.IncludeThread) {
		expanded, expandErr := expandArchiveUIDsForApproval(ctx, c, creds, archiveParams.Folder, archiveParams.MessageIDs)
		if expandErr != nil {
			return nil, fmt.Errorf("%w: thread expansion: %v", connectors.ErrConstraintMetadataUnavailable, expandErr)
		}
		uids = expanded
	}

	metaByUID, err := resolveMessageEnvelopes(ctx, c, creds, archiveParams.Folder, uids, connectors.MailboxUIDValidityFromContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("%w: envelope fetch: %v", connectors.ErrConstraintMetadataUnavailable, err)
	}
	if len(metaByUID) == 0 {
		return nil, connectors.ErrConstraintMetadataUnavailable
	}

	senders := make([]string, 0, len(uids))
	for _, uid := range uids {
		meta, ok := metaByUID[uid]
		if !ok {
			return nil, connectors.ErrConstraintMetadataUnavailable
		}
		sender, ok := primarySenderAddress(meta.From)
		if !ok {
			return nil, connectors.ErrConstraintMetadataUnavailable
		}
		senders = append(senders, sender)
	}

	result := map[string]any{
		"senders": senders,
	}
	if len(senders) == 1 {
		result["sender"] = senders[0]
	}
	return result, nil
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
