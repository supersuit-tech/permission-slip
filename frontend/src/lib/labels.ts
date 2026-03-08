/**
 * Human-readable display labels for OAuth providers and credential services.
 * Shared across Settings and Connector configuration pages.
 */

const PROVIDER_LABELS: Record<string, string> = {
  github: "GitHub",
  google: "Google",
  hubspot: "HubSpot",
  intercom: "Intercom",
  kroger: "Kroger",
  linkedin: "LinkedIn",
  meta: "Meta",
  microsoft: "Microsoft",
  salesforce: "Salesforce",
  square: "Square",
  stripe: "Stripe",
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
  github_pat: "GitHub Personal Access Token",
  square_api_key: "Square API Key",
};

/**
 * Returns a human-readable label for a credential service identifier.
 * Falls back to the raw service ID if no explicit label is defined.
 */
export function serviceLabel(service: string): string {
  return SERVICE_LABELS[service] ?? service;
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
