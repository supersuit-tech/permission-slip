/** Build the full URL for initiating an OAuth authorization redirect.
 *  Used by both the connector credentials section and the settings page. */
export function buildOAuthAuthorizeUrl(
  provider: string,
  accessToken: string,
): string {
  const baseUrl =
    import.meta.env.VITE_API_BASE_URL?.replace(/\/v1\/?$/, "") ?? "/api";
  const authorizeUrl = `${baseUrl}/v1/oauth/${provider}/authorize`;
  return `${authorizeUrl}?access_token=${encodeURIComponent(accessToken)}`;
}
