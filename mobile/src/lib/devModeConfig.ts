import * as SecureStore from "expo-secure-store";

const DEV_MODE_KEY = "developer_mode_enabled";

let cachedEnabled = false;
const listeners = new Set<() => void>();

/**
 * Hydrate the in-memory cache from SecureStore during app startup. Called
 * once from App.tsx before any subtree that reads the toggle is mounted.
 * Returns immediately when the stored value matches the current cache.
 */
export async function loadDevModeConfig(): Promise<void> {
  const stored = await SecureStore.getItemAsync(DEV_MODE_KEY);
  cachedEnabled = stored === "1";
}

/** Synchronous accessor used by middleware on every request. */
export function isDevModeEnabled(): boolean {
  return cachedEnabled;
}

/** Persist the new value and notify subscribers (toggle UI, overlay). */
export async function setDevModeEnabled(enabled: boolean): Promise<void> {
  cachedEnabled = enabled;
  if (enabled) {
    await SecureStore.setItemAsync(DEV_MODE_KEY, "1");
  } else {
    await SecureStore.deleteItemAsync(DEV_MODE_KEY);
  }
  for (const listener of listeners) listener();
}

/** Subscribe to dev-mode toggle changes (for useSyncExternalStore). */
export function subscribeDevMode(listener: () => void): () => void {
  listeners.add(listener);
  return () => listeners.delete(listener);
}
