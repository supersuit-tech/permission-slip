/**
 * Shared provider display-name mapping and label helper used across
 * OAuth-related UI (connector credentials, connected accounts, BYOA config).
 */

const PROVIDER_LABELS: Record<string, string> = {
  google: "Google",
  intercom: "Intercom",
  microsoft: "Microsoft",
  netlify: "Netlify",
};

/**
 * Returns a human-friendly display name for an OAuth provider ID.
 * Falls back to capitalising the first letter when no explicit label exists.
 */
export function providerLabel(id: string): string {
  return PROVIDER_LABELS[id] ?? id.charAt(0).toUpperCase() + id.slice(1);
}

/**
 * Formats a raw service name into a user-friendly label.
 * Strips connector-ID prefixes, replaces separators with spaces, and
 * title-cases the result — e.g. "netlify-api-key" → "API Key".
 */
export function formatServiceName(service: string): string {
  // Strip common connector-id prefix (e.g. "netlify-" from "netlify-api-key")
  const stripped = service.replace(/^[a-z]+-/, "");
  return stripped
    .replace(/[_-]/g, " ")
    .replace(/\bapi\b/gi, "API")
    .replace(/\boauth\b/gi, "OAuth")
    .replace(/\b\w/g, (c) => c.toUpperCase());
}
