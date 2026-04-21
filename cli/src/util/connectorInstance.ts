/**
 * Helpers for merging `connector_instance` into action parameters and parsing
 * `connector_instance_required` error details for interactive recovery.
 */

export interface AvailableConnectorInstance {
  id: string;
  display_name?: string;
}

export function mergeParamsWithConnectorInstance(
  params: unknown,
  instance: string,
): Record<string, unknown> {
  if (typeof params !== "object" || params === null || Array.isArray(params)) {
    throw new Error("--params must be a JSON object when using --instance");
  }
  return {
    ...(params as Record<string, unknown>),
    connector_instance: instance,
  };
}

const uuidRe =
  /^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i;

function looksLikeUUID(s: string): boolean {
  return uuidRe.test(s.trim());
}

/**
 * Normalizes `error.details.available_instances` from the API (objects with
 * id/display_name, or legacy string nicknames only).
 */
export function parseAvailableInstances(
  details: Record<string, unknown> | undefined,
): AvailableConnectorInstance[] {
  const raw = details?.["available_instances"];
  if (!Array.isArray(raw)) {
    return [];
  }
  const out: AvailableConnectorInstance[] = [];
  for (const item of raw) {
    if (typeof item === "string") {
      if (looksLikeUUID(item)) {
        out.push({ id: item.trim() });
      }
      continue;
    }
    if (item !== null && typeof item === "object" && !Array.isArray(item)) {
      const o = item as Record<string, unknown>;
      const id = o["id"];
      if (typeof id !== "string" || !looksLikeUUID(id)) {
        continue;
      }
      const row: AvailableConnectorInstance = { id: id.trim() };
      const dn = o["display_name"];
      if (typeof dn === "string" && dn.length > 0) {
        row.display_name = dn;
      }
      out.push(row);
    }
  }
  return out;
}
