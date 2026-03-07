package twilio

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// lookupPhoneAction implements connectors.Action for twilio.lookup_phone.
// It retrieves phone number information via the Twilio Lookup API v2.
type lookupPhoneAction struct {
	conn *TwilioConnector
}

type lookupPhoneParams struct {
	PhoneNumber string `json:"phone_number"`
}

func (p *lookupPhoneParams) validate() error {
	if p.PhoneNumber == "" {
		return &connectors.ValidationError{Message: "missing required parameter: phone_number"}
	}
	if err := validateE164("phone_number", p.PhoneNumber); err != nil {
		return err
	}
	return nil
}

// Execute looks up information about a phone number.
func (a *lookupPhoneAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params lookupPhoneParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	reqURL := a.conn.lookupURL + "/PhoneNumbers/" + url.PathEscape(params.PhoneNumber)

	var resp struct {
		PhoneNumber    string `json:"phone_number"`
		CountryCode    string `json:"country_code"`
		NationalFormat string `json:"national_format"`
		Valid          bool   `json:"valid"`
		CallingCode    string `json:"calling_country_code"`
	}

	if err := a.conn.doGet(ctx, req.Credentials, reqURL, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
