/**
 * Shared OAuth utilities for provider labels and URL construction.
 */

const PROVIDER_LABELS: Record<string, string> = {
  google: "Google",
  hubspot: "HubSpot",
  intercom: "Intercom",
  kroger: "Kroger",
  linkedin: "LinkedIn",
  meta: "Meta",
  microsoft: "Microsoft",
  salesforce: "Salesforce",
  slack: "Slack",
  stripe: "Stripe",
  zoom: "Zoom",
};

/** Returns a human-readable label for an OAuth provider ID. */
export function providerLabel(id: string): string {
  return PROVIDER_LABELS[id] ?? id.charAt(0).toUpperCase() + id.slice(1);
}

/**
 * Builds the URL to initiate an OAuth authorization flow for a provider.
 * The backend redirects from this URL to the provider's consent screen.
 */
export function getOAuthAuthorizeUrl(
  providerId: string,
  accessToken: string,
): string {
  const baseUrl =
    import.meta.env.VITE_API_BASE_URL?.replace(/\/v1\/?$/, "") ?? "/api";
  return `${baseUrl}/v1/oauth/${providerId}/authorize?access_token=${encodeURIComponent(accessToken)}`;
}
