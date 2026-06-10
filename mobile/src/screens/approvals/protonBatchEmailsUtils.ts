/**
 * Parses the batch `messages` map written by Proton Mail enrichment
 * (resource_details.messages, keyed by IMAP UID) for batch email actions.
 */
export interface ProtonBatchEmail {
  uid: string;
  subject: string;
  from: string[];
  to: string[];
  date: string;
}

function stringArray(v: unknown): string[] {
  if (!Array.isArray(v)) return [];
  return v.filter((x): x is string => typeof x === "string");
}

/**
 * Returns the batch emails sorted by UID, or null unless at least two
 * messages resolved — single-message actions use flat subject/from/to/date
 * fields instead.
 */
export function parseProtonBatchEmails(
  resourceDetails: Record<string, unknown> | null | undefined,
): ProtonBatchEmail[] | null {
  if (!resourceDetails) return null;
  const raw = resourceDetails.messages;
  if (raw == null || typeof raw !== "object" || Array.isArray(raw)) return null;

  const emails: ProtonBatchEmail[] = [];
  for (const [uid, value] of Object.entries(raw as Record<string, unknown>)) {
    if (value == null || typeof value !== "object" || Array.isArray(value)) continue;
    const meta = value as Record<string, unknown>;
    emails.push({
      uid,
      subject: typeof meta.subject === "string" ? meta.subject : "",
      from: stringArray(meta.from),
      to: stringArray(meta.to),
      date: typeof meta.date === "string" ? meta.date : "",
    });
  }
  if (emails.length < 2) return null;
  emails.sort((a, b) => Number(a.uid) - Number(b.uid));
  return emails;
}
