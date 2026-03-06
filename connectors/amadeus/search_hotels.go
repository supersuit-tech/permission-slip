package amadeus

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

// searchHotelsAction implements connectors.Action for amadeus.search_hotels.
// It performs a two-step hotel search:
//  1. GET /v1/reference-data/locations/hotels/by-city → hotel IDs
//  2. GET /v3/shopping/hotel-offers?hotelIds=...       → offers with prices
type searchHotelsAction struct {
	conn *AmadeusConnector
}

type searchHotelsParams struct {
	CityCode     string `json:"city_code"`
	Latitude     string `json:"latitude"`
	Longitude    string `json:"longitude"`
	CheckInDate  string `json:"check_in_date"`
	CheckOutDate string `json:"check_out_date"`
	Adults       int    `json:"adults"`
	RoomQuantity int    `json:"room_quantity"`
	Ratings      []int  `json:"ratings"`
	PriceRange   string `json:"price_range"`
	Currency     string `json:"currency"`
}

func (p *searchHotelsParams) validate() error {
	if p.CityCode == "" && (p.Latitude == "" || p.Longitude == "") {
		return &connectors.ValidationError{Message: "either city_code or both latitude and longitude are required"}
	}
	if p.CityCode != "" && !validIATACode(p.CityCode) {
		return &connectors.ValidationError{Message: "city_code must be a 3-letter IATA code (e.g., PAR)"}
	}
	if p.CheckInDate == "" {
		return &connectors.ValidationError{Message: "missing required parameter: check_in_date"}
	}
	if !validDate(p.CheckInDate) {
		return &connectors.ValidationError{Message: "check_in_date must be YYYY-MM-DD format"}
	}
	if p.CheckOutDate == "" {
		return &connectors.ValidationError{Message: "missing required parameter: check_out_date"}
	}
	if !validDate(p.CheckOutDate) {
		return &connectors.ValidationError{Message: "check_out_date must be YYYY-MM-DD format"}
	}
	for _, r := range p.Ratings {
		if r < 1 || r > 5 {
			return &connectors.ValidationError{Message: "ratings must be between 1 and 5"}
		}
	}
	return nil
}

func (a *searchHotelsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params searchHotelsParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	// Step 1: Get hotel IDs for the location.
	hotelIDs, err := a.fetchHotelIDs(ctx, req.Credentials, &params)
	if err != nil {
		return nil, err
	}
	if len(hotelIDs) == 0 {
		// No hotels found — return empty result rather than erroring.
		return connectors.JSONResult(map[string]any{"data": []any{}})
	}

	// Step 2: Get offers for those hotels.
	return a.fetchHotelOffers(ctx, req.Credentials, &params, hotelIDs)
}

// fetchHotelIDs calls the hotel list endpoint and returns up to 20 hotel IDs.
func (a *searchHotelsAction) fetchHotelIDs(ctx context.Context, creds connectors.Credentials, params *searchHotelsParams) ([]string, error) {
	var q url.Values

	if params.CityCode != "" {
		q = url.Values{"cityCode": {params.CityCode}}
	} else {
		q = url.Values{
			"latitude":  {params.Latitude},
			"longitude": {params.Longitude},
		}
	}

	if len(params.Ratings) > 0 {
		ratings := make([]string, len(params.Ratings))
		for i, r := range params.Ratings {
			ratings[i] = strconv.Itoa(r)
		}
		q.Set("ratings", strings.Join(ratings, ","))
	}

	var path string
	if params.CityCode != "" {
		path = "/v1/reference-data/locations/hotels/by-city?" + q.Encode()
	} else {
		path = "/v1/reference-data/locations/hotels/by-geocode?" + q.Encode()
	}

	var resp struct {
		Data []struct {
			HotelID string `json:"hotelId"`
		} `json:"data"`
	}
	if err := a.conn.do(ctx, creds, http.MethodGet, path, nil, &resp, nil); err != nil {
		return nil, err
	}

	// Limit to 20 hotel IDs to keep the offers query manageable.
	limit := 20
	if len(resp.Data) < limit {
		limit = len(resp.Data)
	}
	ids := make([]string, limit)
	for i := 0; i < limit; i++ {
		ids[i] = resp.Data[i].HotelID
	}
	return ids, nil
}

// fetchHotelOffers calls the hotel offers endpoint with the given hotel IDs.
func (a *searchHotelsAction) fetchHotelOffers(ctx context.Context, creds connectors.Credentials, params *searchHotelsParams, hotelIDs []string) (*connectors.ActionResult, error) {
	adults := params.Adults
	if adults < 1 {
		adults = 1
	}
	roomQty := params.RoomQuantity
	if roomQty < 1 {
		roomQty = 1
	}

	q := url.Values{
		"hotelIds":     {strings.Join(hotelIDs, ",")},
		"checkInDate":  {params.CheckInDate},
		"checkOutDate": {params.CheckOutDate},
		"adults":       {strconv.Itoa(adults)},
		"roomQuantity": {strconv.Itoa(roomQty)},
	}
	if params.PriceRange != "" {
		q.Set("priceRange", params.PriceRange)
	}
	if params.Currency != "" {
		q.Set("currency", params.Currency)
	}

	path := "/v3/shopping/hotel-offers?" + q.Encode()

	var resp struct {
		Data []json.RawMessage `json:"data"`
	}
	if err := a.conn.do(ctx, creds, http.MethodGet, path, nil, &resp, nil); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
