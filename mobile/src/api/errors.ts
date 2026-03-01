import type { components } from "./schema";

type ApiError = components["schemas"]["Error"];

/**
 * Extracts the human-readable error message from an openapi-fetch error response.
 *
 * All API error responses follow the ErrorResponse schema: `{ error: { code, message, ... } }`.
 * This function safely navigates that shape and returns the server-provided message,
 * falling back to the provided default when the shape doesn't match.
 */
export function getApiErrorMessage(
  error: unknown,
  fallback: string,
): string {
  if (typeof error !== "object" || error === null) return fallback;
  if (!("error" in error)) return fallback;

  // Safe: `in` check above guarantees "error" exists on the object.
  const inner = (error as { error: unknown }).error;
  if (typeof inner !== "object" || inner === null) return fallback;
  if (!("message" in inner)) return fallback;

  // Safe: `in` check above guarantees "message" exists on the inner object.
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

  // Safe: `in` check above guarantees "error" exists on the object.
  const inner = (error as { error: unknown }).error;
  if (typeof inner !== "object" || inner === null) return undefined;
  if (!("code" in inner)) return undefined;

  // Safe: `in` check + narrowing above guarantee "code" exists; cast matches the Error schema.
  return (inner as { code: unknown }).code as ApiError["code"];
}

/** Error codes returned when a plan resource limit is exceeded. */
const PLAN_LIMIT_ERROR_CODES: ReadonlySet<string> = new Set([
  "agent_limit_reached",
  "standing_approval_limit_reached",
  "credential_limit_reached",
]);

/**
 * Returns true if the error is a plan limit error (403 with a *_limit_reached code).
 * Use this to show upgrade-specific UI instead of a generic error toast.
 */
export function isPlanLimitError(error: unknown): boolean {
  const code = getApiErrorCode(error);
  return code !== undefined && PLAN_LIMIT_ERROR_CODES.has(code);
}
