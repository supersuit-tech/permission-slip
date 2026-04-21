package microsoft

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// graphFullMessage is a subset of Graph message fields for thread building and reply context.
type graphFullMessage struct {
	ID               string              `json:"id"`
	Subject          string              `json:"subject"`
	ConversationID   string              `json:"conversationId"`
	ReceivedDateTime string              `json:"receivedDateTime"`
	BodyPreview      string              `json:"bodyPreview"`
	From             *graphMailAddress   `json:"from,omitempty"`
	ToRecipients     []*graphMailAddress `json:"toRecipients,omitempty"`
	CCRecipients     []*graphMailAddress `json:"ccRecipients,omitempty"`
	Body             graphEmailBody      `json:"body"`
	Attachments      []graphAttachment   `json:"attachments,omitempty"`
}

type graphAttachment struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
}

type graphMessagesPage struct {
	Value []graphFullMessage `json:"value"`
}

func addressListFromRecipients(rs []*graphMailAddress) []string {
	if len(rs) == 0 {
		return nil
	}
	out := make([]string, 0, len(rs))
	for _, r := range rs {
		if r == nil {
			continue
		}
		addr := strings.TrimSpace(r.EmailAddress.Address)
		if addr != "" {
			out = append(out, addr)
		}
	}
	return out
}

func stripGraphHeaderNewlines(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "\r", ""), "\n", "")
}

// fetchGraphMessage loads a single message with body and expanded attachments metadata.
func (c *MicrosoftConnector) fetchGraphMessage(ctx context.Context, creds connectors.Credentials, messageID string) (*graphFullMessage, error) {
	if err := validateGraphID("message_id", messageID); err != nil {
		return nil, err
	}
	path := "/me/messages/" + url.PathEscape(messageID) +
		"?$select=" + url.QueryEscape("id,subject,conversationId,receivedDateTime,bodyPreview,from,toRecipients,ccRecipients,body") +
		"&$expand=" + url.QueryEscape("attachments($select=name,size)")
	var msg graphFullMessage
	if err := c.doRequest(ctx, http.MethodGet, path, creds, nil, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

// buildMicrosoftEmailThread fetches the anchor message and full conversation ordered oldest→newest.
func (c *MicrosoftConnector) buildMicrosoftEmailThread(ctx context.Context, creds connectors.Credentials, messageID string) (*connectors.EmailThreadPayload, error) {
	src, err := c.fetchGraphMessage(ctx, creds, messageID)
	if err != nil {
		return nil, err
	}
	convID := strings.TrimSpace(src.ConversationID)
	if convID == "" {
		return singleMessageThread(src), nil
	}

	filter := fmt.Sprintf("conversationId eq '%s'", escapeODataString(convID))
	order := "receivedDateTime asc"
	selectFields := "id,subject,conversationId,receivedDateTime,bodyPreview,from,toRecipients,ccRecipients,body"
	path := "/me/messages?$filter=" + url.QueryEscape(filter) +
		"&$orderby=" + url.QueryEscape(order) +
		"&$select=" + url.QueryEscape(selectFields) +
		"&$expand=" + url.QueryEscape("attachments($select=name,size)") +
		"&$top=100"

	var page graphMessagesPage
	if err := c.doRequest(ctx, http.MethodGet, path, creds, nil, &page); err != nil {
		return singleMessageThread(src), nil
	}
	if len(page.Value) == 0 {
		return singleMessageThread(src), nil
	}

	subject := stripGraphHeaderNewlines(page.Value[0].Subject)
	if subject == "" {
		subject = stripGraphHeaderNewlines(src.Subject)
	}

	msgs := make([]connectors.EmailThreadMessage, 0, len(page.Value))
	for _, m := range page.Value {
		em := graphMessageToThreadMessage(&m)
		msgs = append(msgs, em)
	}
	return &connectors.EmailThreadPayload{Subject: subject, Messages: msgs}, nil
}

func escapeODataString(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

func singleMessageThread(m *graphFullMessage) *connectors.EmailThreadPayload {
	if m == nil {
		return &connectors.EmailThreadPayload{}
	}
	em := graphMessageToThreadMessage(m)
	return &connectors.EmailThreadPayload{
		Subject:  stripGraphHeaderNewlines(m.Subject),
		Messages: []connectors.EmailThreadMessage{em},
	}
}

func graphMessageToThreadMessage(m *graphFullMessage) connectors.EmailThreadMessage {
	var from string
	if m.From != nil {
		from = stripGraphHeaderNewlines(m.From.EmailAddress.Address)
	}
	em := connectors.EmailThreadMessage{
		From:      from,
		To:        addressListFromRecipients(m.ToRecipients),
		Cc:        addressListFromRecipients(m.CCRecipients),
		Date:      stripGraphHeaderNewlines(m.ReceivedDateTime),
		Snippet:   stripGraphHeaderNewlines(m.BodyPreview),
		MessageID: m.ID,
	}
	ct := strings.ToLower(strings.TrimSpace(m.Body.ContentType))
	content := m.Body.Content
	switch ct {
	case "html":
		em.BodyHTML = content
		em.BodyText = connectors.StripHTMLToPlain(content)
	default:
		em.BodyText = content
	}
	for _, a := range m.Attachments {
		fn := strings.TrimSpace(a.Name)
		if fn == "" {
			fn = "(no filename)"
		}
		em.Attachments = append(em.Attachments, connectors.EmailThreadAttachment{
			Filename:  fn,
			SizeBytes: a.Size,
		})
	}
	connectors.TruncateEmailThreadBodies(&em)
	return em
}
