/**
 * Shared OAuth utilities for URL construction.
 * Provider labels are re-exported from @/lib/labels for convenience.
 */

export { providerLabel } from "@/lib/labels";

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
