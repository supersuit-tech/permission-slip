package expedia

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// searchHotelsAction implements connectors.Action for expedia.search_hotels.
// It searches available hotels with pricing via GET /v3/properties/availability.
type searchHotelsAction struct {
	conn *ExpediaConnector
}

// searchHotelsParams are the parameters parsed from ActionRequest.Parameters.
type searchHotelsParams struct {
	Checkin    string    `json:"checkin"`
	Checkout   string    `json:"checkout"`
	Occupancy  string    `json:"occupancy"`
	RegionID   string    `json:"region_id"`
	Latitude   *float64  `json:"latitude"`
	Longitude  *float64  `json:"longitude"`
	Currency   string    `json:"currency"`
	Language   string    `json:"language"`
	SortBy     string    `json:"sort_by"`
	StarRating []int     `json:"star_rating"`
	Limit      int       `json:"limit"`
}

var validSortBy = map[string]bool{
	"price":    true,
	"distance": true,
	"rating":   true,
}

func (p *searchHotelsParams) validate() error {
	if p.Checkin == "" {
		return &connectors.ValidationError{Message: "missing required parameter: checkin"}
	}
	if p.Checkout == "" {
		return &connectors.ValidationError{Message: "missing required parameter: checkout"}
	}
	if err := validateDate("checkin", p.Checkin); err != nil {
		return err
	}
	if err := validateDate("checkout", p.Checkout); err != nil {
		return err
	}
	if err := validateDateRange(p.Checkin, p.Checkout); err != nil {
		return err
	}
	if p.Occupancy == "" {
		return &connectors.ValidationError{Message: "missing required parameter: occupancy"}
	}
	if err := validateOccupancy(p.Occupancy); err != nil {
		return err
	}
	if p.RegionID == "" && p.Latitude == nil && p.Longitude == nil {
		return &connectors.ValidationError{Message: "either region_id or latitude+longitude is required"}
	}
	if (p.Latitude == nil) != (p.Longitude == nil) {
		return &connectors.ValidationError{Message: "both latitude and longitude must be provided together"}
	}
	if p.SortBy != "" && !validSortBy[p.SortBy] {
		return &connectors.ValidationError{Message: fmt.Sprintf("invalid sort_by %q: must be price, distance, or rating", p.SortBy)}
	}
	if len(p.StarRating) > 5 {
		return &connectors.ValidationError{Message: "star_rating cannot have more than 5 entries (valid star ratings are 1-5)"}
	}
	if p.Limit > 200 {
		return &connectors.ValidationError{Message: "limit cannot exceed 200"}
	}
	return nil
}

// Execute searches for available hotels and returns property results with rates.
func (a *searchHotelsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params searchHotelsParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	q := url.Values{}
	q.Set("checkin", params.Checkin)
	q.Set("checkout", params.Checkout)
	q.Set("occupancy", params.Occupancy)
	if params.RegionID != "" {
		q.Set("region_id", params.RegionID)
	}
	if params.Latitude != nil {
		q.Set("latitude", strconv.FormatFloat(*params.Latitude, 'f', -1, 64))
	}
	if params.Longitude != nil {
		q.Set("longitude", strconv.FormatFloat(*params.Longitude, 'f', -1, 64))
	}
	if params.Currency != "" {
		q.Set("currency", params.Currency)
	}
	if params.Language != "" {
		q.Set("language", params.Language)
	}
	if params.SortBy != "" {
		q.Set("sort_by", params.SortBy)
	}
	if len(params.StarRating) > 0 {
		ratings := make([]string, len(params.StarRating))
		for i, r := range params.StarRating {
			ratings[i] = strconv.Itoa(r)
		}
		q.Set("star_rating", strings.Join(ratings, ","))
	}
	if params.Limit > 0 {
		q.Set("limit", strconv.Itoa(params.Limit))
	}

	path := "/v3/properties/availability?" + q.Encode()

	var resp json.RawMessage
	if err := a.conn.do(ctx, req.Credentials, http.MethodGet, path, defaultCustomerIP, nil, &resp); err != nil {
		return nil, err
	}

	return &connectors.ActionResult{Data: resp}, nil
}
