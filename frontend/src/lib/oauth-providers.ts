/**
 * Shared OAuth provider display labels. Used across settings pages and
 * connector credential sections to render human-friendly provider names.
 *
 * Add new providers here rather than in individual components.
 */
const PROVIDER_LABELS: Record<string, string> = {
  google: "Google",
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
 * Returns a human-friendly label for an OAuth provider ID.
 * Falls back to title-casing the ID if no label is defined.
 */
export function providerLabel(id: string): string {
  return PROVIDER_LABELS[id] ?? id.charAt(0).toUpperCase() + id.slice(1);
}

/** User-friendly labels for auth types. */
export function authTypeLabel(authType: string): string {
  switch (authType) {
    case "api_key":
      return "API Key";
    case "basic":
      return "Username & Password";
    case "oauth2":
      return "OAuth";
    case "custom":
      return "Custom";
    default:
      return authType;
  }
}

/**
 * Derives a friendly display name from a service ID like "square_api_key"
 * by stripping common credential-type suffixes and looking up the base name.
 */
export function serviceDisplayName(service: string): string {
  const base = service.replace(/_(api_key|oauth|token|creds?)$/i, "");
  return providerLabel(base);
}
