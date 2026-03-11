/**
 * Shared OAuth utilities for URL construction.
 * Provider labels are re-exported from @/lib/labels for convenience.
 */

export { providerLabel } from "@/lib/labels";

/**
 * Providers that require a shop subdomain to construct per-shop OAuth URLs.
 * When a provider is in this set, the UI prompts the user for their store
 * subdomain (e.g. "mystore") which is appended as `&shop=mystore` to the
 * authorize URL.
 */
export const SHOP_REQUIRED_PROVIDERS = new Set(["shopify"]);

/**
 * Builds the URL to initiate an OAuth authorization flow for a provider.
 * The backend redirects from this URL to the provider's consent screen.
 * After the OAuth flow completes, the user is redirected back to the
 * current page (window.location.pathname + search + hash).
 */
export function getOAuthAuthorizeUrl(
  providerId: string,
  accessToken: string,
): string {
  const baseUrl =
    import.meta.env.VITE_API_BASE_URL?.replace(/\/v1\/?$/, "") ?? "/api";
  const url = new URL(window.location.href);
  url.searchParams.delete("oauth_status");
  url.searchParams.delete("oauth_provider");
  url.searchParams.delete("oauth_error");
  const returnTo = url.pathname + (url.search || "") + url.hash;
  return `${baseUrl}/v1/oauth/${providerId}/authorize?access_token=${encodeURIComponent(accessToken)}&return_to=${encodeURIComponent(returnTo)}`;
}
