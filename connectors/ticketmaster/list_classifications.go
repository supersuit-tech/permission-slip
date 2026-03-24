package ticketmaster

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// listClassificationsAction implements connectors.Action for ticketmaster.list_classifications.
type listClassificationsAction struct {
	conn *TicketmasterConnector
}

type listClassificationsParams struct {
	ClassificationID string `json:"classification_id"`
	Locale           string `json:"locale"`
	Source           string `json:"source"`
}

func (a *listClassificationsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params listClassificationsParams
	if err := parseParams(req.Parameters, &params); err != nil {
		return nil, err
	}

	q := url.Values{}
	appendNonEmpty(q, "locale", trimString(params.Locale))
	appendNonEmpty(q, "source", trimString(params.Source))

	path := "classifications.json"
	if id := trimString(params.ClassificationID); id != "" {
		if err := validateTicketmasterID(id, "classification_id"); err != nil {
			return nil, err
		}
		path = fmt.Sprintf("classifications/%s.json", url.PathEscape(id))
	}

	var out json.RawMessage
	if err := a.conn.doGET(ctx, req.Credentials, path, q, &out); err != nil {
		return nil, err
	}
	return &connectors.ActionResult{Data: out}, nil
}
