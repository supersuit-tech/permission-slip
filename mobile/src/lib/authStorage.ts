import * as SecureStore from "expo-secure-store";

const REFRESH_TOKEN_KEY = "ps_refresh_token";

export async function getStoredRefreshToken(): Promise<string | null> {
  try {
    return await SecureStore.getItemAsync(REFRESH_TOKEN_KEY);
  } catch {
    return null;
  }
}

export async function setStoredRefreshToken(token: string): Promise<void> {
  try {
    await SecureStore.setItemAsync(REFRESH_TOKEN_KEY, token);
  } catch {
    // quota or unavailable
  }
}

export async function clearStoredRefreshToken(): Promise<void> {
  try {
    await SecureStore.deleteItemAsync(REFRESH_TOKEN_KEY);
  } catch {
    // ignore
  }
}
