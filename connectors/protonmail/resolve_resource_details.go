package protonmail

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/supersuit-tech/permission-slip/connectors"
)

// resolveMessageEnvelopes is the IMAP-backed implementation used by
// ResolveResourceDetails. Tests may replace it to avoid a live proxy.
var resolveMessageEnvelopes = resolveMessageEnvelopesWithRetry

const envelopeFetchMaxAttempts = 3

// ResolveResourceDetails fetches human-readable email metadata for approval
// prompts. Only subject, from, to, and date are returned. Errors are non-fatal
// for the approval flow — callers store the approval without details on failure.
func (c *ProtonMailConnector) ResolveResourceDetails(ctx context.Context, actionType string, params json.RawMessage, creds connectors.Credentials) (map[string]any, error) {
	switch actionType {
	case "protonmail.read_email":
		return c.resolveReadEmailDetails(ctx, params, creds)
	case "protonmail.archive_email":
		return c.resolveArchiveEmailDetails(ctx, params, creds)
	case "protonmail.apply_label", "protonmail.remove_label":
		return c.resolveLabelMessageDetails(ctx, params, creds)
	case "protonmail.reply_email":
		return c.resolveReplyEmailDetails(ctx, params, creds)
	case "protonmail.mark_read",
		"protonmail.mark_unread",
		"protonmail.flag",
		"protonmail.unflag",
		"protonmail.move_to_folder",
		"protonmail.delete":
		return c.resolveUIDMessageDetails(ctx, params, creds)
	default:
		return nil, nil
	}
}

func (c *ProtonMailConnector) resolveReadEmailDetails(ctx context.Context, params json.RawMessage, creds connectors.Credentials) (map[string]any, error) {
	var p readEmailParams
	if err := json.Unmarshal(params, &p); err != nil {
		log.Printf("protonmail: resolve read_email details: parse params: %v", err)
		return nil, nil
	}
	if err := p.validate(); err != nil {
		log.Printf("protonmail: resolve read_email details: invalid params: %v", err)
		return nil, nil
	}

	metaByUID, err := resolveMessageEnvelopes(ctx, c, creds, p.Folder, []uint32{p.MessageID}, connectors.MailboxUIDValidityFromContext(ctx))
	if err != nil {
		log.Printf("protonmail: resolve read_email details: envelope fetch (folder %q, uid %d): %v", p.Folder, p.MessageID, err)
		return nil, nil
	}
	meta, ok := metaByUID[p.MessageID]
	if !ok {
		log.Printf("protonmail: resolve read_email details: uid %d not found in folder %q", p.MessageID, p.Folder)
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
		log.Printf("protonmail: resolve reply_email details: parse params: %v", err)
		return nil, nil
	}
	if p.InReplyToMessageID == 0 {
		log.Printf("protonmail: resolve reply_email details: missing in_reply_to_message_id")
		return nil, nil
	}
	if p.Folder == "" {
		p.Folder = "INBOX"
	}

	metaByUID, err := resolveMessageEnvelopes(ctx, c, creds, p.Folder, []uint32{p.InReplyToMessageID}, connectors.MailboxUIDValidityFromContext(ctx))
	if err != nil {
		log.Printf("protonmail: resolve reply_email details: envelope fetch (folder %q, uid %d): %v", p.Folder, p.InReplyToMessageID, err)
		return nil, nil
	}
	meta, ok := metaByUID[p.InReplyToMessageID]
	if !ok {
		log.Printf("protonmail: resolve reply_email details: uid %d not found in folder %q", p.InReplyToMessageID, p.Folder)
		return nil, nil
	}
	return map[string]any{"in_reply_to": meta.asMap()}, nil
}

func (c *ProtonMailConnector) resolveUIDMessageDetails(ctx context.Context, params json.RawMessage, creds connectors.Credentials) (map[string]any, error) {
	uidParams, err := parseUIDMessageParams(params)
	if err != nil {
		log.Printf("protonmail: resolve message details: parse params: %v", err)
		return nil, nil
	}
	if err := uidParams.validate(); err != nil {
		log.Printf("protonmail: resolve message details: invalid params: %v", err)
		return nil, nil
	}
	return c.resolveMessageDetailsForUIDs(ctx, creds, uidParams.Folder, uidParams.MessageIDs)
}

func (c *ProtonMailConnector) resolveArchiveEmailDetails(ctx context.Context, params json.RawMessage, creds connectors.Credentials) (map[string]any, error) {
	archiveParams, err := parseArchiveParams(params)
	if err != nil {
		log.Printf("protonmail: resolve archive_email details: parse params: %v", err)
		return nil, nil
	}
	if err := validateArchiveParams(archiveParams); err != nil {
		log.Printf("protonmail: resolve archive_email details: invalid params: %v", err)
		return nil, nil
	}

	uids := archiveParams.MessageIDs
	if includeThreadEnabled(archiveParams.IncludeThread) {
		expanded, err := expandArchiveUIDsForApproval(ctx, c, creds, archiveParams.Folder, archiveParams.MessageIDs)
		if err != nil {
			log.Printf("protonmail: resolve archive_email details: thread expansion (folder %q, %d uids): %v", archiveParams.Folder, len(archiveParams.MessageIDs), err)
			return c.resolveMessageDetailsForUIDs(ctx, creds, archiveParams.Folder, archiveParams.MessageIDs)
		}
		uids = expanded
	}

	return c.resolveMessageDetailsForUIDs(ctx, creds, archiveParams.Folder, uids)
}

