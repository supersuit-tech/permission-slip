package kroger

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// searchLocationsAction implements connectors.Action for kroger.search_locations.
// It finds stores via GET /v1/locations.
type searchLocationsAction struct {
	conn *KrogerConnector
}

type searchLocationsParams struct {
	ZipCode     string  `json:"zip_code"`
	Lat         float64 `json:"lat"`
	Lon         float64 `json:"lon"`
	RadiusMiles int     `json:"radius_miles"`
	Chain       string  `json:"chain"`
	Limit       int     `json:"limit"`
}

func (p *searchLocationsParams) validate() error {
	hasZip := p.ZipCode != ""
	hasCoords := p.Lat != 0 || p.Lon != 0
	if !hasZip && !hasCoords {
		return &connectors.ValidationError{Message: "at least one location filter is required: provide zip_code or lat/lon coordinates"}
	}
	if p.Lat != 0 || p.Lon != 0 {
		if p.Lat < -90 || p.Lat > 90 {
			return &connectors.ValidationError{Message: fmt.Sprintf("lat must be between -90 and 90 (got %.6f)", p.Lat)}
		}
		if p.Lon < -180 || p.Lon > 180 {
			return &connectors.ValidationError{Message: fmt.Sprintf("lon must be between -180 and 180 (got %.6f)", p.Lon)}
		}
	}
	if p.RadiusMiles != 0 && (p.RadiusMiles < 1 || p.RadiusMiles > 100) {
		return &connectors.ValidationError{Message: fmt.Sprintf("radius_miles must be between 1 and 100 (got %d)", p.RadiusMiles)}
	}
	if p.Limit != 0 && (p.Limit < 1 || p.Limit > 200) {
		return &connectors.ValidationError{Message: fmt.Sprintf("limit must be between 1 and 200 (got %d)", p.Limit)}
	}
	return nil
}

// Execute searches for Kroger store locations.
func (a *searchLocationsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params searchLocationsParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	limit := params.Limit
	if limit == 0 {
		limit = 10
	}

	path := "/locations?filter.limit=" + strconv.Itoa(limit)
	if params.ZipCode != "" {
		path += "&filter.zipCode.near=" + url.QueryEscape(params.ZipCode)
	}
	if params.Lat != 0 || params.Lon != 0 {
		path += "&filter.lat.near=" + strconv.FormatFloat(params.Lat, 'f', 6, 64) +
			"&filter.lon.near=" + strconv.FormatFloat(params.Lon, 'f', 6, 64)
	}
	if params.RadiusMiles > 0 {
		path += "&filter.radiusInMiles=" + strconv.Itoa(params.RadiusMiles)
	}
	if params.Chain != "" {
		path += "&filter.chain=" + url.QueryEscape(params.Chain)
	}

	var resp json.RawMessage
	if err := a.conn.do(ctx, req.Credentials, http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}

	return &connectors.ActionResult{Data: resp}, nil
}
