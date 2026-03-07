package google

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// getPresentationAction implements connectors.Action for google.get_presentation.
// It retrieves metadata about a presentation via GET /v1/presentations/{presentationId}.
type getPresentationAction struct {
	conn *GoogleConnector
}

// getPresentationParams is the user-facing parameter schema.
type getPresentationParams struct {
	PresentationID string `json:"presentation_id"`
}

func (p *getPresentationParams) validate() error {
	if p.PresentationID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: presentation_id"}
	}
	return nil
}

// slidesGetResponse is the Google Slides API response from presentations.get.
type slidesGetResponse struct {
	PresentationID string       `json:"presentationId"`
	Title          string       `json:"title"`
	Slides         []slidePage  `json:"slides"`
}

type slidePage struct {
	ObjectID string `json:"objectId"`
}

// Execute retrieves metadata about a Google Slides presentation.
func (a *getPresentationAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params getPresentationParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	var resp slidesGetResponse
	getURL := a.conn.slidesBaseURL + "/v1/presentations/" + url.PathEscape(params.PresentationID)
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodGet, getURL, nil, &resp); err != nil {
		return nil, err
	}

	slideIDs := make([]string, 0, len(resp.Slides))
	for _, s := range resp.Slides {
		slideIDs = append(slideIDs, s.ObjectID)
	}

	return connectors.JSONResult(map[string]any{
		"presentation_id": resp.PresentationID,
		"title":           resp.Title,
		"url":             presentationURL(resp.PresentationID),
		"slide_count":     len(slideIDs),
		"slides":          slideIDs,
	})
}
