package protonmail

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/emersion/go-imap/v2"
	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// archiveMailbox is the IMAP folder that Proton Mail Bridge exposes for
// archived messages. Proton Bridge maps the "Archive" label to this folder.
const archiveMailbox = "Archive"

// archiveEmailAction moves one or more emails to the Archive folder via IMAP
// MOVE. If the server doesn't support MOVE (RFC 6851), the go-imap client
// falls back to COPY + STORE \Deleted + EXPUNGE automatically.
type archiveEmailAction struct {
	conn *ProtonMailConnector
}

type archiveEmailParams struct {
	MessageIDs []uint32 `json:"message_ids"`
	Folder     string   `json:"folder"`
}

func (p *archiveEmailParams) validate() error {
	if len(p.MessageIDs) == 0 {
		return &connectors.ValidationError{Message: "missing required parameter: message_ids (must contain at least one sequence number)"}
	}
	if len(p.MessageIDs) > maxLimit {
		return &connectors.ValidationError{Message: fmt.Sprintf("too many message_ids: maximum is %d", maxLimit)}
	}
	for _, id := range p.MessageIDs {
		if id == 0 {
			return &connectors.ValidationError{Message: "message_ids must not contain zero values"}
		}
	}
	if p.Folder == "" {
		p.Folder = "INBOX"
	}
	if p.Folder == archiveMailbox {
		return &connectors.ValidationError{Message: "cannot archive emails that are already in the Archive folder"}
	}
	return nil
}

func (a *archiveEmailAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params archiveEmailParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	session, err := connectIMAP(req.Credentials, a.conn.timeout)
	if err != nil {
		return nil, err
	}
	defer session.close()

	// Open the source folder in read-write mode so we can move messages.
	mboxData, err := session.selectMailboxReadWrite(params.Folder)
	if err != nil {
		return nil, err
	}

	// Validate all sequence numbers are within range.
	for _, id := range params.MessageIDs {
		if id > mboxData.NumMessages {
			return nil, &connectors.ValidationError{
				Message: fmt.Sprintf("message_id %d not found (mailbox has %d messages)", id, mboxData.NumMessages),
			}
		}
	}

	var seqSet imap.SeqSet
	for _, id := range params.MessageIDs {
		seqSet.AddNum(id)
	}

	// MOVE messages to the Archive folder. The go-imap client handles fallback
	// to COPY + STORE + EXPUNGE if the server lacks MOVE support.
	moveCmd := session.client.Move(seqSet, archiveMailbox)
	if _, err := moveCmd.Wait(); err != nil {
		return nil, mapIMAPError(err)
	}

	return connectors.JSONResult(map[string]any{
		"status":   "archived",
		"folder":   params.Folder,
		"archived": len(params.MessageIDs),
	})
}
