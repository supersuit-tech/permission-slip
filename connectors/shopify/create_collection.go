package shopify

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// createCollectionAction implements connectors.Action for shopify.create_collection.
// It creates a custom collection via POST /admin/api/2024-10/custom_collections.json.
type createCollectionAction struct {
	conn *ShopifyConnector
}

// createCollectionParams maps the JSON parameters for the create_collection action.
type createCollectionParams struct {
	Title     string                 `json:"title"`
	BodyHTML  string                 `json:"body_html,omitempty"`
	Published *bool                  `json:"published,omitempty"`
	SortOrder string                 `json:"sort_order,omitempty"`
	Image     map[string]interface{} `json:"image,omitempty"`
}

// validSortOrders are the sort orders accepted by the Shopify Custom Collections API.
// See: https://shopify.dev/docs/api/admin-rest/2024-10/resources/custom-collection
var validSortOrders = map[string]bool{
	"alpha-asc":       true,
	"alpha-desc":      true,
	"best-selling":    true,
	"created":         true,
	"created-desc":    true,
	"manual":          true,
	"price-asc":       true,
	"price-desc":      true,
}

func (p *createCollectionParams) validate() error {
	if p.Title == "" {
		return &connectors.ValidationError{Message: "missing required parameter: title"}
	}
	if p.SortOrder != "" && !validSortOrders[p.SortOrder] {
		return &connectors.ValidationError{
			Message: fmt.Sprintf("invalid sort_order %q: must be one of %s", p.SortOrder, sortedKeys(validSortOrders)),
		}
	}
	return nil
}

// Execute creates a custom collection in the Shopify store.
func (a *createCollectionAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createCollectionParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	collection := map[string]interface{}{
		"title": params.Title,
	}
	if params.BodyHTML != "" {
		collection["body_html"] = params.BodyHTML
	}
	if params.Published != nil {
		collection["published"] = *params.Published
	}
	if params.SortOrder != "" {
		collection["sort_order"] = params.SortOrder
	}
	if params.Image != nil {
		collection["image"] = params.Image
	}

	reqBody := map[string]interface{}{
		"custom_collection": collection,
	}

	var resp struct {
		CustomCollection json.RawMessage `json:"custom_collection"`
	}
	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, "/custom_collections.json", reqBody, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
