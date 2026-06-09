package protonmail

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/supersuit-tech/permission-slip/connectors"
)

type searchEmailsAction struct {
	conn *ProtonMailConnector
}

type searchEmailsParams struct {
	Folder  string `json:"folder"`
	Subject string `json:"subject"`
	From    string `json:"from"`
	Since   string `json:"since"`
	Before  string `json:"before"`
	Limit   int    `json:"limit"`
}

func (p *searchEmailsParams) validate() error {
	if p.Folder == "" {
		p.Folder = "INBOX"
	}
	if err := validateLimit(&p.Limit); err != nil {
		return err
	}
	if p.Subject == "" && p.From == "" && p.Since == "" && p.Before == "" {
		return &connectors.ValidationError{Message: "at least one search criterion (subject, from, since, before) is required"}
	}
	if p.Since != "" {
		if _, err := time.Parse("2006-01-02", p.Since); err != nil {
			return &connectors.ValidationError{Message: fmt.Sprintf("invalid since date format, expected YYYY-MM-DD: %v", err)}
		}
	}
	if p.Before != "" {
		if _, err := time.Parse("2006-01-02", p.Before); err != nil {
			return &connectors.ValidationError{Message: fmt.Sprintf("invalid before date format, expected YYYY-MM-DD: %v", err)}
		}
	}
	return nil
}

func (a *searchEmailsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params searchEmailsParams
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

	criteria := &imap.SearchCriteria{}

	if params.Subject != "" {
		criteria.Header = append(criteria.Header, imap.SearchCriteriaHeaderField{
			Key:   "SUBJECT",
			Value: params.Subject,
		})
	}
	if params.From != "" {
		criteria.Header = append(criteria.Header, imap.SearchCriteriaHeaderField{
			Key:   "FROM",
			Value: params.From,
		})
	}
	if params.Since != "" {
		t, _ := time.Parse("2006-01-02", params.Since)
		criteria.Since = t
	}
	if params.Before != "" {
		t, _ := time.Parse("2006-01-02", params.Before)
		criteria.Before = t
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

	emails, err := fetchEnvelopesByUID(session, uidSet)
	if err != nil {
		return nil, err
	}
	return emailListResultWithFolder(params.Folder, emails)
}
