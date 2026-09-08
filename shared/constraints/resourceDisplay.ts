export interface ResolvedResource {
  name: string;
  url?: string;
}

const RESOURCE_NAME_ALIASES: Record<string, string[]> = {
  spreadsheet_id: ["title"],
  document_id: ["title"],
  presentation_id: ["presentation_title", "title"],
  file_id: ["file_name"],
  item_id: ["file_name", "document_title", "workbook_title", "presentation_title"],
  folder_id: ["folder_name"],
  parent_id: ["folder_name", "parent_name"],
  drive_id: ["folder_name"],
  space_name: ["space_display_name"],
  channel: ["channel_name"],
  channel_id: ["channel_name"],
  event_id: ["title"],
  message_id: ["subject"],
  thread_id: ["subject"],
  calendar_id: ["calendar_name"],
  user_id: ["user_name"],
};

function resourceNameKey(paramKey: string): string {
  if (paramKey.endsWith("_id")) {
    return `${paramKey.slice(0, -3)}_name`;
  }
  return `${paramKey}_name`;
}

function resourceUrlKey(paramKey: string): string {
  if (paramKey.endsWith("_id")) {
    return `${paramKey.slice(0, -3)}_url`;
  }
  return `${paramKey}_url`;
}

function asRecord(value: unknown): Record<string, unknown> | null {
  if (value != null && typeof value === "object" && !Array.isArray(value)) {
    return value as Record<string, unknown>;
  }
  return null;
}

function stringField(obj: Record<string, unknown> | null, key: string): string | undefined {
  if (!obj) return undefined;
  const value = obj[key];
  return typeof value === "string" && value.length > 0 ? value : undefined;
}

/** HTTPS-only URL for resource links. Non-https values are ignored. */
export function safeHttpsHref(url?: string | null): string | undefined {
  if (!url) return undefined;
  try {
    const parsed = new URL(url);
    if (parsed.protocol === "https:") return parsed.toString();
  } catch {
    return undefined;
  }
  return undefined;
}

function resolved(name: string, url?: string): ResolvedResource {
  const href = safeHttpsHref(url);
  return href ? { name, url: href } : { name };
}

/**
 * Resolve an opaque parameter ID to a display name and optional URL from
 * resource_details. Lookup order: resources[param][id], {param}_name /
 * {param}_url, then legacy aliases such as title / folder_name.
 */
export function lookupResolvedResource(
  paramKey: string,
  rawValue: unknown,
  resourceDetails?: Record<string, unknown> | null,
): ResolvedResource | null {
  if (resourceDetails == null) return null;
  if (typeof rawValue !== "string" && typeof rawValue !== "number") return null;
  const id = String(rawValue);
  if (id.length === 0) return null;

  const resources = asRecord(resourceDetails.resources);
  const byID = asRecord(resources?.[paramKey]);
  const entry = asRecord(byID?.[id]);
  const mappedName = stringField(entry, "name");
  if (mappedName) {
    return resolved(mappedName, stringField(entry, "url"));
  }

  const overlayName = stringField(resourceDetails, resourceNameKey(paramKey));
  if (overlayName) {
    return resolved(overlayName, stringField(resourceDetails, resourceUrlKey(paramKey)));
  }

  for (const alias of RESOURCE_NAME_ALIASES[paramKey] ?? []) {
    const aliasName = stringField(resourceDetails, alias);
    if (aliasName) {
      return resolved(aliasName);
    }
  }
  return null;
}

/** Overlay resolved names onto parameter values for display-template lookup. */
export function overlayResolvedNamesOnParams(
  params: Record<string, unknown>,
  resourceDetails?: Record<string, unknown> | null,
): Record<string, unknown> {
  const out: Record<string, unknown> = { ...params };
  for (const [key, value] of Object.entries(params)) {
    const resolved = lookupResolvedResource(key, value, resourceDetails);
    if (resolved) {
      out[key] = resolved.name;
    }
  }
  if (resourceDetails) {
    return { ...out, ...resourceDetails };
  }
  return out;
}
