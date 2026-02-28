import createClient from "openapi-fetch";
import type { paths } from "./schema";

/**
 * Strips any /v1 suffix and trailing slashes from the env-provided URL,
 * yielding the version-free API root (e.g. "/api").
 */
function normalizeBase(): string {
  const envUrl = import.meta.env.VITE_API_BASE_URL;
  if (!envUrl) return "/api";
  return envUrl.replace(/\/v1\/?$/, "").replace(/\/$/, "");
}

function resolveBaseUrl(): string {
  const base = normalizeBase();
  // openapi-fetch requires an absolute URL for the URL constructor.
  // In the browser, resolve relative paths against the current origin.
  if (base.startsWith("/") && typeof globalThis.location !== "undefined") {
    return `${globalThis.location.origin}${base}`;
  }
  return base;
}

/**
 * Typed API client generated from the OpenAPI spec.
 * Uses "/api" as the base because spec paths already include the "/v1" prefix.
 */
const client = createClient<paths>({ baseUrl: resolveBaseUrl() });

export default client;
