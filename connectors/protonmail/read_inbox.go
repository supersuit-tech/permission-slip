package protonmail

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/emersion/go-imap/v2"
	"github.com/supersuit-tech/permission-slip-web/connectors"
)

type readInboxAction struct {
	conn *ProtonMailConnector
}

type readInboxParams struct {
	Folder     string `json:"folder"`
	Limit      int    `json:"limit"`
	UnreadOnly bool   `json:"unread_only"`
}

func (p *readInboxParams) validate() error {
	if p.Folder == "" {
		p.Folder = "INBOX"
	}
	if p.Limit <= 0 {
		p.Limit = 10
	}
	if p.Limit > 50 {
		return &connectors.ValidationError{Message: "limit must be at most 50"}
	}
	return nil
}

func (a *readInboxAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params readInboxParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	session, err := connectIMAP(ctx, req.Credentials, a.conn.timeout)
	if err != nil {
		return nil, err
	}
	defer session.close()

	mboxData, err := session.selectMailbox(params.Folder)
	if err != nil {
		return nil, err
	}

	if mboxData.NumMessages == 0 {
		return connectors.JSONResult(map[string]any{
			"emails": []emailSummary{},
			"total":  0,
		})
	}

	// If unread_only, search for unseen messages first.
	var seqNums []uint32
	if params.UnreadOnly {
		criteria := &imap.SearchCriteria{
			NotFlag: []imap.Flag{imap.FlagSeen},
		}
		searchData, err := session.client.Search(criteria, nil).Wait()
		if err != nil {
			return nil, mapIMAPError(err)
		}
		seqNums = searchData.AllSeqNums()
	}

	// Determine which messages to fetch.
	var seqSet imap.SeqSet
	if params.UnreadOnly {
		if len(seqNums) == 0 {
			return connectors.JSONResult(map[string]any{
				"emails": []emailSummary{},
				"total":  0,
			})
		}
		// Take only the last `limit` unread messages (most recent).
		start := 0
		if len(seqNums) > params.Limit {
			start = len(seqNums) - params.Limit
		}
		for _, n := range seqNums[start:] {
			seqSet.AddNum(n)
		}
	} else {
		// Fetch the last `limit` messages by sequence number.
		from := uint32(1)
		if mboxData.NumMessages > uint32(params.Limit) {
			from = mboxData.NumMessages - uint32(params.Limit) + 1
		}
		seqSet.AddRange(from, mboxData.NumMessages)
	}

	fetchCmd := session.client.Fetch(seqSet, &imap.FetchOptions{
		Envelope: true,
		Flags:    true,
	})
	defer fetchCmd.Close()

	var emails []emailSummary
	for {
		msg := fetchCmd.Next()
		if msg == nil {
			break
		}
		buf, err := msg.Collect()
		if err != nil {
			return nil, mapIMAPError(err)
		}
		if buf.Envelope != nil {
			emails = append(emails, envelopeToSummary(buf.SeqNum, buf.Envelope, buf.Flags))
		}
	}

	return connectors.JSONResult(map[string]any{
		"emails": emails,
		"total":  len(emails),
	})
}
