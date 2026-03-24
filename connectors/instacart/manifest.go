package instacart

import (
	_ "embed"
	"encoding/json"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

//go:embed logo.svg
var logoSVG string

// Manifest returns the connector's metadata manifest. Used by the server to
// auto-seed DB rows on startup.
func (c *InstacartConnector) Manifest() *connectors.ConnectorManifest {
	return &connectors.ConnectorManifest{
		ID:          "instacart",
		Name:        "Instacart",
		Description: "Instacart Developer Platform for shoppable recipe and shopping-list landing pages across 1,400+ retail banners. Users open the generated link, pick a store, and check out on Instacart.",
		LogoSVG:     logoSVG,
		Actions: []connectors.ManifestAction{
			{
				ActionType:      "instacart.create_products_link",
				Name:            "Create Products Link",
				Description:     "Create an Instacart-hosted landing page from product line items and return a shareable products_link_url. Requires an approved Instacart Developer Platform API key.",
				RiskLevel:       "low",
				DisplayTemplate: "Create Instacart link — {{line_items:count}} items",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["line_items"],
					"properties": {
						"title": {
							"type": "string",
							"description": "Title shown on the shopping list or recipe page"
						},
						"image_url": {
							"type": "string",
							"description": "Image URL for the page (Instacart recommends 500x500 px)"
						},
						"link_type": {
							"type": "string",
							"enum": ["shopping_list", "recipe"],
							"description": "Page type: shopping_list (default) or recipe"
						},
						"expires_in": {
							"type": "integer",
							"minimum": 1,
							"maximum": 365,
							"description": "Days until the link expires (max 365). Recipe links default to 30 days on Instacart's side when omitted."
						},
						"instructions": {
							"type": "array",
							"maxItems": 50,
							"items": {"type": "string", "maxLength": 2000},
							"description": "Extra context (e.g. recipe steps or dietary notes)"
						},
						"line_items": {
							"type": "array",
							"minItems": 1,
							"maxItems": 200,
							"items": {
								"type": "object",
								"required": ["name"],
								"additionalProperties": true,
								"properties": {
									"name": {
										"type": "string",
										"maxLength": 2048,
										"description": "Product or ingredient text for Instacart to match (e.g. \"2 lb chicken breast\")"
									},
									"display_text": {
										"type": "string",
										"description": "Display title for the matched item"
									},
									"quantity": {
										"type": "number",
										"description": "Deprecated by Instacart; prefer line_item_measurements"
									},
									"unit": {
										"type": "string",
										"description": "Deprecated by Instacart; prefer line_item_measurements"
									},
									"line_item_measurements": {
										"type": "array",
										"items": {
											"type": "object",
											"properties": {
												"quantity": {"type": "number"},
												"unit": {"type": "string"}
											},
											"additionalProperties": false
										}
									},
									"upcs": {
										"type": "array",
										"items": {"type": "string"}
									},
									"product_ids": {
										"type": "array",
										"items": {"type": "integer"}
									}
								}
							},
							"description": "Items to include. Each element is a LineItem object with at least name, or use line_item_measurements for quantities (see Instacart docs). You may use the parameter name items instead of line_items. For convenience, each element may be a plain string (same as {\"name\": \"...\"})."
						},
						"landing_page_configuration": {
							"type": "object",
							"additionalProperties": true,
							"description": "Optional landing page options (e.g. partner_linkback_url, enable_pantry_items for recipe links)"
						}
					},
					"additionalProperties": false
				}`)),
			},
		},
		RequiredCredentials: []connectors.ManifestCredential{
			{
				Service:         "instacart",
				AuthType:        "api_key",
				InstructionsURL: "https://www.instacart.com/developer",
			},
		},
		Templates: []connectors.ManifestTemplate{
			{
				ID:          "tpl_instacart_shopping_list",
				ActionType:  "instacart.create_products_link",
				Name:        "Create shoppable shopping list link",
				Description: "Turn a list of product names into an Instacart landing page URL.",
				Parameters:  json.RawMessage(`{"title":"*","line_items":[{"name":"*"}]}`),
			},
			{
				ID:          "tpl_instacart_recipe_link",
				ActionType:  "instacart.create_products_link",
				Name:        "Create shoppable recipe link",
				Description: "Recipe-style Instacart page with ingredient line items (link_type recipe).",
				Parameters:  json.RawMessage(`{"title":"*","link_type":"recipe","line_items":[{"name":"*"}]}`),
			},
		},
	}
}
