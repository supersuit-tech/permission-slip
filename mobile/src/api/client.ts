import createClient from "openapi-fetch";
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

export default client;
