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
/**
 * Options for building an OAuth authorize URL.
 *
 * @param replaceId - When set, the backend replaces the specified existing
 *   connection instead of creating a new one alongside it. Used for the
 *   "Reconnect" flow.
 */
interface OAuthAuthorizeOptions {
  scopes?: string[];
  replaceId?: string;
}

export function getOAuthAuthorizeUrl(
  providerId: string,
  accessToken: string,
  scopesOrOptions?: string[] | OAuthAuthorizeOptions,
): string {
  // Backward-compatible: accept either scopes array or options object.
  const options: OAuthAuthorizeOptions = Array.isArray(scopesOrOptions)
    ? { scopes: scopesOrOptions }
    : scopesOrOptions ?? {};

  const baseUrl =
    import.meta.env.VITE_API_BASE_URL?.replace(/\/v1\/?$/, "") ?? "/api";
  const url = new URL(window.location.href);
  url.searchParams.delete("oauth_status");
  url.searchParams.delete("oauth_provider");
  url.searchParams.delete("oauth_error");
  const returnTo = url.pathname + (url.search || "") + url.hash;
  let result = `${baseUrl}/v1/oauth/${providerId}/authorize?access_token=${encodeURIComponent(accessToken)}&return_to=${encodeURIComponent(returnTo)}`;
  if (options.scopes?.length) {
    for (const s of options.scopes) {
      result += `&scope=${encodeURIComponent(s)}`;
    }
  }
  if (options.replaceId) {
    result += `&replace=${encodeURIComponent(options.replaceId)}`;
  }
  return result;
}
