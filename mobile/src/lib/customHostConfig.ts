import * as SecureStore from "expo-secure-store";

const CUSTOM_HOST_KEY = "custom_host_url";
const GATEWAY_SECRET_KEY = "gateway_secret";

/**
 * Placeholder base URL for openapi-fetch when neither EXPO_PUBLIC_API_BASE_URL
 * nor a saved custom host exists yet. {@link customHostMiddleware} rewrites this
 * prefix to the configured self-hosted API base after the user saves a URL.
 */
export const PLACEHOLDER_API_BASE = "https://__permission_slip_no_host__.invalid/api";

/**
 * Canonicalize a user-entered server URL into the form the app needs to talk
 * to the backend. Users enter the base origin of their deployment (the same
 * URL they use in the web app, e.g. https://permission-slip.example.com);
 * the backend mounts the API at `/api`, so we append it automatically when
 * it's missing. Without this, a user who types only the origin gets HTML
 * 404s from their reverse proxy on every auth request and the app surfaces
 * a confusing "Something went wrong" error.
 *
 * Behavior:
 *   - Strips surrounding whitespace, trailing slashes, and a trailing `/v1`
 *     (which spec paths already include).
 *   - Returns `null` for empty input so callers can distinguish "not set".
 *   - Idempotent: a URL already ending in `/api` is left alone, so we never
 *     produce `…/api/api`.
 *   - Conservative on custom paths: any non-empty path that doesn't already
 *     end in `/api` gets `/api` appended, including `https://host.tld/foo`
 *     → `https://host.tld/foo/api`. Operators with non-standard mounts can
 *     still enter the full path explicitly.
 */
export function normalizeApiBase(input: string | null | undefined): string | null {
  if (input == null) return null;
  let url = input.trim();
  if (!url) return null;
  url = url.replace(/\/+$/, "");
  url = url.replace(/\/v1$/, "");
  url = url.replace(/\/+$/, "");
  if (!url) return null;
  if (!/\/api$/.test(url)) {
    url = `${url}/api`;
  }
  return url;
}

/**
 * In-memory cache so middleware reads are synchronous and fast.
 * Call loadCustomHostConfig() at app startup to hydrate from SecureStore.
 */
let cachedHost: string | null = null;
let cachedSecret: string | null = null;

/**
 * Hydrate the in-memory cache from SecureStore during app startup
 * (see `App.tsx`, which gates rendering on this promise). The API client
 * in `client.ts` is constructed at module import — well before `App`
 * runs — so the middleware synchronously reads this in-memory cache via
 * `getCustomHost()` / `getGatewaySecret()` on each request. Blocking UI
 * render until hydration completes is what prevents the first request
 * from firing before the cache is populated.
 */
export async function loadCustomHostConfig(): Promise<void> {
  const storedHost = (await SecureStore.getItemAsync(CUSTOM_HOST_KEY)) ?? null;
  // Normalize on read so existing saves from older app versions (which
  // required the user to type `/api` themselves) start working even before
  // the user re-enters their URL.
  cachedHost = normalizeApiBase(storedHost);
  cachedSecret = (await SecureStore.getItemAsync(GATEWAY_SECRET_KEY)) ?? null;
}

/** Returns the custom host URL, or null if not configured. */
export function getCustomHost(): string | null {
  return cachedHost;
}

/** Returns the gateway secret, or null if not configured. */
export function getGatewaySecret(): string | null {
  return cachedSecret;
}

/** Returns true when a custom host is configured and non-empty. */
export function isCustomHostEnabled(): boolean {
  return cachedHost != null && cachedHost.length > 0;
}

/**
 * True when the app has an explicit API base: build-time env or a saved
 * custom host. Mock-auth dev mode does not need a real host.
 */
export function hasConfiguredApiBase(): boolean {
  const env = process.env.EXPO_PUBLIC_API_BASE_URL?.trim();
  if (env && env.length > 0) {
    return true;
  }
  return isCustomHostEnabled();
}

/**
 * Persist custom host config to SecureStore and update the in-memory cache.
 * Pass null to clear a value.
 */
export async function setCustomHostConfig(
  host: string | null,
  secret: string | null,
): Promise<void> {
  const normalized = normalizeApiBase(host);
  if (normalized) {
    await SecureStore.setItemAsync(CUSTOM_HOST_KEY, normalized);
    cachedHost = normalized;
  } else {
    await SecureStore.deleteItemAsync(CUSTOM_HOST_KEY);
    cachedHost = null;
  }
  if (secret && secret.trim().length > 0) {
    await SecureStore.setItemAsync(GATEWAY_SECRET_KEY, secret.trim());
    cachedSecret = secret.trim();
  } else {
    await SecureStore.deleteItemAsync(GATEWAY_SECRET_KEY);
    cachedSecret = null;
  }
}

/**
 * Clear all custom host configuration. The app will require EXPO_PUBLIC_API_BASE_URL
 * or a newly saved server URL (see first-launch server setup) before API calls work.
 */
export async function clearCustomHostConfig(): Promise<void> {
  await SecureStore.deleteItemAsync(CUSTOM_HOST_KEY);
  await SecureStore.deleteItemAsync(GATEWAY_SECRET_KEY);
  cachedHost = null;
  cachedSecret = null;
}