func (c *ProtonMailConnector) resolveLabelMessageDetails(ctx context.Context, params json.RawMessage, creds connectors.Credentials) (map[string]any, error) {
	labelParams, err := parseLabelMessageParams(params)
	if err != nil {
		log.Printf("protonmail: resolve label message details: parse params: %v", err)
		return nil, nil
	}
	if err := labelParams.validate(); err != nil {
		log.Printf("protonmail: resolve label message details: invalid params: %v", err)
		return nil, nil
	}

	uids := labelParams.MessageIDs
	if includeThreadEnabled(labelParams.IncludeThread) {
		expanded, err := expandArchiveUIDsForApproval(ctx, c, creds, labelParams.Folder, labelParams.MessageIDs)
		if err != nil {
			log.Printf("protonmail: resolve label message details: thread expansion (folder %q, %d uids): %v", labelParams.Folder, len(labelParams.MessageIDs), err)
			return c.resolveMessageDetailsForUIDs(ctx, creds, labelParams.Folder, labelParams.MessageIDs)
		}
		uids = expanded
	}

	return c.resolveMessageDetailsForUIDs(ctx, creds, labelParams.Folder, uids)
}

// expandArchiveUIDsForApproval is the IMAP-backed thread expansion used at
// approval time. Tests may replace it to avoid a live proxy.
var expandArchiveUIDsForApproval = expandArchiveUIDsForApprovalIMAP

func expandArchiveUIDsForApprovalIMAP(ctx context.Context, conn *ProtonMailConnector, creds connectors.Credentials, folder string, targetUIDs []uint32) ([]uint32, error) {
	if len(targetUIDs) == 0 {
		return nil, nil
	}
	if username, ok := creds.Get(credKeyUsername); !ok || username == "" {
		return nil, fmt.Errorf("missing IMAP username credential")
	}
	if password, ok := creds.Get(credKeyPassword); !ok || password == "" {
		return nil, fmt.Errorf("missing IMAP password credential")
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
	if err := syncUIDValidity(folder, mboxData, connectors.MailboxUIDValidityFromContext(ctx), uidValidityVerify); err != nil {
		return nil, err
	}

	return expandArchiveUIDs(session, targetUIDs)
}

func (c *ProtonMailConnector) resolveMessageDetailsForUIDs(ctx context.Context, creds connectors.Credentials, folder string, messageIDs []uint32) (map[string]any, error) {
	metaByUID, err := resolveMessageEnvelopes(ctx, c, creds, folder, messageIDs, connectors.MailboxUIDValidityFromContext(ctx))
	if err != nil {
		log.Printf("protonmail: resolve message details: envelope fetch (folder %q, %d uids): %v", folder, len(messageIDs), err)
		return nil, nil
	}
	if len(metaByUID) == 0 {
		log.Printf("protonmail: resolve message details: no envelopes returned (folder %q, %d uids)", folder, len(messageIDs))
		return nil, nil
	}

	if len(messageIDs) == 1 {
		if meta, ok := metaByUID[messageIDs[0]]; ok {
			return meta.asMap(), nil
		}
		log.Printf("protonmail: resolve message details: uid %d not found in folder %q", messageIDs[0], folder)
		return nil, nil
	}

	messages := make(map[string]any, len(metaByUID))
	for _, id := range messageIDs {
		if meta, ok := metaByUID[id]; ok {
			messages[fmt.Sprintf("%d", id)] = meta.asMap()
		}
	}
	if len(messages) == 0 {
		log.Printf("protonmail: resolve message details: none of %d uids found in folder %q", len(messageIDs), folder)
		return nil, nil
	}
	return map[string]any{"messages": messages}, nil
}

func resolveMessageEnvelopesIMAPImpl(ctx context.Context, conn *ProtonMailConnector, creds connectors.Credentials, folder string, uids []uint32, store connectors.MailboxUIDValidityStore) (map[uint32]emailEnvelopeMetadata, error) {
	if len(uids) == 0 {
		return nil, nil
	}
	if username, ok := creds.Get(credKeyUsername); !ok || username == "" {
		log.Printf("protonmail: resolve message details: missing IMAP username credential, skipping enrichment")
		return nil, nil
	}
	if password, ok := creds.Get(credKeyPassword); !ok || password == "" {
		log.Printf("protonmail: resolve message details: missing IMAP password credential, skipping enrichment")
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

// resolveMessageEnvelopesIMAP is the single-attempt IMAP fetch. Tests may replace it.
var resolveMessageEnvelopesIMAP = resolveMessageEnvelopesIMAPImpl

func resolveMessageEnvelopesWithRetry(ctx context.Context, conn *ProtonMailConnector, creds connectors.Credentials, folder string, uids []uint32, store connectors.MailboxUIDValidityStore) (map[uint32]emailEnvelopeMetadata, error) {
	var lastErr error
	for attempt := 1; attempt <= envelopeFetchMaxAttempts; attempt++ {
		meta, err := resolveMessageEnvelopesIMAP(ctx, conn, creds, folder, uids, store)
		if err == nil {
			if len(meta) > 0 || len(uids) == 0 {
				return meta, nil
			}
			lastErr = fmt.Errorf("envelope fetch returned no metadata for %d uid(s) in folder %q", len(uids), folder)
		} else {
			lastErr = err
		}
		if attempt < envelopeFetchMaxAttempts {
			log.Printf("protonmail: envelope fetch attempt %d/%d failed (folder %q, %d uid(s)): %v",
				attempt, envelopeFetchMaxAttempts, folder, len(uids), lastErr)
			backoff := time.Duration(attempt) * 200 * time.Millisecond
			timer := time.NewTimer(backoff)
			select {
			case <-ctx.Done():
				timer.Stop()
				return nil, ctx.Err()
			case <-timer.C:
			}
		}
	}
	return nil, lastErr
}

var _ connectors.ResourceDetailResolver = (*ProtonMailConnector)(nil)
