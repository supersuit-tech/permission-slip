package cloudflare

import (
	_ "embed"
	"encoding/json"

	"github.com/supersuit-tech/permission-slip/connectors"
)

//go:embed logo.svg
var logoSVG string

func (c *CloudflareConnector) Manifest() *connectors.ConnectorManifest {
	return &connectors.ConnectorManifest{
		ID:          "cloudflare",
		Name:        "Cloudflare",
		Description: "Cloudflare integration for DNS management, domain settings, tunnels, and cache purging",
		Status:      "early_preview",
		LogoSVG:     logoSVG,
		Actions: []connectors.ManifestAction{
			{
				ActionType:  "cloudflare.list_zones",
				Name:        "List Zones",
				Description: "List all zones (domains) in the Cloudflare account",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"properties": {
						"name": {
							"type": "string",
							"description": "Filter by domain name (e.g. example.com)"
						},
						"status": {
							"type": "string",
							"enum": ["active", "pending", "initializing", "moved", "deleted", "deactivated"],
							"description": "Filter by zone status"
						},
						"page": {
							"type": "integer",
							"description": "Page number for pagination"
						}
					}
				}`)),
			},
			{
				ActionType:  "cloudflare.get_zone",
				Name:        "Get Zone Details",
				Description: "Get detailed information about a specific zone",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["zone_id"],
					"properties": {
						"zone_id": {
							"type": "string",
							"description": "Zone ID to retrieve"
						}
					}
				}`)),
			},
			{
				ActionType:  "cloudflare.list_dns_records",
				Name:        "List DNS Records",
				Description: "List DNS records for a zone",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["zone_id"],
					"properties": {
						"zone_id": {
							"type": "string",
							"description": "Zone ID to list records for"
						},
						"type": {
							"type": "string",
							"enum": ["A", "AAAA", "CNAME", "MX", "TXT", "NS", "SRV", "CAA", "PTR"],
							"description": "Filter by record type"
						},
						"name": {
							"type": "string",
							"description": "Filter by record name"
						},
						"page": {
							"type": "integer",
							"description": "Page number for pagination"
						}
					}
				}`)),
			},
			{
				ActionType:  "cloudflare.create_dns_record",
				Name:        "Create DNS Record",
				Description: "Create a new DNS record in a zone",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["zone_id", "type", "name", "content"],
					"properties": {
						"zone_id": {
							"type": "string",
							"description": "Zone ID to create the record in"
						},
						"type": {
							"type": "string",
							"enum": ["A", "AAAA", "CNAME", "MX", "TXT", "NS", "SRV", "CAA", "PTR"],
							"description": "DNS record type"
						},
						"name": {
							"type": "string",
							"description": "DNS record name (e.g. subdomain.example.com or @ for root)"
						},
						"content": {
							"type": "string",
							"description": "DNS record content (e.g. IP address, hostname, or text value)"
						},
						"ttl": {
							"type": "integer",
							"description": "Time to live in seconds (1 = automatic)"
						},
						"proxied": {
							"type": "boolean",
							"description": "Whether the record is proxied through Cloudflare"
						}
					}
				}`)),
			},
			{
				ActionType:  "cloudflare.update_dns_record",
				Name:        "Update DNS Record",
				Description: "Update an existing DNS record",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["zone_id", "record_id"],
					"properties": {
						"zone_id": {
							"type": "string",
							"description": "Zone ID the record belongs to"
						},
						"record_id": {
							"type": "string",
							"description": "DNS record ID to update"
						},
						"type": {
							"type": "string",
							"enum": ["A", "AAAA", "CNAME", "MX", "TXT", "NS", "SRV", "CAA", "PTR"],
							"description": "Updated record type"
						},
						"name": {
							"type": "string",
							"description": "Updated record name"
						},
						"content": {
							"type": "string",
							"description": "Updated record content"
						},
						"ttl": {
							"type": "integer",
							"description": "Updated TTL in seconds (1 = automatic)"
						},
						"proxied": {
							"type": "boolean",
							"description": "Whether the record is proxied through Cloudflare"
						}
					}
				}`)),
			},
			{
				ActionType:  "cloudflare.delete_dns_record",
				Name:        "Delete DNS Record",
				Description: "Delete a DNS record from a zone",
				RiskLevel:   "high",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["zone_id", "record_id"],
					"properties": {
						"zone_id": {
							"type": "string",
							"description": "Zone ID the record belongs to"
						},
						"record_id": {
							"type": "string",
							"description": "DNS record ID to delete"
						}
					}
				}`)),
			},
			{
				ActionType:  "cloudflare.list_tunnels",
				Name:        "List Tunnels",
				Description: "List Cloudflare Tunnels in an account",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["account_id"],
					"properties": {
						"account_id": {
							"type": "string",
							"description": "Cloudflare account ID"
						},
						"name": {
							"type": "string",
							"description": "Filter by tunnel name"
						},
						"is_deleted": {
							"type": "boolean",
							"description": "Include deleted tunnels"
						}
					}
				}`)),
			},
			{
				ActionType:  "cloudflare.create_tunnel",
				Name:        "Create Tunnel",
				Description: "Create a new Cloudflare Tunnel. A tunnel secret is auto-generated if not provided.",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["account_id", "name"],
					"properties": {
						"account_id": {
							"type": "string",
							"description": "Cloudflare account ID"
						},
						"name": {
							"type": "string",
							"description": "Name for the tunnel"
						},
						"tunnel_secret": {
							"type": "string",
							"description": "Optional base64-encoded 32-byte secret. Auto-generated if omitted. WARNING: sensitive value — avoid passing through approval systems when possible."
						}
					}
				}`)),
			},
			{
				ActionType:  "cloudflare.delete_tunnel",
				Name:        "Delete Tunnel",
				Description: "Delete a Cloudflare Tunnel",
				RiskLevel:   "high",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["account_id", "tunnel_id"],
					"properties": {
						"account_id": {
							"type": "string",
							"description": "Cloudflare account ID"
						},
						"tunnel_id": {
							"type": "string",
							"description": "Tunnel ID to delete"
						}
					}
				}`)),
			},
			{
				ActionType:  "cloudflare.get_tunnel",
				Name:        "Get Tunnel Details",
				Description: "Get detailed information about a specific tunnel",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["account_id", "tunnel_id"],
					"properties": {
						"account_id": {
							"type": "string",
							"description": "Cloudflare account ID"
						},
						"tunnel_id": {
							"type": "string",
							"description": "Tunnel ID to retrieve"
						}
					}
				}`)),
			},
			{
				ActionType:  "cloudflare.list_tunnel_configs",
				Name:        "Get Tunnel Configuration",
				Description: "Get the configuration for a Cloudflare Tunnel",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["account_id", "tunnel_id"],
					"properties": {
						"account_id": {
							"type": "string",
							"description": "Cloudflare account ID"
						},
						"tunnel_id": {
							"type": "string",
							"description": "Tunnel ID to get config for"
						}
					}
				}`)),
			},
			{
				ActionType:  "cloudflare.update_tunnel_config",
				Name:        "Update Tunnel Configuration",
				Description: "Update the configuration for a Cloudflare Tunnel (ingress rules, etc.)",
				RiskLevel:   "high",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["account_id", "tunnel_id", "config"],
					"properties": {
						"account_id": {
							"type": "string",
							"description": "Cloudflare account ID"
						},
						"tunnel_id": {
							"type": "string",
							"description": "Tunnel ID to configure"
						},
						"config": {
							"type": "object",
							"description": "Tunnel configuration object with ingress rules"
						}
					}
				}`)),
			},
			{
				ActionType:  "cloudflare.check_domain",
				Name:        "Check Domain Registration",
				Description: "Check domain registration status and availability via Cloudflare Registrar",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["account_id", "domain"],
					"properties": {
						"account_id": {
							"type": "string",
							"description": "Cloudflare account ID"
						},
						"domain": {
							"type": "string",
							"description": "Domain name to check (e.g. example.com)"
						}
					}
				}`)),
			},
			{
				ActionType:  "cloudflare.update_domain_settings",
				Name:        "Update Domain Settings",
				Description: "Update settings (e.g. auto-renewal) for a domain already registered in Cloudflare Registrar",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["account_id", "domain"],
					"properties": {
						"account_id": {
							"type": "string",
							"description": "Cloudflare account ID"
						},
						"domain": {
							"type": "string",
							"description": "Domain name already registered in your account (e.g. example.com)"
						},
						"auto_renew": {
							"type": "boolean",
							"description": "Enable automatic renewal"
						}
					}
				}`)),
			},
			{
				ActionType:  "cloudflare.purge_cache",
				Name:        "Purge Cache",
				Description: "Purge cached content for a zone (all files or specific URLs/tags/hosts)",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["zone_id"],
					"properties": {
						"zone_id": {
							"type": "string",
							"description": "Zone ID to purge cache for"
						},
						"purge_everything": {
							"type": "boolean",
							"description": "Purge all cached files (use with caution)"
						},
						"files": {
							"type": "array",
							"items": {"type": "string"},
							"description": "List of specific URLs to purge"
						},
						"tags": {
							"type": "array",
							"items": {"type": "string"},
							"description": "List of cache tags to purge (Enterprise only)"
						},
						"hosts": {
							"type": "array",
							"items": {"type": "string"},
							"description": "List of hostnames to purge (Enterprise only)"
						}
					}
				}`)),
			},
		},
		RequiredCredentials: []connectors.ManifestCredential{
			{Service: "cloudflare", AuthType: "api_key", InstructionsURL: "https://developers.cloudflare.com/fundamentals/api/get-started/create-token/"},
		},
		Templates: []connectors.ManifestTemplate{
			{
				ID:          "tpl_cloudflare_list_zones",
				ActionType:  "cloudflare.list_zones",
				Name:        "List all zones",
				Description: "Agent can list all zones in the account.",
				Parameters:  json.RawMessage(`{}`),
			},
			{
				ID:          "tpl_cloudflare_manage_dns",
				ActionType:  "cloudflare.create_dns_record",
				Name:        "Create DNS records in a zone",
				Description: "Agent can create DNS records in a specific zone.",
				Parameters:  json.RawMessage(`{"zone_id":"*","type":"*","name":"*","content":"*","ttl":"*","proxied":"*"}`),
			},
			{
				ID:          "tpl_cloudflare_list_dns",
				ActionType:  "cloudflare.list_dns_records",
				Name:        "List DNS records",
				Description: "Agent can list DNS records for a zone.",
				Parameters:  json.RawMessage(`{"zone_id":"*"}`),
			},
			{
				ID:          "tpl_cloudflare_update_dns",
				ActionType:  "cloudflare.update_dns_record",
				Name:        "Update DNS records",
				Description: "Agent can update DNS records in a zone.",
				Parameters:  json.RawMessage(`{"zone_id":"*","record_id":"*","type":"*","name":"*","content":"*","ttl":"*","proxied":"*"}`),
			},
			{
				ID:          "tpl_cloudflare_delete_dns",
				ActionType:  "cloudflare.delete_dns_record",
				Name:        "Delete DNS records",
				Description: "Agent can delete DNS records from a zone.",
				Parameters:  json.RawMessage(`{"zone_id":"*","record_id":"*"}`),
			},
			{
				ID:          "tpl_cloudflare_list_tunnels",
				ActionType:  "cloudflare.list_tunnels",
				Name:        "List tunnels",
				Description: "Agent can list Cloudflare Tunnels.",
				Parameters:  json.RawMessage(`{"account_id":"*"}`),
			},
			{
				ID:          "tpl_cloudflare_create_tunnel",
				ActionType:  "cloudflare.create_tunnel",
				Name:        "Create a tunnel",
				Description: "Agent can create Cloudflare Tunnels.",
				Parameters:  json.RawMessage(`{"account_id":"*","name":"*","tunnel_secret":"*"}`),
			},
			{
				ID:          "tpl_cloudflare_purge_cache",
				ActionType:  "cloudflare.purge_cache",
				Name:        "Purge zone cache",
				Description: "Agent can purge cached content for a zone.",
				Parameters:  json.RawMessage(`{"zone_id":"*","purge_everything":"*","files":"*"}`),
			},
			{
				ID:          "tpl_cloudflare_check_domain",
				ActionType:  "cloudflare.check_domain",
				Name:        "Check domain registration",
				Description: "Agent can check domain registration status.",
				Parameters:  json.RawMessage(`{"account_id":"*","domain":"*"}`),
			},
			{
				ID:          "tpl_cloudflare_update_domain_settings",
				ActionType:  "cloudflare.update_domain_settings",
				Name:        "Update domain settings",
				Description: "Agent can update settings for domains already registered in Cloudflare Registrar.",
				Parameters:  json.RawMessage(`{"account_id":"*","domain":"*","auto_renew":"*"}`),
			},
		},
	}
}
