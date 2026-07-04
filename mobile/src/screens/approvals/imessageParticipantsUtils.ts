/**
 * Parses raw iMessage participant handles from resource_details (#1400).
 */
export function parseImessageParticipants(
  resourceDetails: Record<string, unknown> | null | undefined,
): string[] | null {
  if (!resourceDetails) return null;
  const raw = resourceDetails.participants;
  if (!Array.isArray(raw) || raw.length === 0) return null;
  const handles = raw.filter((v): v is string => typeof v === "string" && v.length > 0);
  return handles.length > 0 ? handles : null;
}

export function formatImessageParticipants(handles: string[]): string {
  return handles.join(", ");
}
