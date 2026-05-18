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
  cachedHost = (await SecureStore.getItemAsync(CUSTOM_HOST_KEY)) ?? null;
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
  if (host && host.trim().length > 0) {
    await SecureStore.setItemAsync(CUSTOM_HOST_KEY, host.trim());
    cachedHost = host.trim();
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
