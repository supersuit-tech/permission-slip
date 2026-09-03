/**
 * Maps opaque action-parameter IDs to human-readable names from
 * resource_details (e.g. folder_id → folder_name, calendar_id → calendar_name).
 *
 * Convention: `{foo}_id` resolves via `foo_name`; a bare key like `channel`
 * resolves via `channel_name`.
 */
export function resolvedResourceDisplayValue(
  paramKey: string,
  rawValue: unknown,
  resourceDetails?: Record<string, unknown> | null,
): string | null {
  if (resourceDetails == null) return null;
  if (typeof rawValue !== "string" && typeof rawValue !== "number") return null;
  const raw = String(rawValue);
  if (raw.length === 0) return null;

  const nameKey = resourceNameKey(paramKey);
  const name = resourceDetails[nameKey];
  if (typeof name === "string" && name.length > 0 && name !== raw) {
    return `${name} (${raw})`;
  }
  return null;
}

function resourceNameKey(paramKey: string): string {
  if (paramKey.endsWith("_id")) {
    return `${paramKey.slice(0, -3)}_name`;
  }
  return `${paramKey}_name`;
}
