package protonmail

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/emersion/go-imap/v2"
	"github.com/supersuit-tech/permission-slip/connectors"
)

// fetchAttachmentContent fetches and decodes a single attachment from IMAP.
// Tests may replace this variable to avoid a live IMAP server.
var fetchAttachmentContent = fetchAttachmentContentFromIMAP

type downloadAttachmentAction struct {
	conn *ProtonMailConnector
}

type downloadAttachmentParams struct {
	MessageID    uint32 `json:"message_id"`
	Folder       string `json:"folder"`
	AttachmentID string `json:"attachment_id"`
}

func (p *downloadAttachmentParams) validate() error {
	if p.MessageID == 0 {
		return &connectors.ValidationError{Message: "missing required parameter: message_id"}
	}
	p.AttachmentID = strings.TrimSpace(p.AttachmentID)
	if p.AttachmentID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: attachment_id"}
	}
	if _, err := parsePartPath(p.AttachmentID); err != nil {
		return &connectors.ValidationError{Message: fmt.Sprintf("invalid attachment_id: %v", err)}
	}
	if p.Folder == "" {
		p.Folder = "INBOX"
	}
	return nil
}

type downloadAttachmentResult struct {
	UID           uint32 `json:"uid"`
	Folder        string `json:"folder"`
	AttachmentID  string `json:"attachment_id"`
	Filename      string `json:"filename"`
	ContentType   string `json:"content_type"`
	Size          int    `json:"size"`
	ContentBase64 string `json:"content_base64"`
}

func (a *downloadAttachmentAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params downloadAttachmentParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	partPath, err := parsePartPath(params.AttachmentID)
	if err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid attachment_id: %v", err)}
	}

	result, err := fetchAttachmentContent(ctx, a.conn, req.Credentials, req.MailboxUIDValidity, params.Folder, params.MessageID, partPath)
	if err != nil {
		return nil, err
	}

	return connectors.JSONResult(downloadAttachmentResult{
		UID:           params.MessageID,
		Folder:        params.Folder,
		AttachmentID:  params.AttachmentID,
		Filename:      result.Filename,
		ContentType:   result.ContentType,
		Size:          len(result.Content),
		ContentBase64: base64.StdEncoding.EncodeToString(result.Content),
	})
}

type fetchedAttachment struct {
	Filename    string
	ContentType string
	Content     []byte
}

func fetchAttachmentContentFromIMAP(
	ctx context.Context,
	conn *ProtonMailConnector,
	creds connectors.Credentials,
	mailboxUIDValidity map[string]uint32,
	folder string,
	messageID uint32,
	partPath []int,
) (*fetchedAttachment, error) {
	_ = ctx

	session, err := connectIMAP(creds, conn.timeout)
	if err != nil {
		return nil, err
	}
	defer session.close()

	mboxData, err := session.selectMailbox(folder)
	if err != nil {
		return nil, err
	}
	if err := syncUIDValidity(folder, mboxData, mailboxUIDValidity, uidValidityVerify); err != nil {
		return nil, err
	}

	uidSet := imap.UIDSetNum(imap.UID(messageID))
	bodySection := &imap.FetchItemBodySection{
		Part: partPath,
		Peek: true,
		Partial: &imap.SectionPartial{
			Offset: 0,
			Size:   int64(maxBodySize) + 1,
		},
	}

	fetchCmd := session.client.Fetch(uidSet, &imap.FetchOptions{
		UID: true,
		BodyStructure: &imap.FetchItemBodyStructure{
			Extended: true,
		},
		BodySection: []*imap.FetchItemBodySection{bodySection},
	})
	defer fetchCmd.Close()

	msg := fetchCmd.Next()
	if msg == nil {
		return nil, &connectors.ValidationError{
			Message: fmt.Sprintf("message uid %d not found in folder %q", messageID, folder),
		}
	}

	buf, err := msg.Collect()
	if err != nil {
		return nil, mapIMAPError(err)
	}
	if buf.UID == 0 {
		return nil, &connectors.ValidationError{
			Message: fmt.Sprintf("message uid %d not found in folder %q", messageID, folder),
		}
	}

	part, ok := findAttachmentPart(buf.BodyStructure, partPath)
	if !ok {
		return nil, &connectors.ValidationError{
			Message: fmt.Sprintf("attachment %q not found on message uid %d in folder %q", formatPartPath(partPath), messageID, folder),
		}
	}

	raw := buf.FindBodySection(bodySection)
	if raw == nil {
		return nil, &connectors.ExternalError{
			Message: fmt.Sprintf("failed to fetch attachment %q from message uid %d", formatPartPath(partPath), messageID),
		}
	}
	if len(raw) > maxBodySize {
		return nil, &connectors.ValidationError{
			Message: fmt.Sprintf("attachment %q exceeds maximum size of %d bytes", formatPartPath(partPath), maxBodySize),
		}
	}

	decoded, err := decodeTransferEncoding(raw, part.Encoding)
	if err != nil {
		return nil, err
	}
	if len(decoded) > maxBodySize {
		return nil, &connectors.ValidationError{
			Message: fmt.Sprintf("attachment %q exceeds maximum size of %d bytes after decoding", formatPartPath(partPath), maxBodySize),
		}
	}

	return &fetchedAttachment{
		Filename:    part.Filename(),
		ContentType: part.MediaType(),
		Content:     decoded,
	}, nil
}
