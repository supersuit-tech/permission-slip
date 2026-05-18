import { createAuthError } from "../auth/errors";
import type { AuthError } from "../auth/types";
import {
  getCustomHost,
  getGatewaySecret,
  PLACEHOLDER_API_BASE,
} from "./customHostConfig";

function defaultApiRoot(): string {
  const envUrl = process.env.EXPO_PUBLIC_API_BASE_URL?.trim();
  if (envUrl) {
    return envUrl.replace(/\/v1\/?$/, "").replace(/\/$/, "");
  }
  return PLACEHOLDER_API_BASE.replace(/\/v1\/?$/, "").replace(/\/$/, "");
}

export function apiRoot(): string {
  const custom = getCustomHost();
  if (custom) {
    return custom.replace(/\/v1\/?$/, "").replace(/\/$/, "");
  }
  return defaultApiRoot();
}

export type AuthTokenResponse = {
  access_token: string;
  refresh_token: string;
  expires_at: string;
};

function parseJsonSafe(text: string): unknown {
  try {
    return JSON.parse(text) as unknown;
  } catch {
    return null;
  }
}

export async function postAuth(
  path: "signup" | "login" | "refresh" | "logout",
  body: Record<string, string>,
): Promise<{ data: AuthTokenResponse | null; error: AuthError | null }> {
  const url = `${apiRoot()}/auth/${path}`;
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
  };
  const secret = getGatewaySecret();
  if (secret) {
    headers["X-Gateway-Secret"] = secret;
  }

  const res = await fetch(url, {
    method: "POST",
    headers,
    body: JSON.stringify(body),
  });

  if (path === "logout") {
    if (res.status === 204) {
      return { data: null, error: null };
    }
    const msg = await res.text();
    return {
      data: null,
      error: createAuthError("logout_failed", msg || "Logout failed", res.status),
    };
  }

  const text = await res.text();
  const json = parseJsonSafe(text) as Record<string, unknown> | null;

  if (!res.ok) {
    const code =
      json &&
      typeof json === "object" &&
      "error" in json &&
      typeof (json as { error: unknown }).error === "object" &&
      (json as { error: { code?: unknown } }).error !== null &&
      typeof (json as { error: { code?: unknown } }).error.code === "string"
        ? String((json as { error: { code: string } }).error.code)
        : "request_failed";
    const message =
      json &&
      typeof json === "object" &&
      "error" in json &&
      typeof (json as { error: unknown }).error === "object" &&
      (json as { error: { message?: unknown } }).error !== null &&
      typeof (json as { error: { message?: unknown } }).error.message === "string"
        ? String((json as { error: { message: string } }).error.message)
        : "Request failed";
    return {
      data: null,
      error: createAuthError(code, message, res.status),
    };
  }

  if (!json || typeof json.access_token !== "string" || typeof json.refresh_token !== "string") {
    return {
      data: null,
      error: createAuthError("invalid_response", "Unexpected response from server", res.status),
    };
  }

  const expiresAt =
    typeof json.expires_at === "string"
      ? json.expires_at
      : new Date(Date.now() + 15 * 60 * 1000).toISOString();

  return {
    data: {
      access_token: json.access_token,
      refresh_token: json.refresh_token,
      expires_at: expiresAt,
    },
    error: null,
  };
}
