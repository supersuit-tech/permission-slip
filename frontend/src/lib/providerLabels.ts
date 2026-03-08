/** Human-readable display names for OAuth provider IDs. */
const PROVIDER_LABELS: Record<string, string> = {
  google: "Google",
  intercom: "Intercom",
  kroger: "Kroger",
  linkedin: "LinkedIn",
  meta: "Meta",
  microsoft: "Microsoft",
  pagerduty: "PagerDuty",
  salesforce: "Salesforce",
  zoom: "Zoom",
};

/**
 * Returns the display-friendly label for an OAuth provider ID.
 * Falls back to naive capitalization if the provider isn't in the map.
 */
export function providerLabel(id: string): string {
  return PROVIDER_LABELS[id] ?? id.charAt(0).toUpperCase() + id.slice(1);
}
