package google

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// createPresentationAction implements connectors.Action for google.create_presentation.
// It creates a new Google Slides presentation via POST /v1/presentations.
type createPresentationAction struct {
	conn *GoogleConnector
}

// createPresentationParams is the user-facing parameter schema.
type createPresentationParams struct {
	Title string `json:"title"`
}

func (p *createPresentationParams) validate() error {
	if p.Title == "" {
		return &connectors.ValidationError{Message: "missing required parameter: title"}
	}
	return nil
}

// slidesCreateRequest is the Google Slides API request body for presentations.create.
type slidesCreateRequest struct {
	Title string `json:"title"`
}

// slidesCreateResponse is the Google Slides API response from presentations.create.
type slidesCreateResponse struct {
	PresentationID string `json:"presentationId"`
	Title          string `json:"title"`
}

// Execute creates a new Google Slides presentation and returns its metadata.
func (a *createPresentationAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createPresentationParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	body := slidesCreateRequest{Title: params.Title}
	var resp slidesCreateResponse
	url := a.conn.slidesBaseURL + "/v1/presentations"
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodPost, url, body, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]string{
		"presentation_id": resp.PresentationID,
		"title":           resp.Title,
	})
}
