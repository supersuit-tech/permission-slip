/**
 * Shared OAuth provider display labels. Used across settings pages and
 * connector credential sections to render human-friendly provider names.
 *
 * Add new providers here rather than in individual components.
 */
const PROVIDER_LABELS: Record<string, string> = {
  figma: "Figma",
  google: "Google",
  intercom: "Intercom",
  kroger: "Kroger",
  linkedin: "LinkedIn",
  meta: "Meta",
  microsoft: "Microsoft",
  salesforce: "Salesforce",
  stripe: "Stripe",
  zoom: "Zoom",
};

/**
 * Returns a human-friendly label for an OAuth provider ID.
 * Falls back to title-casing the ID if no label is defined.
 */
export function providerLabel(id: string): string {
  return PROVIDER_LABELS[id] ?? id.charAt(0).toUpperCase() + id.slice(1);
}
