package ticketmaster

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// searchAttractionsAction implements connectors.Action for ticketmaster.search_attractions.
type searchAttractionsAction struct {
	conn *TicketmasterConnector
}

type searchAttractionsParams struct {
	Keyword          string `json:"keyword"`
	ClassificationID string `json:"classification_id"`
	SegmentID        string `json:"segment_id"`
	GenreID          string `json:"genre_id"`
	SubGenreID       string `json:"sub_genre_id"`
	Source           string `json:"source"`
	Sort             string `json:"sort"`
	Size             int    `json:"size"`
	Page             int    `json:"page"`
}

func (p *searchAttractionsParams) validate() error {
	if trimString(p.Keyword) == "" && trimString(p.ClassificationID) == "" &&
		trimString(p.SegmentID) == "" && trimString(p.GenreID) == "" && trimString(p.SubGenreID) == "" {
		return &connectors.ValidationError{Message: "provide at least one of: keyword, classification_id, segment_id, genre_id, or sub_genre_id"}
	}
	if p.Size < 0 || p.Size > 200 {
		return &connectors.ValidationError{Message: "size must be between 0 and 200 (0 uses API default)"}
	}
	if p.Page < 0 {
		return &connectors.ValidationError{Message: "page must be zero or positive"}
	}
	return nil
}

func (a *searchAttractionsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params searchAttractionsParams
	if err := parseParams(req.Parameters, &params); err != nil {
		return nil, err
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	q := url.Values{}
	appendNonEmpty(q, "keyword", trimString(params.Keyword))
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
	if err := a.conn.doGET(ctx, req.Credentials, "attractions.json", q, &out); err != nil {
		return nil, err
	}
	return &connectors.ActionResult{Data: out}, nil
}
