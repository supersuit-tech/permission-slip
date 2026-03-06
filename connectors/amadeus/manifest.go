package amadeus

import (
	"encoding/json"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// Manifest returns the connector's metadata manifest. Used by the server to
// auto-seed DB rows on startup.
func (c *AmadeusConnector) Manifest() *connectors.ConnectorManifest {
	return &connectors.ConnectorManifest{
		ID:          "amadeus",
		Name:        "Amadeus",
		Description: "Amadeus travel APIs for flights, hotels, and car rentals",
		Actions: []connectors.ManifestAction{
			{
				ActionType:  "amadeus.search_airports",
				Name:        "Search Airports",
				Description: "Look up airports by name or IATA code",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(`{
					"type": "object",
					"properties": {
						"keyword": {"type": "string", "description": "Airport name or IATA code (e.g. 'San Francisco' or 'SFO')"},
						"subtype": {"type": "string", "enum": ["AIRPORT", "CITY"], "description": "Filter by location type"}
					},
					"required": ["keyword"]
				}`),
			},
			{
				ActionType:  "amadeus.search_flights",
				Name:        "Search Flights",
				Description: "Search flight offers between airports",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(`{
					"type": "object",
					"properties": {
						"origin": {"type": "string", "description": "Origin IATA code (e.g. 'SFO')"},
						"destination": {"type": "string", "description": "Destination IATA code (e.g. 'LAX')"},
						"departure_date": {"type": "string", "format": "date", "description": "Departure date (YYYY-MM-DD)"},
						"return_date": {"type": "string", "format": "date", "description": "Return date for round trip (YYYY-MM-DD)"},
						"adults": {"type": "integer", "minimum": 1, "maximum": 9, "default": 1, "description": "Number of adult travelers (1-9)"},
						"cabin": {"type": "string", "enum": ["ECONOMY", "PREMIUM_ECONOMY", "BUSINESS", "FIRST"], "description": "Cabin class"},
						"nonstop": {"type": "boolean", "default": false, "description": "Only show nonstop flights"},
						"max_results": {"type": "integer", "default": 10, "description": "Maximum number of results"}
					},
					"required": ["origin", "destination", "departure_date"]
				}`),
			},
			{
				ActionType:  "amadeus.price_flight",
				Name:        "Price Flight Offer",
				Description: "Confirm real-time pricing for a specific flight offer before booking",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(`{
					"type": "object",
					"properties": {
						"flight_offer": {"type": "object", "description": "Flight offer object from search results"}
					},
					"required": ["flight_offer"]
				}`),
			},
			{
				ActionType:  "amadeus.book_flight",
				Name:        "Book Flight",
				Description: "Create a flight booking (PNR). High risk — creates a real reservation.",
				RiskLevel:   "high",
				ParametersSchema: json.RawMessage(`{
					"type": "object",
					"properties": {
						"flight_offer": {"type": "object", "description": "Priced flight offer object"},
						"travelers": {
							"type": "array",
							"items": {
								"type": "object",
								"properties": {
									"name": {
										"type": "object",
										"properties": {
											"firstName": {"type": "string", "description": "Traveler's first/given name"},
											"lastName": {"type": "string", "description": "Traveler's last/family name"}
										},
										"required": ["firstName", "lastName"]
									},
									"dateOfBirth": {"type": "string", "format": "date", "description": "YYYY-MM-DD"},
									"gender": {"type": "string", "enum": ["MALE", "FEMALE"]},
									"contact": {
										"type": "object",
										"properties": {
											"email": {"type": "string", "format": "email"},
											"phone": {"type": "string"}
										},
										"required": ["email", "phone"]
									}
								},
								"required": ["name", "dateOfBirth", "gender", "contact"]
							},
							"description": "Array of traveler details"
						},
						"payment_method_id": {"type": "string", "description": "Stored payment method ID"},
						"remarks": {"type": "string", "description": "Optional booking remarks"}
					},
					"required": ["flight_offer", "travelers", "payment_method_id"]
				}`),
			},
			{
				ActionType:  "amadeus.search_hotels",
				Name:        "Search Hotels",
				Description: "Search hotel offers by city or coordinates",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(`{
					"type": "object",
					"properties": {
						"city_code": {"type": "string", "description": "City IATA code (e.g. 'PAR')"},
						"latitude": {"type": "string", "description": "Latitude for geo search"},
						"longitude": {"type": "string", "description": "Longitude for geo search"},
						"check_in_date": {"type": "string", "format": "date", "description": "Check-in date (YYYY-MM-DD)"},
						"check_out_date": {"type": "string", "format": "date", "description": "Check-out date (YYYY-MM-DD)"},
						"adults": {"type": "integer", "default": 1, "description": "Number of adults"},
						"room_quantity": {"type": "integer", "default": 1, "description": "Number of rooms"},
						"ratings": {"type": "array", "items": {"type": "integer", "minimum": 1, "maximum": 5}, "description": "Hotel star ratings to filter by"},
						"price_range": {"type": "string", "description": "Price range (e.g. '100-300')"},
						"currency": {"type": "string", "description": "Currency code (e.g. 'USD')"}
					},
					"required": ["check_in_date", "check_out_date"]
				}`),
			},
			{
				ActionType:  "amadeus.search_cars",
				Name:        "Search Car Rentals",
				Description: "Search available rental cars at a location",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(`{
					"type": "object",
					"properties": {
						"pickup_location": {"type": "string", "description": "Pickup IATA code"},
						"pickup_date": {"type": "string", "description": "Pickup date/time"},
						"dropoff_date": {"type": "string", "description": "Dropoff date/time"},
						"dropoff_location": {"type": "string", "description": "Dropoff IATA code (defaults to pickup location)"},
						"provider": {"type": "string", "description": "Transfer type/provider filter"}
					},
					"required": ["pickup_location", "pickup_date", "dropoff_date"]
				}`),
			},
		},
		RequiredCredentials: []connectors.ManifestCredential{
			{
				Service:         "amadeus",
				AuthType:        "api_key",
				InstructionsURL: "https://developers.amadeus.com/get-started/get-started-with-self-service-apis-335",
			},
		},
		Templates: []connectors.ManifestTemplate{},
	}
}

// Actions returns the registered action handlers keyed by action_type.
func (c *AmadeusConnector) Actions() map[string]connectors.Action {
	return map[string]connectors.Action{
		"amadeus.search_airports": &searchAirportsAction{conn: c},
		"amadeus.search_flights":  &searchFlightsAction{conn: c},
		"amadeus.price_flight":    &priceFlightAction{conn: c},
		"amadeus.book_flight":     &bookFlightAction{conn: c},
		"amadeus.search_hotels":   &searchHotelsAction{conn: c},
		"amadeus.search_cars":     &searchCarsAction{conn: c},
	}
}
