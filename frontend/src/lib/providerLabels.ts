/** Canonical display names for OAuth provider IDs. */
const PROVIDER_LABELS: Record<string, string> = {
  google: "Google",
  microsoft: "Microsoft",
  square: "Square",
  zoom: "Zoom",
  salesforce: "Salesforce",
  meta: "Meta",
  linkedin: "LinkedIn",
  kroger: "Kroger",
};

/**
 * Returns a user-friendly display name for an OAuth provider ID.
 * Falls back to title-casing the raw ID if no explicit label exists.
 */
export function providerLabel(id: string): string {
  return PROVIDER_LABELS[id] ?? id.charAt(0).toUpperCase() + id.slice(1);
}

/**
 * Derives a friendly display name from a service ID like "square_api_key"
 * by stripping common credential-type suffixes and looking up the base name.
 */
export function serviceDisplayName(service: string): string {
  const base = service.replace(/_(api_key|oauth|token|creds?)$/i, "");
  return providerLabel(base);
}
