package protonmail

import (
	"context"

	"github.com/supersuit-tech/permission-slip/connectors"
)

type listLabelsAction struct {
	conn *ProtonMailConnector
}

func (a *listLabelsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	session, err := connectIMAP(req.Credentials, a.conn.timeout)
	if err != nil {
		return nil, err
	}
	defer session.close()

	listCmd := session.client.List("", "*", nil)
	mailboxes, err := listCmd.Collect()
	if err != nil {
		return nil, mapIMAPError(err)
	}

	labels := make([]map[string]any, 0)
	for _, mbox := range mailboxes {
		if mbox == nil || !isLabelMailbox(mbox.Mailbox) {
			continue
		}
		entry := map[string]any{
			"name":    labelDisplayName(mbox.Mailbox),
			"mailbox": mbox.Mailbox,
		}
		if len(mbox.Attrs) > 0 {
			attrs := make([]string, 0, len(mbox.Attrs))
			for _, attr := range mbox.Attrs {
				attrs = append(attrs, string(attr))
			}
			entry["attributes"] = attrs
		}
		if mbox.Delim != 0 {
			entry["delimiter"] = string(mbox.Delim)
		}
		labels = append(labels, entry)
	}

	return connectors.JSONResult(map[string]any{
		"labels": labels,
		"total":  len(labels),
	})
}
