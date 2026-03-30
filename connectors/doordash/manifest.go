package doordash

import (
	_ "embed"
	"encoding/json"
	"strings"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// statusEnumJSON builds the JSON enum array from deliveryStatuses so the
// manifest schema and Go validation always stay in sync.
func statusEnumJSON() string {
	quoted := make([]string, len(deliveryStatuses))
	for i, s := range deliveryStatuses {
		quoted[i] = `"` + s + `"`
	}
	return "[" + strings.Join(quoted, ", ") + "]"
}

//go:embed logo.svg
var logoSVG string

// Manifest returns the connector's metadata manifest for DB auto-seeding.
func (c *DoorDashConnector) Manifest() *connectors.ConnectorManifest {
	return &connectors.ConnectorManifest{
		ID:          "doordash",
		Name:        "DoorDash Drive",
		Description: "DoorDash Drive delivery-as-a-service integration for creating deliveries, getting quotes, and tracking delivery status",
		LogoSVG:     logoSVG,
		Actions: []connectors.ManifestAction{
			getQuoteManifest(),
			createDeliveryManifest(),
			getDeliveryManifest(),
			cancelDeliveryManifest(),
			listDeliveriesManifest(),
		},
		RequiredCredentials: []connectors.ManifestCredential{
			{
				Service:         "doordash",
				AuthType:        "api_key",
				InstructionsURL: "https://developer.doordash.com/en-US/docs/drive/getting-started/",
			},
		},
		Templates: []connectors.ManifestTemplate{
			{
				ID:          "tpl_doordash_get_quote",
				ActionType:  "doordash.get_quote",
				Name:        "Get delivery quotes (read-only)",
				Description: "Agent can get delivery fee estimates and ETAs without creating actual deliveries.",
				Parameters:  json.RawMessage(`{"pickup_address":"*","dropoff_address":"*","pickup_phone":"*","dropoff_phone":"*","order_value":"*"}`),
			},
			{
				ID:          "tpl_doordash_create_delivery",
				ActionType:  "doordash.create_delivery",
				Name:        "Create deliveries (requires approval)",
				Description: "Agent can create delivery requests. WARNING: dispatches a real Dasher and charges money. Requires human approval per delivery.",
				Parameters:  json.RawMessage(`{"pickup_address":"*","pickup_phone":"*","pickup_business_name":"*","pickup_instructions":"*","dropoff_address":"*","dropoff_phone":"*","dropoff_contact_given_name":"*","dropoff_instructions":"*","order_value":"*","items":"*"}`),
			},
			{
				ID:          "tpl_doordash_get_delivery",
				ActionType:  "doordash.get_delivery",
				Name:        "Track deliveries (read-only)",
				Description: "Agent can check the status of existing deliveries.",
				Parameters:  json.RawMessage(`{"delivery_id":"*"}`),
			},
			{
				ID:          "tpl_doordash_cancel_delivery",
				ActionType:  "doordash.cancel_delivery",
				Name:        "Cancel deliveries",
				Description: "Agent can cancel active deliveries. May incur cancellation fees depending on delivery status.",
				Parameters:  json.RawMessage(`{"delivery_id":"*"}`),
			},
			{
				ID:          "tpl_doordash_list_deliveries",
				ActionType:  "doordash.list_deliveries",
				Name:        "List deliveries (read-only)",
				Description: "Agent can list and filter recent deliveries.",
				Parameters:  json.RawMessage(`{"limit":"*","starting_after":"*","status":"*"}`),
			},
		},
	}
}

func getQuoteManifest() connectors.ManifestAction {
	return connectors.ManifestAction{
		ActionType:  "doordash.get_quote",
		Name:        "Get Delivery Quote",
		Description: "Get a delivery fee estimate and ETA before creating a delivery. Agents should always quote first so the user can approve the cost. Returns the estimated fee in cents, currency, and estimated delivery time.",
		RiskLevel:   "low",
		ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
			"type": "object",
			"required": ["pickup_address", "dropoff_address", "pickup_phone", "dropoff_phone"],
			"additionalProperties": false,
			"properties": {
				"pickup_address": {
					"type": "string",
					"description": "Full street address for pickup (e.g. \"901 Market St, San Francisco, CA 94103\")",
					"x-ui": {"label": "Pickup address", "placeholder": "901 Market St, San Francisco, CA 94103"}
				},
				"dropoff_address": {
					"type": "string",
					"description": "Full street address for dropoff (e.g. \"123 Main St, San Francisco, CA 94105\")",
					"x-ui": {"label": "Dropoff address", "placeholder": "123 Main St, San Francisco, CA 94105"}
				},
				"pickup_phone": {
					"type": "string",
					"description": "Phone number for pickup contact (e.g. \"+15551234567\")",
					"x-ui": {"label": "Pickup phone", "placeholder": "+15551234567"}
				},
				"dropoff_phone": {
					"type": "string",
					"description": "Phone number for dropoff contact (e.g. \"+15559876543\")",
					"x-ui": {"label": "Dropoff phone", "placeholder": "+15559876543"}
				},
				"order_value": {
					"type": "integer",
					"minimum": 0,
					"description": "Total value of items being delivered in cents (e.g. 2500 = $25.00). Affects delivery fee calculation.",
					"x-ui": {"label": "Order value", "help_text": "In cents (e.g. 1999 = $19.99)"}
				}
			}
		}`)),
	}
}

func createDeliveryManifest() connectors.ManifestAction {
	return connectors.ManifestAction{
		ActionType:            "doordash.create_delivery",
		Name:                  "Create Delivery",
		Description:           "Create a delivery request that dispatches a Dasher. WARNING: This is a high-risk action — it dispatches a real courier and charges money. Always use get_quote first so the user can approve the cost before creating the delivery.",
		RiskLevel:             "high",
		RequiresPaymentMethod: true,
		ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
			"type": "object",
			"required": ["pickup_address", "pickup_phone", "dropoff_address", "dropoff_phone", "dropoff_contact_given_name"],
			"additionalProperties": false,
			"properties": {
				"pickup_address": {
					"type": "string",
					"description": "Full street address for pickup (e.g. \"901 Market St, San Francisco, CA 94103\")",
					"x-ui": {"label": "Pickup address", "placeholder": "901 Market St, San Francisco, CA 94103"}
				},
				"pickup_phone": {
					"type": "string",
					"description": "Phone number for pickup contact (e.g. \"+15551234567\")",
					"x-ui": {"label": "Pickup phone", "placeholder": "+15551234567"}
				},
				"pickup_business_name": {
					"type": "string",
					"description": "Business name at the pickup location",
					"x-ui": {"label": "Pickup business name", "placeholder": "Joe's Pizza"}
				},
				"pickup_instructions": {
					"type": "string",
					"description": "Instructions for the Dasher at pickup (e.g. \"Ring doorbell, ask for John\")",
					"x-ui": {"label": "Pickup instructions", "placeholder": "Ring doorbell, ask for John", "widget": "textarea"}
				},
				"dropoff_address": {
					"type": "string",
					"description": "Full street address for dropoff (e.g. \"123 Main St, San Francisco, CA 94105\")",
					"x-ui": {"label": "Dropoff address", "placeholder": "123 Main St, San Francisco, CA 94105"}
				},
				"dropoff_phone": {
					"type": "string",
					"description": "Phone number for dropoff contact (e.g. \"+15559876543\")",
					"x-ui": {"label": "Dropoff phone", "placeholder": "+15559876543"}
				},
				"dropoff_contact_given_name": {
					"type": "string",
					"description": "First name of the person receiving the delivery",
					"x-ui": {"label": "Recipient first name", "placeholder": "John"}
				},
				"dropoff_instructions": {
					"type": "string",
					"description": "Instructions for the Dasher at dropoff (e.g. \"Leave at front door\")",
					"x-ui": {"label": "Dropoff instructions", "placeholder": "Leave at front door", "widget": "textarea"}
				},
				"order_value": {
					"type": "integer",
					"minimum": 0,
					"description": "Total value of items being delivered in cents (e.g. 2500 = $25.00)",
					"x-ui": {"label": "Order value", "help_text": "In cents (e.g. 1999 = $19.99)"}
				},
				"items": {
					"type": "array",
					"description": "List of items being delivered",
					"x-ui": {"label": "Items"},
					"items": {
						"type": "object",
						"required": ["name", "quantity"],
						"additionalProperties": false,
						"properties": {
							"name": {
								"type": "string",
								"description": "Item name (e.g. \"Documents\", \"Package\")",
								"x-ui": {"label": "Item name", "placeholder": "Documents"}
							},
							"quantity": {
								"type": "integer",
								"minimum": 1,
								"description": "Number of this item",
								"x-ui": {"label": "Quantity"}
							},
							"description": {
								"type": "string",
								"description": "Additional item description",
								"x-ui": {"label": "Description", "placeholder": "Fragile, handle with care"}
							}
						}
					}
				}
			}
		}`)),
	}
}

