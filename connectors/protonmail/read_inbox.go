package protonmail

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/emersion/go-imap/v2"
	"github.com/supersuit-tech/permission-slip/connectors"
)

type readInboxAction struct {
	conn *ProtonMailConnector
}

type readInboxParams struct {
	Folder     string `json:"folder"`
	Limit      int    `json:"limit"`
	UnreadOnly bool   `json:"unread_only"`
	// GroupByThread collapses results to one entry per conversation (the
	// latest message). Defaults to true when omitted; nil distinguishes
	// "omitted" from an explicit false.
	GroupByThread *bool `json:"group_by_thread"`
}

func (p *readInboxParams) validate() error {
	if p.Folder == "" {
		p.Folder = "INBOX"
	}
	return validateLimit(&p.Limit)
}

func (a *readInboxAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params readInboxParams
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

	mboxData, err := session.selectMailbox(params.Folder)
	if err != nil {
		return nil, err
	}
	if err := syncUIDValidity(params.Folder, mboxData, req.MailboxUIDValidity, uidValidityRecord); err != nil {
		return nil, err
	}

	if mboxData.NumMessages == 0 {
		return emailListResultWithFolder(params.Folder, nil)
	}

	var emails []emailSummary
	if params.UnreadOnly {
		criteria := &imap.SearchCriteria{
			NotFlag: []imap.Flag{imap.FlagSeen},
		}
		searchData, err := session.client.UIDSearch(criteria, nil).Wait()
		if err != nil {
			return nil, mapIMAPError(err)
		}
		uids := searchData.AllUIDs()
		if len(uids) == 0 {
			return emailListResultWithFolder(params.Folder, nil)
		}

		start := 0
		if len(uids) > params.Limit {
			start = len(uids) - params.Limit
		}
		limited := uids[start:]

		var uidSet imap.UIDSet
		for _, uid := range limited {
			uidSet.AddNum(uid)
		}
		emails, err = fetchEnvelopesByUID(session, uidSet)
		if err != nil {
			return nil, err
		}
	} else {
		from := uint32(1)
		if mboxData.NumMessages > uint32(params.Limit) {
			from = mboxData.NumMessages - uint32(params.Limit) + 1
		}
		var seqSet imap.SeqSet
		seqSet.AddRange(from, mboxData.NumMessages)

		emails, err = fetchRecentEnvelopesBySeq(session, seqSet)
		if err != nil {
			return nil, err
		}
	}

	if groupByThreadEnabled(params.GroupByThread) {
		emails = groupIntoThreads(emails)
	}
	return emailListResultWithFolder(params.Folder, emails)
}
