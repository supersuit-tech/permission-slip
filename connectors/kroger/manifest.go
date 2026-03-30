package kroger

import (
	_ "embed"
	"encoding/json"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// Manifest returns the connector's metadata manifest. Used by the server to
// auto-seed DB rows on startup.
//go:embed logo.svg
var logoSVG string

func (c *KrogerConnector) Manifest() *connectors.ConnectorManifest {
	return &connectors.ConnectorManifest{
		ID:          "kroger",
		Name:        "Kroger",
		Description: "Kroger grocery integration for product search, store locator, and cart management across ~2,800 stores (Kroger, Ralphs, Fred Meyer, Harris Teeter, and more)",
		LogoSVG:     logoSVG,
		Actions: []connectors.ManifestAction{
			{
				ActionType:  "kroger.search_products",
				Name:        "Search Products",
				Description: "Search for products by keyword with optional location-specific pricing and availability",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["term"],
					"properties": {
						"term": {
							"type": "string",
							"description": "Search term for products (e.g., \"organic milk\", \"sourdough bread\")",
							"x-ui": {"label": "Search term", "placeholder": "organic milk"}
						},
						"location_id": {
							"type": "string",
							"description": "Kroger location ID for store-specific pricing and availability. Get IDs from kroger.search_locations (e.g., \"01400376\")",
							"x-ui": {"label": "Store location", "help_text": "Kroger location ID — use kroger.search_locations to find IDs"}
						},
						"limit": {
							"type": "integer",
							"minimum": 1,
							"maximum": 50,
							"default": 10,
							"description": "Maximum number of results (1-50, default 10)",
							"x-ui": {"label": "Max results"}
						},
						"start": {
							"type": "integer",
							"minimum": 1,
							"description": "Starting index for pagination (use with limit for paging through results)",
							"x-ui": {"hidden": true}
						}
					}
				}`)),
			},
			{
				ActionType:  "kroger.get_product",
				Name:        "Get Product Details",
				Description: "Get detailed product information including nutrition, price, availability, aisle location, and images by UPC",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["product_id"],
					"properties": {
						"product_id": {
							"type": "string",
							"description": "Kroger product ID — a UPC barcode (e.g., \"0001111041700\"). Found in search results as productId",
							"x-ui": {"label": "Product", "help_text": "UPC barcode or Kroger product ID"}
						},
						"location_id": {
							"type": "string",
							"description": "Kroger location ID for store-specific pricing and availability. Get IDs from kroger.search_locations (e.g., \"01400376\")",
							"x-ui": {"label": "Store location", "help_text": "Kroger location ID — use kroger.search_locations to find IDs"}
						}
					}
				}`)),
			},
			{
				ActionType:  "kroger.search_locations",
				Name:        "Search Locations",
				Description: "Find Kroger family stores by zip code or coordinates. Covers all banners: Kroger, Ralphs, Fred Meyer, Harris Teeter, etc.",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"properties": {
						"zip_code": {
							"type": "string",
							"description": "US ZIP code to search near (e.g., \"45202\"). Provide zip_code or lat/lon",
							"x-ui": {"label": "ZIP code", "placeholder": "45202"}
						},
						"lat": {
							"type": "number",
							"minimum": -90,
							"maximum": 90,
							"description": "Latitude for location search (e.g., 39.1031). Use with lon",
							"x-ui": {"label": "Latitude"}
						},
						"lon": {
							"type": "number",
							"minimum": -180,
							"maximum": 180,
							"description": "Longitude for location search (e.g., -84.5120). Use with lat",
							"x-ui": {"label": "Longitude"}
						},
						"radius_miles": {
							"type": "integer",
							"minimum": 1,
							"maximum": 100,
							"default": 10,
							"description": "Search radius in miles (1-100, default 10)",
							"x-ui": {"label": "Radius (miles)"}
						},
						"chain": {
							"type": "string",
							"description": "Filter by store banner (e.g., \"Kroger\", \"Ralphs\", \"Fred Meyer\", \"Harris Teeter\")",
							"x-ui": {"label": "Store chain", "help_text": "e.g. Kroger, Ralphs, Fred Meyer"}
						},
						"limit": {
							"type": "integer",
							"minimum": 1,
							"maximum": 200,
							"default": 10,
							"description": "Maximum number of results (1-200, default 10)",
							"x-ui": {"label": "Max results"}
						}
					}
				}`)),
			},
			{
				ActionType:  "kroger.add_to_cart",
				Name:        "Add to Cart",
				Description: "Add items to the authenticated user's Kroger cart. Requires user OAuth consent with cart.basic:write scope",
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
										"description": "Product UPC code from search results (e.g., \"0001111041700\")",
										"x-ui": {"label": "UPC", "help_text": "Product UPC barcode"}
									},
									"quantity": {
										"type": "integer",
										"minimum": 1,
										"description": "Quantity to add to cart",
										"x-ui": {"label": "Quantity"}
									}
								}
							},
							"description": "Items to add to the cart (max 25 per request)",
							"x-ui": {"label": "Items"}
						},
						"modality": {
							"type": "string",
							"enum": ["PICKUP", "DELIVERY"],
							"description": "Fulfillment method — PICKUP for in-store pickup, DELIVERY for home delivery. Omit to use the user's default",
							"x-ui": {"label": "Fulfillment", "widget": "select"}
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
				OAuthScopes:   []string{"product.compact", "cart.basic:write"},
			},
		},
		Templates: []connectors.ManifestTemplate{
			{
				ID:          "tpl_kroger_search_products",
				ActionType:  "kroger.search_products",
				Name:        "Search for grocery products",
				Description: "Agent can search Kroger's product catalog by keyword.",
				Parameters:  json.RawMessage(`{"term":"*"}`),
			},
			{
				ID:          "tpl_kroger_search_products_at_store",
				ActionType:  "kroger.search_products",
				Name:        "Search products at a specific store",
				Description: "Agent can search products with pricing locked to a specific store location.",
				Parameters:  json.RawMessage(`{"term":"*","location_id":"LOCATION_ID"}`),
			},
			{
				ID:          "tpl_kroger_get_product",
				ActionType:  "kroger.get_product",
				Name:        "View product details",
				Description: "Agent can look up detailed information for any Kroger product by UPC.",
				Parameters:  json.RawMessage(`{"product_id":"*"}`),
			},
			{
				ID:          "tpl_kroger_search_locations",
				ActionType:  "kroger.search_locations",
				Name:        "Find nearby Kroger stores",
				Description: "Agent can search for Kroger store locations by zip code or coordinates.",
				Parameters:  json.RawMessage(`{}`),
			},
			{
				ID:          "tpl_kroger_search_locations_by_zip",
				ActionType:  "kroger.search_locations",
				Name:        "Find stores near a zip code",
				Description: "Agent can find Kroger stores near a specific zip code within a 10-mile radius.",
				Parameters:  json.RawMessage(`{"zip_code":"*","radius_miles":10}`),
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
