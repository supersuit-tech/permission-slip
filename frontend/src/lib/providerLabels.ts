/**
 * Shared provider display name mapping. Used across the settings page,
 * connector credential sections, and OAuth provider management.
 *
 * Add new providers here — all consumers import from this single source.
 */
const PROVIDER_LABELS: Record<string, string> = {
  figma: "Figma",
  google: "Google",
  intercom: "Intercom",
  microsoft: "Microsoft",
  salesforce: "Salesforce",
  zoom: "Zoom",
};

export function providerLabel(id: string): string {
  return PROVIDER_LABELS[id] ?? id.charAt(0).toUpperCase() + id.slice(1);
}
