/**
 * Canonical display labels for known OAuth provider IDs.
 * Centralised here to avoid inconsistent capitalisation across pages
 * (e.g. "linkedin" → "Linkedin" vs "LinkedIn").
 */
const PROVIDER_LABELS: Record<string, string> = {
  google: "Google",
  intercom: "Intercom",
  kroger: "Kroger",
  linear: "Linear",
  linkedin: "LinkedIn",
  meta: "Meta",
  microsoft: "Microsoft",
  salesforce: "Salesforce",
  zoom: "Zoom",
};

/**
 * Returns a human-friendly label for an OAuth provider ID.
 * Uses the canonical map for known providers and falls back to
 * title-casing the first character for unknown ones.
 */
export function providerLabel(id: string): string {
  return PROVIDER_LABELS[id] ?? id.charAt(0).toUpperCase() + id.slice(1);
}
