/**
 * Parses approval-safe `in_reply_to` metadata from resource_details for
 * protonmail.reply_email (issue #1303).
 */
export interface ProtonInReplyToMetadata {
  subject: string;
  from: string[];
  to: string[];
  date: string;
}

export const PROTON_REPLY_ACTION_TYPE = "protonmail.reply_email";

function stringArray(v: unknown): string[] {
  if (!Array.isArray(v)) return [];
  return v.filter((x): x is string => typeof x === "string");
}

export function parseProtonInReplyTo(
  resourceDetails: Record<string, unknown> | null | undefined,
): ProtonInReplyToMetadata | null {
  if (!resourceDetails) return null;
  const raw = resourceDetails.in_reply_to;
  if (raw == null || typeof raw !== "object" || Array.isArray(raw)) {
    return null;
  }
  const meta = raw as Record<string, unknown>;
  const subject = typeof meta.subject === "string" ? meta.subject : "";
  const from = stringArray(meta.from);
  const to = stringArray(meta.to);
  const date = typeof meta.date === "string" ? meta.date : "";
  if (!subject && from.length === 0 && to.length === 0 && !date) {
    return null;
  }
  return { subject, from, to, date };
}

export function isProtonReplyAction(actionType: string): boolean {
  return actionType === PROTON_REPLY_ACTION_TYPE;
}
