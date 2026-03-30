package square

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// listLocationsAction implements connectors.Action for square.list_locations.
// It lists all business locations via GET /v2/locations.
type listLocationsAction struct {
	conn *SquareConnector
}

// Execute retrieves all locations for the Square account.
func (a *listLocationsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	// No parameters needed for this endpoint; unmarshal to validate JSON.
	var params map[string]interface{}
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}

	var resp struct {
		Locations json.RawMessage `json:"locations"`
	}
	if err := a.conn.do(ctx, req.Credentials, http.MethodGet, "/locations", nil, &resp); err != nil {
		return nil, err
	}

	locations := json.RawMessage(resp.Locations)
	if len(locations) == 0 || string(locations) == "null" {
		locations = json.RawMessage("[]")
	}

	return connectors.JSONResult(map[string]interface{}{
		"locations": locations,
	})
}
