package ticketmaster

import (
	_ "embed"
	"encoding/json"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

//go:embed logo.svg
var logoSVG string

// Manifest returns connector metadata for DB seeding.
func (c *TicketmasterConnector) Manifest() *connectors.ConnectorManifest {
	return &connectors.ConnectorManifest{
		ID:          "ticketmaster",
		Name:        "Ticketmaster",
		Description: "Ticketmaster Discovery API for event search, venues, attractions, genres, and purchase links",
		LogoSVG:     logoSVG,
		Actions: []connectors.ManifestAction{
			{
				ActionType:  "ticketmaster.search_events",
				Name:        "Search events",
				Description: "Search events by keyword, location (lat/long, city, DMA, postal code), date range, and genre filters. Returns Ticketmaster event payloads including purchase URLs.",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"properties": {
						"keyword": {"type": "string", "description": "Free-text search keyword"},
						"latlong": {"type": "string", "description": "Latitude and longitude as \"lat,long\" (e.g. \"34.0522,-118.2437\")"},
						"radius": {"type": "string", "description": "Search radius (use with latlong)"},
						"unit": {"type": "string", "enum": ["miles", "km"], "description": "Radius unit"},
						"city": {"type": "string", "description": "City name"},
						"state_code": {"type": "string", "description": "State or region code (e.g. CA)"},
						"country_code": {"type": "string", "description": "ISO country code (e.g. US)"},
						"postal_code": {"type": "string", "description": "Postal or ZIP code"},
						"dma_id": {"type": "string", "description": "Designated Market Area ID"},
						"start_date_time": {"type": "string", "description": "Start of date/time range (ISO-8601 with offset, e.g. 2026-06-01T00:00:00Z)"},
						"end_date_time": {"type": "string", "description": "End of date/time range (ISO-8601 with offset)"},
						"classification_name": {"type": "string", "description": "Filter by classification name (e.g. music, sports)"},
						"classification_id": {"type": "string", "description": "Filter by classification ID"},
						"segment_id": {"type": "string", "description": "Filter by segment ID"},
						"genre_id": {"type": "string", "description": "Filter by genre ID"},
						"sub_genre_id": {"type": "string", "description": "Filter by sub-genre ID"},
						"source": {"type": "string", "description": "Ticket source filter (e.g. ticketmaster, universe, frontgate, tmr)"},
						"sort": {"type": "string", "description": "Sort order (e.g. date,asc)"},
						"size": {"type": "integer", "description": "Page size (1–200; omit for API default)"},
						"page": {"type": "integer", "description": "Zero-based page number"}
					}
				}`)),
			},
			{
				ActionType:  "ticketmaster.get_event",
				Name:        "Get event",
				Description: "Get full event details: performers, venue, dates, price ranges, on-sale dates, and Ticketmaster purchase URL",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["event_id"],
					"properties": {
						"event_id": {"type": "string", "description": "Ticketmaster event ID"},
						"locale": {"type": "string", "description": "Locale (e.g. en-us)"}
					}
				}`)),
			},
			{
				ActionType:  "ticketmaster.search_venues",
				Name:        "Search venues",
				Description: "Search venues by keyword or location (lat/long, city, postal code)",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"properties": {
						"keyword": {"type": "string", "description": "Venue name or keyword"},
						"latlong": {"type": "string", "description": "\"lat,long\" pair"},
						"radius": {"type": "string", "description": "Search radius"},
						"unit": {"type": "string", "enum": ["miles", "km"]},
						"country_code": {"type": "string"},
						"state_code": {"type": "string"},
						"city": {"type": "string"},
						"postal_code": {"type": "string"},
						"source": {"type": "string"},
						"sort": {"type": "string"},
						"size": {"type": "integer"},
						"page": {"type": "integer"}
					}
				}`)),
			},
			{
				ActionType:  "ticketmaster.get_venue",
				Name:        "Get venue",
				Description: "Get venue details: address, capacity, images, and related metadata",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["venue_id"],
					"properties": {
						"venue_id": {"type": "string", "description": "Ticketmaster venue ID"},
						"locale": {"type": "string"}
					}
				}`)),
			},
			{
				ActionType:  "ticketmaster.search_attractions",
				Name:        "Search attractions",
				Description: "Search artists, teams, shows, and other attractions; filter by genre IDs from classifications",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"properties": {
						"keyword": {"type": "string"},
						"classification_id": {"type": "string"},
						"segment_id": {"type": "string"},
						"genre_id": {"type": "string"},
						"sub_genre_id": {"type": "string"},
						"source": {"type": "string"},
						"sort": {"type": "string"},
						"size": {"type": "integer"},
						"page": {"type": "integer"}
					}
				}`)),
			},
			{
				ActionType:  "ticketmaster.get_attraction",
				Name:        "Get attraction",
				Description: "Get attraction (performer) details and metadata",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["attraction_id"],
					"properties": {
						"attraction_id": {"type": "string"},
						"locale": {"type": "string"}
					}
				}`)),
			},
			{
				ActionType:  "ticketmaster.list_classifications",
				Name:        "List classifications",
				Description: "Browse genre hierarchy (segments, genres, sub-genres). Omit classification_id for the full tree, or pass classification_id for one branch",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"properties": {
						"classification_id": {"type": "string", "description": "Optional segment or genre ID to fetch that subtree"},
						"locale": {"type": "string"},
						"source": {"type": "string"}
					}
				}`)),
			},
			{
				ActionType:  "ticketmaster.suggest",
				Name:        "Suggest",
				Description: "Autocomplete suggestions for events, venues, and attractions (Discovery suggest API)",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["keyword"],
					"properties": {
						"keyword": {"type": "string", "description": "Partial search text"},
						"source": {"type": "string", "description": "Restrict suggestion sources"},
						"locale": {"type": "string"}
					}
				}`)),
			},
		},
		RequiredCredentials: []connectors.ManifestCredential{
			{Service: "ticketmaster", AuthType: "api_key", InstructionsURL: "https://developer.ticketmaster.com/products-and-docs/apis/getting-started/"},
		},
		Templates: []connectors.ManifestTemplate{
			{
				ID:          "tpl_ticketmaster_search_events_read",
				ActionType:  "ticketmaster.search_events",
				Name:        "Search events (read-only)",
				Description: "Search Ticketmaster events by keyword and city or coordinates. No purchases.",
				Parameters:  json.RawMessage(`{"keyword":"*","city":"*","start_date_time":"*","end_date_time":"*","size":"*"}`),
			},
			{
				ID:          "tpl_ticketmaster_get_event",
				ActionType:  "ticketmaster.get_event",
				Name:        "Look up one event",
				Description: "Fetch a single event by ID (pricing, venue, purchase link).",
				Parameters:  json.RawMessage(`{"event_id":"*"}`),
			},
			{
				ID:          "tpl_ticketmaster_suggest",
				ActionType:  "ticketmaster.suggest",
				Name:        "Autocomplete",
				Description: "Suggest completions for a user-typed query.",
				Parameters:  json.RawMessage(`{"keyword":"*"}`),
			},
		},
	}
}
