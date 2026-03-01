import createClient from "openapi-fetch";
import type { paths } from "./schema";

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
  if (!envUrl) return "https://app.permissionslip.dev/api";
  return envUrl.replace(/\/v1\/?$/, "").replace(/\/$/, "");
}

/**
 * Typed API client generated from the OpenAPI spec.
 * Uses the same `openapi-fetch` library as the web frontend.
 * Spec paths already include the "/v1" prefix, so the base URL is version-free.
 */
const client = createClient<paths>({ baseUrl: resolveBaseUrl() });

export default client;
