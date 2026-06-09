package protonmail

import (
	"context"
	"fmt"

	"github.com/supersuit-tech/permission-slip/connectors"
)

type listFoldersAction struct {
	conn *ProtonMailConnector
}

func (a *listFoldersAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
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

	folders := make([]map[string]any, 0, len(mailboxes))
	for _, mbox := range mailboxes {
		if mbox == nil {
			continue
		}
		entry := map[string]any{
			"name": mbox.Mailbox,
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
		folders = append(folders, entry)
	}

	return connectors.JSONResult(map[string]any{
		"folders": folders,
		"total":   len(folders),
	})
}
