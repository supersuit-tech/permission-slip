/**
 * Maps opaque action-parameter IDs to human-readable names from
 * resource_details. Implementation lives in @permission-slip/constraints-format
 * so web, mobile, and CLI stay in sync.
 */
import {
  lookupResolvedResource,
  overlayResolvedNamesOnParams,
  safeHttpsHref,
  type ResolvedResource,
} from "@permission-slip/constraints-format";

export type { ResolvedResource };
export { overlayResolvedNamesOnParams, safeHttpsHref, lookupResolvedResource };

/** Display name for a parameter, or null when resource_details has no overlay. */
export function resolvedResourceDisplayValue(
  paramKey: string,
  rawValue: unknown,
  resourceDetails?: Record<string, unknown> | null,
): string | null {
  return lookupResolvedResource(paramKey, rawValue, resourceDetails)?.name ?? null;
}

export function resolvedResourceHref(
  paramKey: string,
  rawValue: unknown,
  resourceDetails?: Record<string, unknown> | null,
): string | undefined {
  return lookupResolvedResource(paramKey, rawValue, resourceDetails)?.url;
}
