package walmart

import (
	_ "embed"
	"encoding/json"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// Manifest returns the connector's metadata manifest. Used by the server to
// auto-seed DB rows on startup.
//go:embed logo.svg
var logoSVG string

func (c *WalmartConnector) Manifest() *connectors.ConnectorManifest {
	return &connectors.ConnectorManifest{
		ID:          "walmart",
		Name:        "Walmart",
		Description: "Walmart Affiliate API for product search, details, taxonomy, and trending products with shoppable cart links. All actions are read-only.",
		LogoSVG:     logoSVG,
		Actions: []connectors.ManifestAction{
			{
				ActionType:  "walmart.search_products",
				Name:        "Search Products",
				Description: "Search the Walmart product catalog by keyword with optional category filtering and sorting",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"properties": {
						"query": {
							"type": "string",
							"description": "Search keyword (e.g. 'paper towels', 'laptop', 'organic milk')"
						},
						"category_id": {
							"type": "string",
							"description": "Category ID to filter results (e.g. \"3944\" for Grocery — from walmart.get_taxonomy)"
						},
						"sort": {
							"type": "string",
							"enum": ["relevance", "price", "title", "bestseller", "customerRating", "new"],
							"description": "Sort field for results (defaults to relevance)"
						},
						"order": {
							"type": "string",
							"enum": ["asc", "desc"],
							"description": "Sort order: ascending or descending (defaults to relevance order)"
						},
						"start": {
							"type": "integer",
							"minimum": 0,
							"default": 0,
							"description": "Starting index for pagination",
							"x-ui": {"hidden": true}
						},
						"limit": {
							"type": "integer",
							"minimum": 1,
							"maximum": 25,
							"default": 10,
							"description": "Maximum number of products to return"
						}
					},
					"required": ["query"],
					"additionalProperties": false
				}`)),
			},
			{
				ActionType:  "walmart.get_product",
				Name:        "Get Product",
				Description: "Get detailed product information by item ID including price, availability, images, reviews, ratings, and add-to-cart URL",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"properties": {
						"item_id": {
							"type": "string",
							"pattern": "^\\d+$",
							"description": "The Walmart item ID (numeric, e.g. \"12345678\")"
						}
					},
					"required": ["item_id"],
					"additionalProperties": false
				}`)),
			},
			{
				ActionType:  "walmart.get_taxonomy",
				Name:        "Get Taxonomy",
				Description: "Browse the Walmart product category taxonomy to discover category IDs for filtered searches",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"properties": {},
					"additionalProperties": false
				}`)),
			},
			{
				ActionType:  "walmart.get_trending",
				Name:        "Get Trending Products",
				Description: "Discover currently trending products on Walmart.com",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"properties": {},
					"additionalProperties": false
				}`)),
			},
		},
		RequiredCredentials: []connectors.ManifestCredential{
			{
				Service:         "walmart",
				AuthType:        "custom",
				InstructionsURL: "https://walmart.io/docs/affiliate/onboarding-guide",
			},
		},
		Templates: []connectors.ManifestTemplate{
			{
				ID:          "walmart-search-products",
				ActionType:  "walmart.search_products",
				Name:        "Search products",
				Description: "Agent can search the Walmart catalog by keyword. Read-only, no purchase capability.",
				Parameters:  json.RawMessage(`{"query":"*","category_id":"*","sort":"*","order":"*","start":"*","limit":"*"}`),
			},
			{
				ID:          "walmart-search-best-deals",
				ActionType:  "walmart.search_products",
				Name:        "Search for best deals (price ascending)",
				Description: "Search products sorted by price from lowest to highest",
				Parameters:  json.RawMessage(`{"query":"*","sort":"price","order":"asc","limit":10}`),
			},
			{
				ID:          "walmart-search-top-rated",
				ActionType:  "walmart.search_products",
				Name:        "Search top-rated products",
				Description: "Search products sorted by customer rating",
				Parameters:  json.RawMessage(`{"query":"*","sort":"customerRating","order":"desc","limit":10}`),
			},
			{
				ID:          "walmart-get-product-details",
				ActionType:  "walmart.get_product",
				Name:        "Get product details",
				Description: "Look up a specific product by ID for pricing, reviews, and purchase link",
				Parameters:  json.RawMessage(`{"item_id":"*"}`),
			},
			{
				ID:          "walmart-browse-categories",
				ActionType:  "walmart.get_taxonomy",
				Name:        "Browse product categories",
				Description: "View the product category tree to find category IDs for filtered searches",
				Parameters:  json.RawMessage(`{}`),
			},
			{
				ID:          "walmart-trending-products",
				ActionType:  "walmart.get_trending",
				Name:        "View trending products",
				Description: "See what products are currently trending on Walmart",
				Parameters:  json.RawMessage(`{}`),
			},
		},
	}
}
