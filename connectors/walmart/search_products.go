package walmart

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// searchProductsAction implements connectors.Action for walmart.search_products.
// It searches products via GET /search with keyword and optional category filtering.
type searchProductsAction struct {
	conn *WalmartConnector
}

// searchProductsParams maps the JSON parameters for walmart.search_products.
// Limit defaults to 10 when omitted or zero. Walmart caps results at 25 per page.
type searchProductsParams struct {
	Query      string `json:"query"`
	CategoryID string `json:"category_id"`
	Sort       string `json:"sort"`
	Order      string `json:"order"`
	Start      int    `json:"start"`
	Limit      int    `json:"limit"`
}

// validSortFields lists the Walmart API's accepted sort values.
var validSortFields = map[string]bool{
	"relevance": true,
	"price":     true,
	"title":     true,
	"bestseller": true,
	"customerRating": true,
	"new": true,
}

var validOrderValues = map[string]bool{
	"asc":  true,
	"desc": true,
}

func (p *searchProductsParams) validate() error {
	if p.Query == "" {
		return &connectors.ValidationError{Message: "missing required parameter: query (e.g. \"paper towels\", \"laptop\")"}
	}
	if len(p.Query) > 500 {
		return &connectors.ValidationError{Message: fmt.Sprintf("query exceeds 500 characters (got %d)", len(p.Query))}
	}
	if p.Sort != "" && !validSortFields[p.Sort] {
		return &connectors.ValidationError{Message: fmt.Sprintf("invalid sort %q: must be relevance, price, title, bestseller, customerRating, or new", p.Sort)}
	}
	if p.Order != "" && !validOrderValues[p.Order] {
		return &connectors.ValidationError{Message: fmt.Sprintf("invalid order %q: must be asc or desc", p.Order)}
	}
	if p.Limit == 0 {
		p.Limit = 10
	}
	if p.Limit < 1 || p.Limit > 25 {
		return &connectors.ValidationError{Message: fmt.Sprintf("limit must be between 1 and 25, got %d", p.Limit)}
	}
	if p.Start < 0 {
		return &connectors.ValidationError{Message: "start must be non-negative"}
	}
	return nil
}

func (a *searchProductsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params searchProductsParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	q := url.Values{}
	q.Set("query", params.Query)
	q.Set("numItems", strconv.Itoa(params.Limit))
	q.Set("format", "json")
	if params.CategoryID != "" {
		q.Set("categoryId", params.CategoryID)
	}
	if params.Sort != "" {
		q.Set("sort", params.Sort)
	}
	if params.Order != "" {
		q.Set("order", params.Order)
	}
	if params.Start > 0 {
		q.Set("start", strconv.Itoa(params.Start))
	}

	var resp json.RawMessage
	if err := a.conn.do(ctx, req.Credentials, http.MethodGet, "/search?"+q.Encode(), &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(json.RawMessage(resp))
}
