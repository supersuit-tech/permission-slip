package expedia

// This file is split from expedia.go to keep the main connector file focused
// on struct, auth, and HTTP lifecycle. The manifest contains 6 action schemas
// and 6 templates (~240 lines of JSON Schema definitions) that would obscure
// the business logic if inlined in the connector file.

import (
	_ "embed"
	"encoding/json"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// Manifest returns the connector's metadata manifest. Used by the server to
// auto-seed DB rows on startup. Returns action schemas, required credentials,
// and configuration templates.
//go:embed logo.svg
var logoSVG string

func (c *ExpediaConnector) Manifest() *connectors.ConnectorManifest {
	return &connectors.ConnectorManifest{
		ID:          "expedia",
		Name:        "Expedia Rapid",
		Description: "Expedia Rapid API integration for hotel search and booking",
		LogoSVG:     logoSVG,
		Actions: []connectors.ManifestAction{
			{
				ActionType:  "expedia.search_hotels",
				Name:        "Search Hotels",
				Description: "Search available hotels with pricing",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["checkin", "checkout", "occupancy"],
					"properties": {
						"checkin": {
							"type": "string",
							"format": "date",
							"description": "Check-in date (YYYY-MM-DD)",
							"x-ui": {
								"label": "Check-in date",
								"widget": "date",
								"datetime_range_pair": "checkout",
								"datetime_range_role": "lower"
							}
						},
						"checkout": {
							"type": "string",
							"format": "date",
							"description": "Check-out date (YYYY-MM-DD)",
							"x-ui": {
								"label": "Check-out date",
								"widget": "date",
								"datetime_range_pair": "checkin",
								"datetime_range_role": "upper"
							}
						},
						"region_id": {
							"type": "string",
							"description": "Expedia region ID to search in",
							"x-ui": {"label": "Region ID", "help_text": "Expedia region ID"}
						},
						"latitude": {
							"type": "number",
							"description": "Latitude for location-based search",
							"x-ui": {"label": "Latitude"}
						},
						"longitude": {
							"type": "number",
							"description": "Longitude for location-based search",
							"x-ui": {"label": "Longitude"}
						},
						"occupancy": {
							"type": "string",
							"description": "Occupancy string (e.g. '2' for 2 adults, '2-0,4' for 2 adults + 1 child age 4)",
							"x-ui": {"label": "Occupants", "help_text": "Format: adults=2 or adults=2,children_ages=5,7"}
						},
						"currency": {
							"type": "string",
							"description": "Currency code (e.g. USD, EUR)",
							"x-ui": {"label": "Currency", "placeholder": "USD"}
						},
						"language": {
							"type": "string",
							"description": "Language code (e.g. en-US)",
							"x-ui": {"label": "Language", "placeholder": "en-US"}
						},
						"sort_by": {
							"type": "string",
							"enum": ["price", "distance", "rating"],
							"description": "Sort results by price, distance, or rating",
							"x-ui": {"label": "Sort by", "widget": "select"}
						},
						"star_rating": {
							"type": "array",
							"items": {"type": "integer"},
							"description": "Filter by star rating(s)",
							"x-ui": {"label": "Star rating"}
						},
						"limit": {
							"type": "integer",
							"default": 20,
							"description": "Maximum number of results to return",
							"x-ui": {"label": "Max results"}
						}
					}
				}`)),
			},
			{
				ActionType:  "expedia.get_hotel",
				Name:        "Get Hotel Details",
				Description: "Get full hotel details: photos, amenities, room types, policies, reviews summary",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["property_id"],
					"properties": {
						"property_id": {
							"type": "string",
							"description": "Expedia property ID",
							"x-ui": {"label": "Property ID", "help_text": "Expedia property ID — from search results"}
						},
						"checkin": {
							"type": "string",
							"format": "date",
							"description": "Check-in date (YYYY-MM-DD) for rate information",
							"x-ui": {
								"label": "Check-in date",
								"widget": "date",
								"datetime_range_pair": "checkout",
								"datetime_range_role": "lower"
							}
						},
						"checkout": {
							"type": "string",
							"format": "date",
							"description": "Check-out date (YYYY-MM-DD) for rate information",
							"x-ui": {
								"label": "Check-out date",
								"widget": "date",
								"datetime_range_pair": "checkin",
								"datetime_range_role": "upper"
							}
						},
						"occupancy": {
							"type": "string",
							"description": "Occupancy string for rate information",
							"x-ui": {"label": "Occupants", "help_text": "Format: adults=2 or adults=2,children_ages=5,7"}
						}
					}
				}`)),
			},
			{
				ActionType:  "expedia.price_check",
				Name:        "Price Check",
				Description: "Confirm real-time pricing and availability before booking",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["room_id"],
					"properties": {
						"room_id": {
							"type": "string",
							"description": "Room ID from search results",
							"x-ui": {"label": "Room ID", "help_text": "From expedia.get_hotel or expedia.price_check"}
						}
					}
				}`)),
			},
			{
				ActionType:  "expedia.create_booking",
				Name:        "Create Booking",
				Description: "Book a hotel room. High risk — creates a real reservation and may charge payment.",
				RiskLevel:   "high",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["room_id", "given_name", "family_name", "email", "phone", "payment_method_id"],
					"properties": {
						"room_id": {
							"type": "string",
							"description": "Room ID from a successful price check",
							"x-ui": {"label": "Room ID", "help_text": "From expedia.get_hotel or expedia.price_check"}
						},
						"given_name": {
							"type": "string",
							"description": "Guest first name",
							"x-ui": {"label": "First name", "placeholder": "John"}
						},
						"family_name": {
							"type": "string",
							"description": "Guest last name",
							"x-ui": {"label": "Last name", "placeholder": "Doe"}
						},
						"email": {
							"type": "string",
							"description": "Guest email address",
							"x-ui": {"label": "Email", "placeholder": "guest@example.com"}
						},
						"phone": {
							"type": "string",
							"description": "Guest phone number",
							"x-ui": {"label": "Phone", "placeholder": "+1-555-123-4567"}
						},
						"payment_method_id": {
							"type": "string",
							"description": "Stored payment method ID (resolved server-side)",
							"x-ui": {"label": "Payment method", "help_text": "Stored payment method ID"}
						},
						"special_request": {
							"type": "string",
							"description": "Special requests for the hotel",
							"x-ui": {"label": "Special requests", "widget": "textarea"}
						}
					}
				}`)),
			},
			{
				ActionType:  "expedia.cancel_booking",
				Name:        "Cancel Booking",
				Description: "Cancel a hotel booking — may incur cancellation fees depending on policy",
				RiskLevel:   "high",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["itinerary_id", "room_id"],
					"properties": {
						"itinerary_id": {
							"type": "string",
							"description": "Itinerary ID from the booking",
							"x-ui": {"label": "Itinerary ID", "help_text": "From expedia.create_booking"}
						},
						"room_id": {
							"type": "string",
							"description": "Room ID within the itinerary to cancel",
							"x-ui": {"label": "Room ID", "help_text": "From expedia.get_hotel or expedia.price_check"}
						}
					}
				}`)),
			},
			{
				ActionType:  "expedia.get_booking",
				Name:        "Get Booking",
				Description: "Retrieve booking details and current status",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["itinerary_id", "email"],
					"properties": {
						"itinerary_id": {
							"type": "string",
							"description": "Itinerary ID from the booking",
							"x-ui": {"label": "Itinerary ID", "help_text": "From expedia.create_booking"}
						},
						"email": {
							"type": "string",
							"description": "Email address used for the booking",
							"x-ui": {"label": "Email", "placeholder": "guest@example.com"}
						}
					}
				}`)),
			},
		},
		RequiredCredentials: []connectors.ManifestCredential{
			{Service: "expedia", AuthType: "api_key", InstructionsURL: "https://developers.expediagroup.com/docs/products/rapid/setup/getting-started"},
		},
		Templates: []connectors.ManifestTemplate{
			{
				ID:          "tpl_expedia_search_read_only",
				ActionType:  "expedia.search_hotels",
				Name:        "Search hotels (read-only)",
				Description: "Agent can search hotels for any dates, location, and occupancy. No booking capability.",
				Parameters:  json.RawMessage(`{"checkin":"*","checkout":"*","occupancy":"*","region_id":"*","latitude":"*","longitude":"*","currency":"*","language":"*","sort_by":"*","star_rating":"*","limit":"*"}`),
			},
			{
				ID:          "tpl_expedia_get_hotel",
				ActionType:  "expedia.get_hotel",
				Name:        "View hotel details",
				Description: "Agent can view full details for any hotel property.",
				Parameters:  json.RawMessage(`{"property_id":"*","checkin":"*","checkout":"*","occupancy":"*"}`),
			},
			{
				ID:          "tpl_expedia_price_check",
				ActionType:  "expedia.price_check",
				Name:        "Check room pricing",
				Description: "Agent can confirm pricing and availability for any room.",
				Parameters:  json.RawMessage(`{"room_id":"*"}`),
			},
			{
				ID:          "tpl_expedia_create_booking",
				ActionType:  "expedia.create_booking",
				Name:        "Book hotel rooms",
				Description: "Agent can create hotel bookings. Requires human approval per booking.",
				Parameters:  json.RawMessage(`{"room_id":"*","given_name":"*","family_name":"*","email":"*","phone":"*","payment_method_id":"*","special_request":"*"}`),
			},
			{
				ID:          "tpl_expedia_cancel_booking",
				ActionType:  "expedia.cancel_booking",
				Name:        "Cancel bookings",
				Description: "Agent can cancel hotel bookings. Requires human approval per cancellation.",
				Parameters:  json.RawMessage(`{"itinerary_id":"*","room_id":"*"}`),
			},
			{
				ID:          "tpl_expedia_get_booking",
				ActionType:  "expedia.get_booking",
				Name:        "View booking details",
				Description: "Agent can retrieve booking details and status.",
				Parameters:  json.RawMessage(`{"itinerary_id":"*","email":"*"}`),
			},
		},
	}
}
