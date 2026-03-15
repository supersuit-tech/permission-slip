/**
 * Shared OAuth utilities for URL construction.
 * Provider labels are re-exported from @/lib/labels for convenience.
 */

export { providerLabel } from "@/lib/labels";

/**
 * Developer console URLs for OAuth providers that may need BYOA setup.
 * Used by the BYOASetupBanner to link users to the right place when they
 * need to create an OAuth app.
 */
export const PROVIDER_DEV_CONSOLE_URLS: Partial<Record<string, string>> = {
  atlassian: "https://developer.atlassian.com/console/myapps/",
  datadog: "https://app.datadoghq.com/oauth/manage",
  dropbox: "https://www.dropbox.com/developers/apps",
  figma: "https://www.figma.com/developers/apps",
  github: "https://github.com/settings/developers",
  google: "https://console.cloud.google.com/apis/credentials",
  hubspot: "https://developers.hubspot.com/docs/api/oauth/tokens",
  linear: "https://linear.app/settings/api",
  meta: "https://developers.facebook.com/apps/",
  microsoft: "https://portal.azure.com/#blade/Microsoft_AAD_RegisteredApps/ApplicationsListBlade",
  notion: "https://www.notion.so/my-integrations",
  pagerduty: "https://developer.pagerduty.com/apps/",
  salesforce: "https://developer.salesforce.com/docs/atlas.en-us.api_rest.meta/api_rest/intro_oauth_and_connected_apps.htm",
  shopify: "https://help.shopify.com/en/manual/apps/app-types/custom-apps",
  square: "https://developer.squareup.com/apps",
  stripe: "https://dashboard.stripe.com/settings/applications",
  x: "https://developer.x.com/en/portal/dashboard",
};

/**
 * Providers that require a shop subdomain to construct per-shop OAuth URLs.
 * When a provider is in this set, the UI prompts the user for their store
 * subdomain (e.g. "mystore") which is appended as `&shop=mystore` to the
 * authorize URL.
 */
export const SHOP_REQUIRED_PROVIDERS = new Set(["shopify"]);

/**
 * Options for building an OAuth authorize URL.
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
