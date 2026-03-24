/**
 * Human-readable display labels for OAuth providers and credential services.
 * Shared across Settings and Connector configuration pages.
 */

const PROVIDER_LABELS: Record<string, string> = {
  airtable: "Airtable",
  atlassian: "Atlassian",
  discord: "Discord",
  figma: "Figma",
  github: "GitHub",
  google: "Google",
  hubspot: "HubSpot",
  intercom: "Intercom",
  instacart: "Instacart",
  kroger: "Kroger",
  linear: "Linear",
  linkedin: "LinkedIn",
  meta: "Meta",
  microsoft: "Microsoft",
  netlify: "Netlify",
  pagerduty: "PagerDuty",
  salesforce: "Salesforce",
  shopify: "Shopify",
  slack: "Slack",
  square: "Square",
  stripe: "Stripe",
  vercel: "Vercel",
  zendesk: "Zendesk",
  zoom: "Zoom",
};

/**
 * Returns a human-readable label for an OAuth provider ID.
 * Falls back to title-casing the ID if no explicit label is defined.
 */
export function providerLabel(id: string): string {
  return PROVIDER_LABELS[id] ?? id.charAt(0).toUpperCase() + id.slice(1);
}

const SERVICE_LABELS: Record<string, string> = {
  coinbase_agentkit: "Coinbase AgentKit (CDP)",
  github_pat: "GitHub Personal Access Token",
  "netlify-api-key": "Netlify API Key",
  "vercel-api-key": "Vercel API Key",
  square_api_key: "Square API Key",
};

/**
 * Returns a human-readable label for a credential service identifier.
 * Falls back to providerLabel() for proper casing when service IDs match
 * provider IDs (e.g. "github" → "GitHub").
 */
export function serviceLabel(service: string): string {
  return SERVICE_LABELS[service] ?? PROVIDER_LABELS[service] ?? service;
}

const AUTH_TYPE_LABELS: Record<string, string> = {
  api_key: "API Key",
  oauth2: "OAuth",
  basic: "Username & Password",
  custom: "Custom",
};

/**
 * Returns a human-readable label for an auth_type value.
 */
export function authTypeLabel(authType: string): string {
  return AUTH_TYPE_LABELS[authType] ?? authType;
}
