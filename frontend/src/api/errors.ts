import type { components } from "./schema";

type ApiError = components["schemas"]["Error"];

/**
 * Extracts the human-readable error message from an openapi-fetch error response.
 *
 * All our API error responses follow the ErrorResponse schema: `{ error: { code, message, ... } }`.
 * This function safely navigates that shape and returns the server-provided message,
 * falling back to the provided default when the shape doesn't match.
 */
export function getApiErrorMessage(
  error: unknown,
  fallback: string,
): string {
  if (typeof error !== "object" || error === null) return fallback;
  if (!("error" in error)) return fallback;

  const inner = (error as { error: unknown }).error;
  if (typeof inner !== "object" || inner === null) return fallback;
  if (!("message" in inner)) return fallback;

  const msg = (inner as { message: unknown }).message;
  return typeof msg === "string" ? msg : fallback;
}

/**
 * Extracts the machine-readable error code from an openapi-fetch error response.
 * Returns undefined if the error doesn't match the expected shape.
 */
export function getApiErrorCode(error: unknown): ApiError["code"] | undefined {
  if (typeof error !== "object" || error === null) return undefined;
  if (!("error" in error)) return undefined;

  const inner = (error as { error: unknown }).error;
  if (typeof inner !== "object" || inner === null) return undefined;
  if (!("code" in inner)) return undefined;

  return (inner as { code: unknown }).code as ApiError["code"];
}
