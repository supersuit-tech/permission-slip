package instacart

import (
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

const (
	// maxLineItemNameBytes caps each line item's name field so a single
	// oversized string cannot dominate request bodies or logs.
	maxLineItemNameBytes = 2048
)

// parseAndValidateProductsLinkParams unmarshals JSON, expands string
// line_items shorthand, and validates. Shared by Execute and ValidateRequest
// so approval-time validation and execution always apply the same rules.
func parseAndValidateProductsLinkParams(raw json.RawMessage) (createProductsLinkParams, error) {
	var params createProductsLinkParams
	if err := json.Unmarshal(raw, &params); err != nil {
		return params, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	params.LineItems = expandStringLineItemsInPlace(params.LineItems)
	if err := params.validate(); err != nil {
		return params, err
	}
	return params, nil
}
