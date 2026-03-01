import * as SecureStore from "expo-secure-store";

/**
 * Adapter that gives expo-secure-store the same interface Supabase expects
 * from its `storage` option (getItem / setItem / removeItem returning promises).
 *
 * This stores auth tokens in the platform's secure keychain (iOS Keychain,
 * Android EncryptedSharedPreferences) instead of plain-text AsyncStorage,
 * protecting them on rooted/jailbroken devices.
 *
 * expo-secure-store has a ~2 KB value limit on some platforms. Supabase
 * sessions are well within this (JWTs are typically ~1 KB). If a value
 * exceeds the limit, setItemAsync throws — the Supabase client will treat
 * this as a storage failure and fall back to an in-memory session (still
 * functional, just not persisted across app restarts).
 */
export const secureStorage = {
  getItem: (key: string): Promise<string | null> => {
    return SecureStore.getItemAsync(key);
  },

  setItem: (key: string, value: string): Promise<void> => {
    return SecureStore.setItemAsync(key, value);
  },

  removeItem: (key: string): Promise<void> => {
    return SecureStore.deleteItemAsync(key);
  },
};
