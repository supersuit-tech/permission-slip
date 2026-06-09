package protonmail

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/supersuit-tech/permission-slip/connectors"
)

// resolveMessageEnvelopes is the IMAP-backed implementation used by
// ResolveResourceDetails. Tests may replace it to avoid a live proxy.
var resolveMessageEnvelopes = resolveMessageEnvelopesIMAP

// ResolveResourceDetails fetches human-readable email metadata for approval
// prompts. Only subject, from, to, and date are returned. Errors are non-fatal
// for the approval flow — callers store the approval without details on failure.
func (c *ProtonMailConnector) ResolveResourceDetails(ctx context.Context, actionType string, params json.RawMessage, creds connectors.Credentials) (map[string]any, error) {
	switch actionType {
	case "protonmail.read_email":
		return c.resolveReadEmailDetails(ctx, params, creds)
	case "protonmail.archive_email":
		return c.resolveArchiveEmailDetails(ctx, params, creds)
	case "protonmail.reply_email":
		return c.resolveReplyEmailDetails(ctx, params, creds)
	default:
		return nil, nil
	}
}

func (c *ProtonMailConnector) resolveReadEmailDetails(ctx context.Context, params json.RawMessage, creds connectors.Credentials) (map[string]any, error) {
	var p readEmailParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, nil
	}
	if err := p.validate(); err != nil {
		return nil, nil
	}

	metaByUID, err := resolveMessageEnvelopes(ctx, c, creds, p.Folder, []uint32{p.MessageID}, connectors.MailboxUIDValidityFromContext(ctx))
	if err != nil || len(metaByUID) == 0 {
		return nil, nil
	}
	meta, ok := metaByUID[p.MessageID]
	if !ok {
		return nil, nil
	}
	return meta.asMap(), nil
}

func (c *ProtonMailConnector) resolveReplyEmailDetails(ctx context.Context, params json.RawMessage, creds connectors.Credentials) (map[string]any, error) {
	var p struct {
		InReplyToMessageID uint32 `json:"in_reply_to_message_id"`
		Folder             string `json:"folder"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, nil
	}
	if p.InReplyToMessageID == 0 {
		return nil, nil
	}
	if p.Folder == "" {
		p.Folder = "INBOX"
	}

	metaByUID, err := resolveMessageEnvelopes(ctx, c, creds, p.Folder, []uint32{p.InReplyToMessageID}, connectors.MailboxUIDValidityFromContext(ctx))
	if err != nil || len(metaByUID) == 0 {
		return nil, nil
	}
	meta, ok := metaByUID[p.InReplyToMessageID]
	if !ok {
		return nil, nil
	}
	return map[string]any{"in_reply_to": meta.asMap()}, nil
}

func (c *ProtonMailConnector) resolveArchiveEmailDetails(ctx context.Context, params json.RawMessage, creds connectors.Credentials) (map[string]any, error) {
	archiveParams, err := parseArchiveParams(params)
	if err != nil {
		return nil, nil
	}
	if err := archiveParams.validate(); err != nil {
		return nil, nil
	}

	metaByUID, err := resolveMessageEnvelopes(ctx, c, creds, archiveParams.Folder, archiveParams.MessageIDs, connectors.MailboxUIDValidityFromContext(ctx))
	if err != nil || len(metaByUID) == 0 {
		return nil, nil
	}

	if len(archiveParams.MessageIDs) == 1 {
		if meta, ok := metaByUID[archiveParams.MessageIDs[0]]; ok {
			return meta.asMap(), nil
		}
		return nil, nil
	}

	messages := make(map[string]any, len(metaByUID))
	for _, id := range archiveParams.MessageIDs {
		if meta, ok := metaByUID[id]; ok {
			messages[fmt.Sprintf("%d", id)] = meta.asMap()
		}
	}
	if len(messages) == 0 {
		return nil, nil
	}
	return map[string]any{"messages": messages}, nil
}

func resolveMessageEnvelopesIMAP(ctx context.Context, conn *ProtonMailConnector, creds connectors.Credentials, folder string, uids []uint32, store connectors.MailboxUIDValidityStore) (map[uint32]emailEnvelopeMetadata, error) {
	if len(uids) == 0 {
		return nil, nil
	}
	if username, ok := creds.Get(credKeyUsername); !ok || username == "" {
		return nil, nil
	}
	if password, ok := creds.Get(credKeyPassword); !ok || password == "" {
		return nil, nil
	}

	timeout := conn.timeout
	if deadline, ok := ctx.Deadline(); ok {
		if remaining := time.Until(deadline); remaining > 0 && remaining < timeout {
			timeout = remaining
		}
	}

	session, err := connectIMAP(creds, timeout)
	if err != nil {
		return nil, err
	}
	defer session.close()

	mboxData, err := session.selectMailbox(folder)
	if err != nil {
		return nil, err
	}
	if err := syncUIDValidity(folder, mboxData, store, uidValidityVerify); err != nil {
		return nil, err
	}

	var uidSet imap.UIDSet
	for _, id := range uids {
		uidSet.AddNum(imap.UID(id))
	}

	return fetchEnvelopeMetadataByUID(session, uidSet)
}

var _ connectors.ResourceDetailResolver = (*ProtonMailConnector)(nil)
