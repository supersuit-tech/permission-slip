package plaid

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// getInstitutionAction implements connectors.Action for plaid.get_institution.
// It retrieves institution details via POST /institutions/get_by_id.
type getInstitutionAction struct {
	conn *PlaidConnector
}

type getInstitutionParams struct {
	InstitutionID string   `json:"institution_id"`
	CountryCodes  []string `json:"country_codes"`
}

func (p *getInstitutionParams) validate() error {
	if p.InstitutionID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: institution_id"}
	}
	return nil
}

// Execute retrieves institution details and returns the institution data.
func (a *getInstitutionAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params getInstitutionParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	countryCodes := params.CountryCodes
	if len(countryCodes) == 0 {
		countryCodes = []string{"US"}
	}

	body := map[string]any{
		"institution_id": params.InstitutionID,
		"country_codes":  countryCodes,
	}

	var resp json.RawMessage
	if err := a.conn.doPost(ctx, req.Credentials, "/institutions/get_by_id", body, &resp); err != nil {
		return nil, err
	}

	return &connectors.ActionResult{Data: resp}, nil
}
