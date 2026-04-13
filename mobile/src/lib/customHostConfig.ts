import * as SecureStore from "expo-secure-store";

const CUSTOM_HOST_KEY = "custom_host_url";
const GATEWAY_SECRET_KEY = "gateway_secret";

/**
 * In-memory cache so middleware reads are synchronous and fast.
 * Call loadCustomHostConfig() at app startup to hydrate from SecureStore.
 */
let cachedHost: string | null = null;
let cachedSecret: string | null = null;

/**
 * Hydrate the in-memory cache from SecureStore. Call once at app startup
 * before creating the API client.
 */
export async function loadCustomHostConfig(): Promise<void> {
  cachedHost = (await SecureStore.getItemAsync(CUSTOM_HOST_KEY)) ?? null;
  cachedSecret = (await SecureStore.getItemAsync(GATEWAY_SECRET_KEY)) ?? null;
}

/** Returns the custom host URL, or null if using the default. */
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
 * Clear all custom host configuration. Restores the app to using the
 * default production host.
 */
export async function clearCustomHostConfig(): Promise<void> {
  await SecureStore.deleteItemAsync(CUSTOM_HOST_KEY);
  await SecureStore.deleteItemAsync(GATEWAY_SECRET_KEY);
  cachedHost = null;
  cachedSecret = null;
}
