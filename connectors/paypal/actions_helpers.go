package paypal

import (
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// parseParams unmarshals req.Parameters into dest (a pointer). Centralizes the
// shared "invalid parameters" validation message for all actions.
func parseParams(req connectors.ActionRequest, dest any) error {
	if err := json.Unmarshal(req.Parameters, dest); err != nil {
		return &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	return nil
}

// optionalJSONObject returns nil when raw is empty; otherwise parses a JSON object.
func optionalJSONObject(raw json.RawMessage, fieldName string) (map[string]any, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	return readJSONBody(raw, fieldName)
}
