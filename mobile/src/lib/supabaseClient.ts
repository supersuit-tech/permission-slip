import "react-native-url-polyfill/auto";
import { createClient } from "@supabase/supabase-js";
import { secureStorage } from "./secureStorage";

// Expo inlines EXPO_PUBLIC_* env vars at build time (SDK 49+).
// Set these in a .env file or your shell environment.
const supabaseUrl = process.env.EXPO_PUBLIC_SUPABASE_URL;
const supabasePublishableKey = process.env.EXPO_PUBLIC_SUPABASE_PUBLISHABLE_KEY;

if (!supabaseUrl || !supabasePublishableKey) {
  throw new Error(
    "Missing Supabase configuration. Set EXPO_PUBLIC_SUPABASE_URL and " +
      "EXPO_PUBLIC_SUPABASE_PUBLISHABLE_KEY in your environment (e.g. in .env)."
  );
}

export const supabase = createClient(supabaseUrl, supabasePublishableKey, {
  auth: {
    // Use expo-secure-store (Keychain / EncryptedSharedPreferences) instead
    // of AsyncStorage to protect auth tokens on rooted/jailbroken devices.
    storage: secureStorage,
    autoRefreshToken: true,
    persistSession: true,
    detectSessionInUrl: false, // Disable for React Native — no browser URL bar
  },
});
