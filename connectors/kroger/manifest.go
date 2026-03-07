package kroger

import (
	"encoding/json"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// Manifest returns the connector's metadata manifest. Used by the server to
// auto-seed DB rows on startup.
func (c *KrogerConnector) Manifest() *connectors.ConnectorManifest {
	return &connectors.ConnectorManifest{
		ID:          "kroger",
		Name:        "Kroger",
		Description: "Kroger grocery integration for product search, store locator, and cart management",
		Actions: []connectors.ManifestAction{
			{
				ActionType:  "kroger.search_products",
				Name:        "Search Products",
				Description: "Search for products by keyword with optional location-specific pricing",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["term"],
					"properties": {
						"term": {
							"type": "string",
							"description": "Search term for products"
						},
						"location_id": {
							"type": "string",
							"description": "Kroger location ID for location-specific pricing and availability"
						},
						"limit": {
							"type": "integer",
							"minimum": 1,
							"maximum": 50,
							"default": 10,
							"description": "Maximum number of results (1-50, default 10)"
						},
						"start": {
							"type": "integer",
							"minimum": 1,
							"description": "Starting index for pagination"
						}
					}
				}`)),
			},
			{
				ActionType:  "kroger.get_product",
				Name:        "Get Product Details",
				Description: "Get detailed product information including nutrition, price, availability, and images",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["product_id"],
					"properties": {
						"product_id": {
							"type": "string",
							"description": "Kroger product ID (UPC)"
						},
						"location_id": {
							"type": "string",
							"description": "Kroger location ID for location-specific pricing and availability"
						}
					}
				}`)),
			},
			{
				ActionType:  "kroger.search_locations",
				Name:        "Search Locations",
				Description: "Find Kroger stores by zip code, coordinates, or chain",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"properties": {
						"zip_code": {
							"type": "string",
							"description": "ZIP code to search near"
						},
						"lat": {
							"type": "number",
							"description": "Latitude for location search"
						},
						"lon": {
							"type": "number",
							"description": "Longitude for location search"
						},
						"radius_miles": {
							"type": "integer",
							"minimum": 1,
							"maximum": 100,
							"default": 10,
							"description": "Search radius in miles (1-100, default 10)"
						},
						"chain": {
							"type": "string",
							"description": "Filter by chain name (e.g., Kroger, Ralphs, Fred Meyer)"
						},
						"limit": {
							"type": "integer",
							"minimum": 1,
							"maximum": 200,
							"default": 10,
							"description": "Maximum number of results (1-200, default 10)"
						}
					}
				}`)),
			},
			{
				ActionType:  "kroger.add_to_cart",
				Name:        "Add to Cart",
				Description: "Add items to the authenticated user's Kroger cart (requires user OAuth consent)",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["items"],
					"properties": {
						"items": {
							"type": "array",
							"minItems": 1,
							"maxItems": 25,
							"items": {
								"type": "object",
								"required": ["upc", "quantity"],
								"properties": {
									"upc": {
										"type": "string",
										"description": "Product UPC code"
									},
									"quantity": {
										"type": "integer",
										"minimum": 1,
										"description": "Quantity to add"
									}
								}
							},
							"description": "Items to add to the cart"
						}
					}
				}`)),
			},
		},
		RequiredCredentials: []connectors.ManifestCredential{
			{
				Service:       "kroger",
				AuthType:      "oauth2",
				OAuthProvider: "kroger",
				OAuthScopes:   []string{"product.compact"},
			},
		},
		OAuthProviders: []connectors.ManifestOAuthProvider{
			{
				ID:           "kroger",
				AuthorizeURL: "https://api.kroger.com/v1/connect/oauth2/authorize",
				TokenURL:     "https://api.kroger.com/v1/connect/oauth2/token",
				Scopes:       []string{"product.compact", "cart.basic:write"},
			},
		},
		Templates: []connectors.ManifestTemplate{
			{
				ID:          "tpl_kroger_search_products",
				ActionType:  "kroger.search_products",
				Name:        "Search for grocery products",
				Description: "Agent can search Kroger's product catalog.",
				Parameters:  json.RawMessage(`{"term":"*"}`),
			},
			{
				ID:          "tpl_kroger_get_product",
				ActionType:  "kroger.get_product",
				Name:        "View product details",
				Description: "Agent can view details for any Kroger product.",
				Parameters:  json.RawMessage(`{"product_id":"*"}`),
			},
			{
				ID:          "tpl_kroger_search_locations",
				ActionType:  "kroger.search_locations",
				Name:        "Find nearby Kroger stores",
				Description: "Agent can search for Kroger store locations.",
				Parameters:  json.RawMessage(`{}`),
			},
			{
				ID:          "tpl_kroger_add_to_cart",
				ActionType:  "kroger.add_to_cart",
				Name:        "Add items to my Kroger cart",
				Description: "Agent can add products to the user's Kroger shopping cart.",
				Parameters:  json.RawMessage(`{"items":"*"}`),
			},
		},
	}
}
