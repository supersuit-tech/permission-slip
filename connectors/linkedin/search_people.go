package linkedin

// searchPeopleAction implements connectors.Action for linkedin.search_people.
//
// # Access tier requirements
//
// LinkedIn's People Search API requires Marketing Developer Platform (MDP)
// or Sales Navigator API access. Standard OAuth apps receive HTTP 403 for
// most search queries. Document which access tier is in use when configuring
// this action.
//
// LinkedIn API reference:
// https://learn.microsoft.com/en-us/linkedin/shared/integrations/people/people-search-api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

type searchPeopleAction struct {
	conn *LinkedInConnector
}

type searchPeopleParams struct {
	Keywords string `json:"keywords"`
	Company  string `json:"company"`
	Title    string `json:"title"`
	Count    int    `json:"count"`
	Start    int    `json:"start"`
}

const defaultPeopleCount = 10
const maxPeopleCount = 50

func (p *searchPeopleParams) validate() error {
	if p.Keywords == "" && p.Company == "" && p.Title == "" {
		return &connectors.ValidationError{Message: "at least one of keywords, company, or title is required"}
	}
	if p.Count < 0 {
		return &connectors.ValidationError{Message: "count must be non-negative"}
	}
	if p.Count > maxPeopleCount {
		return &connectors.ValidationError{Message: fmt.Sprintf("count must not exceed %d", maxPeopleCount)}
	}
	if p.Start < 0 {
		return &connectors.ValidationError{Message: "start must be non-negative"}
	}
	return nil
}

// peopleSearchResponse is the LinkedIn people search API response.
type peopleSearchResponse struct {
	Elements []peopleSearchElement `json:"elements"`
	Paging   searchPaging          `json:"paging"`
}

type peopleSearchElement struct {
	ID        string `json:"id"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Headline  string `json:"headline"`
}

// Execute searches LinkedIn members by keywords, company, and/or title.
func (a *searchPeopleAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params searchPeopleParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	count := params.Count
	if count == 0 {
		count = defaultPeopleCount
	}

	q := url.Values{}
	q.Set("q", "search")
	if params.Keywords != "" {
		q.Set("keywords", params.Keywords)
	}
	if params.Company != "" {
		q.Set("facetCurrentCompany", params.Company)
	}
	if params.Title != "" {
		q.Set("facetTitle", params.Title)
	}
	q.Set("count", strconv.Itoa(count))
	q.Set("start", strconv.Itoa(params.Start))

	apiURL := a.conn.restBaseURL + "/people?" + q.Encode()

	var resp peopleSearchResponse
	if err := a.conn.do(ctx, req.Credentials, http.MethodGet, apiURL, nil, &resp, true); err != nil {
		return nil, err
	}

	results := make([]map[string]any, 0, len(resp.Elements))
	for _, el := range resp.Elements {
		results = append(results, map[string]any{
			"id":         el.ID,
			"first_name": el.FirstName,
			"last_name":  el.LastName,
			"headline":   el.Headline,
		})
	}

	return connectors.JSONResult(map[string]any{
		"results": results,
		"total":   resp.Paging.Total,
		"start":   resp.Paging.Start,
		"count":   resp.Paging.Count,
	})
}
