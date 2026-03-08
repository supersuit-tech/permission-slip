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

type localizedName struct {
	Localized        map[string]string `json:"localized"`
	PreferredLocale  preferredLocale   `json:"preferredLocale"`
}

type localizedDescription struct {
	Localized        map[string]string `json:"localized"`
	PreferredLocale  preferredLocale   `json:"preferredLocale"`
}

type preferredLocale struct {
	Country  string `json:"country"`
	Language string `json:"language"`
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
	name := preferredString(resp.Name.Localized, resp.Name.PreferredLocale)
	description := preferredString(resp.Description.Localized, resp.Description.PreferredLocale)

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
		"id":              params.OrganizationID,
		"name":            name,
		"description":     description,
		"staff_count":     resp.StaffCount,
		"locations":       locations,
		"industries":      industries,
		"website_url":     resp.WebsiteURL,
		"founded_year":    resp.FoundedOn.Year,
	})
}

// preferredString extracts the localized string for the preferred locale,
// falling back to any available value when the preferred locale is not found.
func preferredString(localized map[string]string, locale preferredLocale) string {
	key := locale.Language + "_" + locale.Country
	if v, ok := localized[key]; ok {
		return v
	}
	for _, v := range localized {
		return v
	}
	return ""
}
