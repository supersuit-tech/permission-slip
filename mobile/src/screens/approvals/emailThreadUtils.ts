/**
 * Parses `context.details.email_thread` for email reply approvals (issue #975).
 * Defensive against partial API payloads — coerces to the OpenAPI EmailThread shape.
 */
import type { components } from "../../api/schema";
import { safeParams } from "./approvalUtils";

export type EmailThread = components["schemas"]["EmailThread"];
export type EmailThreadMessage = components["schemas"]["EmailThreadMessage"];

const EMAIL_REPLY_ACTIONS = new Set([
  "google.send_email_reply",
  "microsoft.send_email_reply",
]);

export function isEmailReplyAction(actionType: string): boolean {
  return EMAIL_REPLY_ACTIONS.has(actionType);
}

function stringArray(v: unknown): string[] {
  if (!Array.isArray(v)) return [];
  return v.filter((x): x is string => typeof x === "string");
}

function parseAttachments(
  v: unknown,
): components["schemas"]["EmailThreadAttachment"][] | undefined {
  if (!Array.isArray(v)) return undefined;
  const out: components["schemas"]["EmailThreadAttachment"][] = [];
  for (const a of v) {
    if (a == null || typeof a !== "object" || Array.isArray(a)) continue;
    const o = a as Record<string, unknown>;
    if (typeof o.filename !== "string") continue;
    const sz =
      typeof o.size_bytes === "number" && !Number.isNaN(o.size_bytes)
        ? o.size_bytes
        : 0;
    out.push({ filename: o.filename, size_bytes: sz });
  }
  return out.length > 0 ? out : undefined;
}

/**
 * Returns a normalized thread when `email_thread` is present in details.
 * Returns null when the key is absent (caller may still show an empty-state card for reply actions).
 */
export function parseEmailThreadDetails(details: unknown): EmailThread | null {
  const d = safeParams(details);
  if (!Object.prototype.hasOwnProperty.call(d, "email_thread")) {
    return null;
  }
  const raw = d.email_thread;
  if (raw == null || typeof raw !== "object" || Array.isArray(raw)) {
    return { subject: "", messages: [] };
  }
  const r = raw as Record<string, unknown>;
  const subject = typeof r.subject === "string" ? r.subject : "";
  const rawMsgs = r.messages;
  const messages: EmailThreadMessage[] = [];
  if (Array.isArray(rawMsgs)) {
    for (const item of rawMsgs) {
      if (item == null || typeof item !== "object" || Array.isArray(item)) {
        continue;
      }
      const m = item as Record<string, unknown>;
      messages.push({
        from: typeof m.from === "string" ? m.from : "",
        to: stringArray(m.to),
        cc: stringArray(m.cc),
        date: typeof m.date === "string" ? m.date : "",
        body_html: typeof m.body_html === "string" ? m.body_html : "",
        body_text: typeof m.body_text === "string" ? m.body_text : "",
        snippet: typeof m.snippet === "string" ? m.snippet : "",
        message_id: typeof m.message_id === "string" ? m.message_id : "",
        truncated:
          typeof m.truncated === "boolean" ? m.truncated : false,
        attachments: parseAttachments(m.attachments),
      });
    }
  }
  return { subject, messages };
}
