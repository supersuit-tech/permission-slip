import createClient, { type Middleware } from "openapi-fetch";
import type { paths } from "./schema";
import mockClient from "./mockClient";

/**
 * Returns the API base URL for the mobile app.
 *
 * In React Native there is no browser origin to resolve relative paths against,
 * so the base URL must always be an absolute URL. It reads from the
 * EXPO_PUBLIC_API_BASE_URL env var (set via Expo's env config or .env file),
 * stripping any trailing "/v1" since spec paths already include the prefix.
 *
 * Falls back to the production URL if no env var is set.
 */
function resolveBaseUrl(): string {
  const envUrl = process.env.EXPO_PUBLIC_API_BASE_URL;
  if (!envUrl) {
    if (__DEV__) {
      console.warn(
        "[api] EXPO_PUBLIC_API_BASE_URL is not set — falling back to production (https://app.permissionslip.dev/api). " +
          "Set it in your .env or app.config to point at a local/staging server.",
      );
    }
    return "https://app.permissionslip.dev/api";
  }
  return envUrl.replace(/\/v1\/?$/, "").replace(/\/$/, "");
}

const useMockApi = __DEV__ && process.env.EXPO_PUBLIC_MOCK_AUTH === "true";

/**
 * Middleware that converts non-JSON responses into structured JSON errors.
 *
 * When a reverse proxy, CDN, or the server's SPA handler returns HTML instead
 * of JSON, openapi-fetch tries to JSON.parse() the body and throws a raw
 * SyntaxError ("JSON Parse error: Unexpected character: <"). This middleware
 * intercepts those responses and replaces them with a well-formed JSON error
 * body that the existing hook error handling can process.
 */
export const jsonSafeMiddleware: Middleware = {
  async onResponse({ response }) {
    // Let bodyless responses through (204 No Content, 304 Not Modified, etc.)
    if (!response.body || response.status === 204 || response.status === 304) {
      return response;
    }

    const contentType = response.headers.get("content-type") ?? "";
    if (contentType.includes("application/json")) {
      return response;
    }

    // Non-JSON body (HTML from SPA handler, proxy error page, plain text, etc.)
    // Convert to a structured JSON error so hook-level `if (error)` handling works.
    const status = response.status >= 400 ? response.status : 502;
    const errorBody = JSON.stringify({
      error: {
        code: "non_json_response",
        message:
          "Unable to reach the server. Please check your connection and try again.",
        // Treat 502/503/504 (and rewritten 2xx→502) as transient; 4xx are not.
        retryable: status >= 500,
      },
    });
    return new Response(errorBody, {
      status,
      statusText: response.statusText,
      headers: { "Content-Type": "application/json" },
    });
  },
};

/**
 * Typed API client generated from the OpenAPI spec.
 * Uses the same `openapi-fetch` library as the web frontend.
 * Spec paths already include the "/v1" prefix, so the base URL is version-free.
 *
 * When EXPO_PUBLIC_MOCK_AUTH=true in dev mode, a mock client is used instead
 * so the app works without a running backend.
 */
// Safe: mockClient implements only GET/POST — the subset the mobile app uses.
// TypeScript enforcement is bypassed because the mock doesn't expose the full
// openapi-fetch interface. resolveBaseUrl() is only called when mock is off.
// eslint-disable-next-line @typescript-eslint/no-explicit-any
const client = useMockApi
  ? (mockClient as any)
  : createClient<paths>({ baseUrl: resolveBaseUrl() });

if (!useMockApi) {
  client.use(jsonSafeMiddleware);
}

export default client;
