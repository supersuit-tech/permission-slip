package sendgrid

import (
	"fmt"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// campaignFields holds the common fields shared between send_campaign and
// schedule_campaign parameters. Extracted to avoid duplicating validation.
type campaignFields struct {
	Name               string   `json:"name"`
	Subject            string   `json:"subject"`
	HTMLContent        string   `json:"html_content"`
	PlainContent       string   `json:"plain_content"`
	ListIDs            []string `json:"list_ids"`
	SenderID           int      `json:"sender_id"`
	SuppressionGroupID int      `json:"suppression_group_id,omitempty"`
}

// validate checks the shared campaign fields.
func (f *campaignFields) validate() error {
	if f.Name == "" {
		return &connectors.ValidationError{Message: "missing required parameter: name"}
	}
	if len(f.Name) > 100 {
		return &connectors.ValidationError{Message: fmt.Sprintf("name exceeds maximum length of 100 characters (got %d)", len(f.Name))}
	}
	if f.Subject == "" {
		return &connectors.ValidationError{Message: "missing required parameter: subject"}
	}
	if len(f.Subject) > 998 {
		return &connectors.ValidationError{Message: fmt.Sprintf("subject exceeds maximum length of 998 characters (got %d)", len(f.Subject))}
	}
	if len(f.ListIDs) == 0 {
		return &connectors.ValidationError{Message: "missing required parameter: list_ids (must contain at least one list ID)"}
	}
	if f.SenderID == 0 {
		return &connectors.ValidationError{Message: "missing required parameter: sender_id"}
	}
	if f.HTMLContent == "" && f.PlainContent == "" {
		return &connectors.ValidationError{Message: "at least one of html_content or plain_content must be provided"}
	}
	return nil
}

// buildSingleSendBody constructs the JSON body for creating a single send.
// Scheduling is handled separately via the /schedule endpoint.
func buildSingleSendBody(fields *campaignFields) map[string]any {
	body := map[string]any{
		"name": fields.Name,
		"email_config": map[string]any{
			"subject":       fields.Subject,
			"sender_id":     fields.SenderID,
			"html_content":  fields.HTMLContent,
			"plain_content": fields.PlainContent,
		},
		"send_to": map[string]any{
			"list_ids": fields.ListIDs,
		},
	}
	if fields.SuppressionGroupID != 0 {
		emailConfig := body["email_config"].(map[string]any)
		emailConfig["suppression_group_id"] = fields.SuppressionGroupID
	}
	return body
}
