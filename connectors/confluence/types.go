package confluence

import "github.com/supersuit-tech/permission-slip/connectors"

// pageResponse is the common response structure returned by the Confluence v2
// pages API (create, update, get). Shared across actions to avoid duplication.
type pageResponse struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Status  string `json:"status"`
	Version struct {
		Number  int    `json:"number"`
		Message string `json:"message,omitempty"`
	} `json:"version"`
	Links struct {
		WebUI string `json:"webui"`
	} `json:"_links"`
}

// toResult converts a pageResponse to an ActionResult, including a web_url
// field when the _links.webui path is available.
func (p *pageResponse) toResult(creds connectors.Credentials) (*connectors.ActionResult, error) {
	result := map[string]interface{}{
		"id":      p.ID,
		"title":   p.Title,
		"status":  p.Status,
		"version": p.Version,
	}
	if p.Links.WebUI != "" {
		site, _ := creds.Get("site")
		result["web_url"] = "https://" + site + ".atlassian.net/wiki" + p.Links.WebUI
	}
	return connectors.JSONResult(result)
}
