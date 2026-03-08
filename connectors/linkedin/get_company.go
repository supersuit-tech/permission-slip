package linkedin

// getCompanyAction implements connectors.Action for linkedin.get_company.
//
// # Access tier requirements
//
// Basic organization profile lookup via GET /rest/organizations/{id} is
// available to apps with r_organization_social scope. Some fields (e.g.
// employee count, industries) may require Marketing Developer Platform access.
//
// LinkedIn API reference:
// https://learn.microsoft.com/en-us/linkedin/marketing/integrations/community-management/organizations/organization-lookup-api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

type getCompanyAction struct {
	conn *LinkedInConnector
}

type getCompanyParams struct {
	OrganizationID string `json:"organization_id"`
}

func (p *getCompanyParams) validate() error {
	if p.OrganizationID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: organization_id"}
	}
	if err := validateOrganizationID(p.OrganizationID); err != nil {
		return err
	}
	return nil
}

// localizedString is a LinkedIn API localized text envelope. Both name and
// description fields in organization responses share this structure.
type localizedString struct {
	Localized       map[string]string `json:"localized"`
	PreferredLocale preferredLocale   `json:"preferredLocale"`
}

// localizedName and localizedDescription are type aliases for localizedString.
// LinkedIn API fields of both types share the same JSON shape.
type (
	localizedName        = localizedString
	localizedDescription = localizedString
)

type preferredLocale struct {
	Country  string `json:"country"`
	Language string `json:"language"`
}

// organizationResponse is the LinkedIn organization profile response.
type organizationResponse struct {
	ID          string                 `json:"id"`
	Name        localizedName          `json:"name"`
	Description localizedDescription   `json:"description"`
	StaffCount  int                    `json:"staffCount"`
	Locations   []organizationLocation `json:"locations"`
	Industries  []industryTag          `json:"industries"`
	WebsiteURL  string                 `json:"websiteUrl"`
	FoundedOn   foundedOn              `json:"foundedOn"`
}

type organizationLocation struct {
	LocalizedName string `json:"localizedName"`
}

type industryTag struct {
	LocalizedName string `json:"localizedName"`
}

type foundedOn struct {
	Year int `json:"year"`
}

// Execute retrieves a LinkedIn company profile by organization ID.
func (a *getCompanyAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params getCompanyParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	apiURL := a.conn.restBaseURL + "/organizations/" + params.OrganizationID

	var resp organizationResponse
	if err := a.conn.do(ctx, req.Credentials, http.MethodGet, apiURL, nil, &resp, true); err != nil {
		return nil, err
	}

	// Extract the preferred locale name, falling back to any available name.
	name := preferredString(resp.Name)
	description := preferredString(resp.Description)

	locations := make([]string, 0, len(resp.Locations))
	for _, loc := range resp.Locations {
		if loc.LocalizedName != "" {
			locations = append(locations, loc.LocalizedName)
		}
	}

	industries := make([]string, 0, len(resp.Industries))
	for _, ind := range resp.Industries {
		if ind.LocalizedName != "" {
			industries = append(industries, ind.LocalizedName)
		}
	}

	return connectors.JSONResult(map[string]any{
		"id":               params.OrganizationID,
		"organization_urn": "urn:li:organization:" + params.OrganizationID,
		"name":             name,
		"description":      description,
		"staff_count":      resp.StaffCount,
		"locations":        locations,
		"industries":       industries,
		"website_url":      resp.WebsiteURL,
		"founded_year":     resp.FoundedOn.Year,
	})
}

// preferredString extracts the localized string for the preferred locale from
// a localizedString envelope. Falls back to the lexicographically first key
// when the preferred locale is not found. Sorting keys ensures a deterministic
// result regardless of Go map iteration order.
func preferredString(ls localizedString) string {
	key := ls.PreferredLocale.Language + "_" + ls.PreferredLocale.Country
	if v, ok := ls.Localized[key]; ok {
		return v
	}
	keys := make([]string, 0, len(ls.Localized))
	for k := range ls.Localized {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	if len(keys) > 0 {
		return ls.Localized[keys[0]]
	}
	return ""
}
