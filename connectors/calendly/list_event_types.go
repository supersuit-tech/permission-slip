package calendly

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// listEventTypesAction implements connectors.Action for calendly.list_event_types.
// It lists event types via GET /event_types?user={user_uri}.
type listEventTypesAction struct {
	conn *CalendlyConnector
}

type listEventTypesParams struct {
	UserURI string `json:"user_uri,omitempty"`
	Active  *bool  `json:"active,omitempty"`
}

// calendlyEventTypesResponse is the Calendly API response from GET /event_types.
type calendlyEventTypesResponse struct {
	Collection []calendlyEventType `json:"collection"`
}

type calendlyEventType struct {
	URI                string `json:"uri"`
	Name               string `json:"name"`
	Active             bool   `json:"active"`
	Slug               string `json:"slug"`
	SchedulingURL      string `json:"scheduling_url"`
	Duration           int    `json:"duration"`
	Kind               string `json:"kind"`
	Type               string `json:"type"`
	Color              string `json:"color"`
	DescriptionPlain   string `json:"description_plain"`
	InternalNote       string `json:"internal_note"`
	PoolingType        string `json:"pooling_type"`
	SecretEvent        bool   `json:"secret"`
}

type eventTypeItem struct {
	URI           string `json:"uri"`
	Name          string `json:"name"`
	Active        bool   `json:"active"`
	Slug          string `json:"slug"`
	SchedulingURL string `json:"scheduling_url"`
	Duration      int    `json:"duration"`
	Kind          string `json:"kind"`
	Description   string `json:"description,omitempty"`
}

func (a *listEventTypesAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params listEventTypesParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}

	userURI, err := a.conn.resolveUserURI(ctx, req.Credentials, params.UserURI)
	if err != nil {
		return nil, err
	}

	q := url.Values{}
	q.Set("user", userURI)
	if params.Active != nil {
		q.Set("active", strconv.FormatBool(*params.Active))
	}

	var resp calendlyEventTypesResponse
	reqURL := a.conn.baseURL + "/event_types?" + q.Encode()
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodGet, reqURL, nil, &resp); err != nil {
		return nil, err
	}

	items := make([]eventTypeItem, 0, len(resp.Collection))
	for _, et := range resp.Collection {
		items = append(items, eventTypeItem{
			URI:           et.URI,
			Name:          et.Name,
			Active:        et.Active,
			Slug:          et.Slug,
			SchedulingURL: et.SchedulingURL,
			Duration:      et.Duration,
			Kind:          et.Kind,
			Description:   et.DescriptionPlain,
		})
	}

	return connectors.JSONResult(map[string]any{
		"total_event_types": len(items),
		"event_types":       items,
	})
}
