import { supabase } from "../lib/supabaseClient";
import type { AuthError } from "@supabase/supabase-js";

/**
 * Resolves the backend API base URL from the same env var the API client uses.
 * Falls back to production if not set.
 */
function getApiBaseUrl(): string {
  const envUrl = process.env.EXPO_PUBLIC_API_BASE_URL;
  if (!envUrl) {
    return "https://app.permissionslip.dev/api";
  }
  return envUrl.replace(/\/v1\/?$/, "").replace(/\/$/, "");
}

/**
 * Attempts to authenticate via the backend's app-review-login endpoint.
 *
 * This is used as a fallback when Supabase OTP verification fails — the
 * backend validates the email/OTP against pre-configured App Store review
 * credentials and returns a valid Supabase session.
 *
 * Returns { error: null } on success, or { error: AuthError } on failure.
 */
export async function tryAppReviewLogin(
  email: string,
  otp: string
): Promise<{ error: AuthError | null }> {
  try {
    const baseUrl = getApiBaseUrl();
    const response = await fetch(`${baseUrl}/v1/auth/app-review-login`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ email, otp }),
    });

    if (!response.ok) {
      return {
        error: {
          message: "Invalid credentials",
          name: "AuthApiError",
          status: response.status,
          code: "invalid_credentials",
        } as AuthError,
      };
    }

    const session = await response.json();

    // Set the session in the Supabase client so onAuthStateChange fires
    // and the rest of the app picks up the authenticated state.
    const { error } = await supabase.auth.setSession({
      access_token: session.access_token,
      refresh_token: session.refresh_token,
    });

    return { error: error ?? null };
  } catch {
    return {
      error: {
        message: "Network error",
        name: "AuthApiError",
        status: 0,
        code: "network_error",
      } as AuthError,
    };
  }
}
