package zoom

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// addRegistrantAction implements connectors.Action for zoom.add_registrant.
// It registers an attendee for a meeting or webinar via POST /meetings/{meeting_id}/registrants.
type addRegistrantAction struct {
	conn *ZoomConnector
}

type addRegistrantParams struct {
	MeetingID string `json:"meeting_id"`
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name,omitempty"`
}

func (p *addRegistrantParams) validate() error {
	p.MeetingID = strings.TrimSpace(p.MeetingID)
	if p.MeetingID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: meeting_id"}
	}
	p.Email = strings.TrimSpace(p.Email)
	if p.Email == "" {
		return &connectors.ValidationError{Message: "missing required parameter: email"}
	}
	// Basic email format check.
	if !strings.Contains(p.Email, "@") || !strings.Contains(p.Email, ".") {
		return &connectors.ValidationError{Message: "email does not appear to be a valid email address"}
	}
	p.FirstName = strings.TrimSpace(p.FirstName)
	if p.FirstName == "" {
		return &connectors.ValidationError{Message: "missing required parameter: first_name"}
	}
	return nil
}

type addRegistrantRequest struct {
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name,omitempty"`
}

type addRegistrantResponse struct {
	ID             string `json:"id"`
	JoinURL        string `json:"join_url"`
	RegistrantID   string `json:"registrant_id"`
	StartTime      string `json:"start_time"`
	Topic          string `json:"topic"`
}

func (a *addRegistrantAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params addRegistrantParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	body := addRegistrantRequest{
		Email:     params.Email,
		FirstName: params.FirstName,
		LastName:  params.LastName,
	}

	var resp addRegistrantResponse
	reqURL := fmt.Sprintf("%s/meetings/%s/registrants", a.conn.baseURL, url.PathEscape(params.MeetingID))
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodPost, reqURL, body, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]string{
		"registrant_id": resp.RegistrantID,
		"join_url":      resp.JoinURL,
		"topic":         resp.Topic,
	})
}
