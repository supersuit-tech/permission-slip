package linkedin

// searchCompaniesAction implements connectors.Action for linkedin.search_companies.
//
// # Access tier requirements
//
// LinkedIn's Organization Search API requires Marketing Developer Platform
// (MDP) access. Standard OAuth apps may receive HTTP 403 for search queries.
//
// LinkedIn API reference:
// https://learn.microsoft.com/en-us/linkedin/marketing/integrations/community-management/organizations/organization-lookup-api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

type searchCompaniesAction struct {
	conn *LinkedInConnector
}

type searchCompaniesParams struct {
	Keywords string `json:"keywords"`
	Count    int    `json:"count"`
	Start    int    `json:"start"`
}

func (p *searchCompaniesParams) validate() error {
	if p.Keywords == "" {
		return &connectors.ValidationError{Message: "missing required parameter: keywords"}
	}
	return validateCountStart(p.Count, maxSearchCount, p.Start)
}

// companySearchResponse is the LinkedIn organization search API response.
type companySearchResponse struct {
	Elements []companySearchElement `json:"elements"`
	Paging   searchPaging           `json:"paging"`
}

type companySearchElement struct {
	ID          string `json:"id"`
	Name        string `json:"localizedName"`
	Description string `json:"localizedDescription"`
	StaffCount  int    `json:"staffCount"`
}

// Execute searches LinkedIn company pages by keyword.
func (a *searchCompaniesAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params searchCompaniesParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	count := resolveCount(params.Count, defaultSearchCount)

	q := url.Values{}
	q.Set("q", "search")
	q.Set("keywords", params.Keywords)
	q.Set("count", strconv.Itoa(count))
	q.Set("start", strconv.Itoa(params.Start))

	apiURL := a.conn.restBaseURL + "/organizations?" + q.Encode()

	var resp companySearchResponse
	if err := a.conn.do(ctx, req.Credentials, http.MethodGet, apiURL, nil, &resp, true); err != nil {
		return nil, err
	}

	results := make([]map[string]any, 0, len(resp.Elements))
	for _, el := range resp.Elements {
		results = append(results, map[string]any{
			"id":               el.ID,
			"organization_urn": "urn:li:organization:" + el.ID,
			"name":             el.Name,
			"description":      el.Description,
			"staff_count":      el.StaffCount,
		})
	}

	return connectors.JSONResult(map[string]any{
		"results":    results,
		"total":      resp.Paging.Total,
		"start":      resp.Paging.Start,
		"count":      len(results),
		"next_start": nextStart(params.Start, len(results)),
	})
}
