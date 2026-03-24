package ticketmaster

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// searchEventsAction implements connectors.Action for ticketmaster.search_events.
type searchEventsAction struct {
	conn *TicketmasterConnector
}

type searchEventsParams struct {
	Keyword            string `json:"keyword"`
	LatLong            string `json:"latlong"`
	Radius             string `json:"radius"`
	Unit               string `json:"unit"`
	City               string `json:"city"`
	StateCode          string `json:"state_code"`
	CountryCode        string `json:"country_code"`
	PostalCode         string `json:"postal_code"`
	DMAID              string `json:"dma_id"`
	StartDateTime      string `json:"start_date_time"`
	EndDateTime        string `json:"end_date_time"`
	ClassificationName string `json:"classification_name"`
	ClassificationID   string `json:"classification_id"`
	SegmentID          string `json:"segment_id"`
	GenreID            string `json:"genre_id"`
	SubGenreID         string `json:"sub_genre_id"`
	Source             string `json:"source"`
	Sort               string `json:"sort"`
	Size               int    `json:"size"`
	Page               int    `json:"page"`
}

func (p *searchEventsParams) validate() error {
	if trimString(p.Keyword) == "" &&
		trimString(p.City) == "" &&
		trimString(p.PostalCode) == "" &&
		trimString(p.DMAID) == "" &&
		trimString(p.LatLong) == "" {
		return &connectors.ValidationError{Message: "provide at least one of: keyword, city, postal_code, dma_id, or latlong"}
	}
	if p.Size < 0 || p.Size > 200 {
		return &connectors.ValidationError{Message: "size must be between 0 and 200 (0 uses API default)"}
	}
	if p.Page < 0 {
		return &connectors.ValidationError{Message: "page must be zero or positive"}
	}
	if p.Unit != "" && p.Unit != "miles" && p.Unit != "km" {
		return &connectors.ValidationError{Message: "unit must be \"miles\" or \"km\" when set"}
	}
	return nil
}

func (a *searchEventsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params searchEventsParams
	if err := parseParams(req.Parameters, &params); err != nil {
		return nil, err
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	q := url.Values{}
	appendNonEmpty(q, "keyword", trimString(params.Keyword))
	appendNonEmpty(q, "latlong", trimString(params.LatLong))
	appendNonEmpty(q, "radius", trimString(params.Radius))
	appendNonEmpty(q, "unit", trimString(params.Unit))
	appendNonEmpty(q, "city", trimString(params.City))
	appendNonEmpty(q, "stateCode", trimString(params.StateCode))
	appendNonEmpty(q, "countryCode", trimString(params.CountryCode))
	appendNonEmpty(q, "postalCode", trimString(params.PostalCode))
	appendNonEmpty(q, "dmaId", trimString(params.DMAID))
	appendNonEmpty(q, "startDateTime", trimString(params.StartDateTime))
	appendNonEmpty(q, "endDateTime", trimString(params.EndDateTime))
	appendNonEmpty(q, "classificationName", trimString(params.ClassificationName))
	appendNonEmpty(q, "classificationId", trimString(params.ClassificationID))
	appendNonEmpty(q, "segmentId", trimString(params.SegmentID))
	appendNonEmpty(q, "genreId", trimString(params.GenreID))
	appendNonEmpty(q, "subGenreId", trimString(params.SubGenreID))
	appendNonEmpty(q, "source", trimString(params.Source))
	appendNonEmpty(q, "sort", trimString(params.Sort))
	if params.Size > 0 {
		q.Set("size", strconv.Itoa(params.Size))
	}
	if params.Page > 0 {
		q.Set("page", strconv.Itoa(params.Page))
	}

	var out json.RawMessage
	if err := a.conn.doGET(ctx, req.Credentials, "events.json", q, &out); err != nil {
		return nil, err
	}
	return &connectors.ActionResult{Data: out}, nil
}
