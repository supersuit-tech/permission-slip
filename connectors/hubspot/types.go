package hubspot

// hubspotObjectRequest is the standard request body for creating/updating
// HubSpot CRM objects. All CRM objects use the same {"properties": {...}} shape.
type hubspotObjectRequest struct {
	Properties map[string]string `json:"properties"`
}

// hubspotObjectResponse is the standard response body from HubSpot CRM
// object create/update/get endpoints.
type hubspotObjectResponse struct {
	ID         string            `json:"id"`
	Properties map[string]string `json:"properties"`
}

// mergeProperties copies entries from extra into a new map, then applies
// overrides on top. This implements the convention where explicit action
// parameters take precedence over the catch-all "properties" map.
func mergeProperties(extra map[string]string, overrides map[string]string) map[string]string {
	props := make(map[string]string, len(extra)+len(overrides))
	for k, v := range extra {
		props[k] = v
	}
	for k, v := range overrides {
		props[k] = v
	}
	return props
}