func getDeliveryManifest() connectors.ManifestAction {
	return connectors.ManifestAction{
		ActionType:  "doordash.get_delivery",
		Name:        "Get Delivery Status",
		Description: "Check the current status of a delivery. Returns the full delivery object including status (created, confirmed, enroute_to_pickup, picked_up, enroute_to_dropoff, delivered, cancelled), Dasher info, and timestamps.",
		RiskLevel:   "low",
		ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
			"type": "object",
			"required": ["delivery_id"],
			"additionalProperties": false,
			"properties": {
				"delivery_id": {
					"type": "string",
					"description": "The external_delivery_id of the delivery to check",
					"x-ui": {"label": "Delivery ID", "help_text": "DoorDash delivery UUID"}
				}
			}
		}`)),
	}
}

func cancelDeliveryManifest() connectors.ManifestAction {
	return connectors.ManifestAction{
		ActionType:  "doordash.cancel_delivery",
		Name:        "Cancel Delivery",
		Description: "Cancel an active delivery. May incur a cancellation fee depending on the delivery's current status (e.g., if a Dasher has already been assigned or picked up the items).",
		RiskLevel:   "medium",
		ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
			"type": "object",
			"required": ["delivery_id"],
			"additionalProperties": false,
			"properties": {
				"delivery_id": {
					"type": "string",
					"description": "The external_delivery_id of the delivery to cancel",
					"x-ui": {"label": "Delivery ID", "help_text": "DoorDash delivery UUID"}
				}
			}
		}`)),
	}
}

func listDeliveriesManifest() connectors.ManifestAction {
	// Build the status enum from the shared deliveryStatuses slice so
	// validation and schema can never drift apart.
	schema := `{
		"type": "object",
		"additionalProperties": false,
		"properties": {
			"limit": {
				"type": "integer",
				"minimum": 1,
				"description": "Maximum number of deliveries to return (default 20)",
				"x-ui": {"label": "Max results"}
			},
			"starting_after": {
				"type": "string",
				"description": "Pagination cursor from a previous list_deliveries response",
				"x-ui": {"label": "Starting after", "hidden": true, "help_text": "Pagination cursor from a previous response"}
			},
			"status": {
				"type": "string",
				"enum": ` + statusEnumJSON() + `,
				"description": "Filter by delivery status",
				"x-ui": {"label": "Status", "widget": "select"}
			}
		}
	}`

	return connectors.ManifestAction{
		ActionType:       "doordash.list_deliveries",
		Name:             "List Deliveries",
		Description:      "List recent deliveries with optional status filter. Supports cursor-based pagination for large result sets.",
		RiskLevel:        "low",
		ParametersSchema: json.RawMessage(connectors.TrimIndent(schema)),
	}
}
