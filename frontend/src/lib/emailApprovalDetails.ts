import { emailDetailsUnavailable } from "@/lib/emailEnrichment";

export interface EmailApprovalDetails {
  subject: string | null;
  from: string | null;
  date: string | null;
}

function stringArray(value: unknown): string[] {
  if (typeof value === "string" && value.length > 0) return [value];
  if (!Array.isArray(value)) return [];
  return value.filter((item): item is string => typeof item === "string" && item.length > 0);
}

function formatRecipients(value: unknown): string | null {
  const parts = stringArray(value);
  if (parts.length === 0) return null;
  if (parts.length <= 3) return parts.join(", ");
  return `${parts[0]}, ${parts[1]}, and ${parts.length - 2} more`;
}

function parseFlatEmailMeta(meta: Record<string, unknown>): EmailApprovalDetails {
  const subject = typeof meta.subject === "string" && meta.subject.length > 0 ? meta.subject : null;
  const from = formatRecipients(meta.from);
  const date = typeof meta.date === "string" && meta.date.length > 0 ? meta.date : null;
  return { subject, from, date };
}

/**
 * Extract subject, from, and date from approval resource_details for email actions.
 * Supports flat single-message fields, reply context, and the first batch message.
 */
export function parseEmailApprovalDetails(
  actionType: string,
  resourceDetails?: Record<string, unknown> | null,
): EmailApprovalDetails | null {
  if (emailDetailsUnavailable(actionType, resourceDetails)) {
    return null;
  }
  if (!resourceDetails) return null;

  if (resourceDetails.in_reply_to && typeof resourceDetails.in_reply_to === "object") {
    return parseFlatEmailMeta(resourceDetails.in_reply_to as Record<string, unknown>);
  }

  if (resourceDetails.messages && typeof resourceDetails.messages === "object" && !Array.isArray(resourceDetails.messages)) {
    const messages = resourceDetails.messages as Record<string, unknown>;
    const first = Object.values(messages).find(
      (entry) => entry && typeof entry === "object" && !Array.isArray(entry),
    );
    if (first && typeof first === "object") {
      return parseFlatEmailMeta(first as Record<string, unknown>);
    }
  }

  return parseFlatEmailMeta(resourceDetails);
}

/** True when at least one email detail field resolved. */
export function hasPartialEmailApprovalDetails(details: EmailApprovalDetails | null): boolean {
  if (!details) return false;
  return details.subject !== null || details.from !== null || details.date !== null;
}

/** True when the approval detail UI should render the email metadata section. */
export function shouldShowEmailApprovalSection(
  actionType: string,
  resourceDetails?: Record<string, unknown> | null,
): boolean {
  if (emailDetailsUnavailable(actionType, resourceDetails)) return true;
  return hasPartialEmailApprovalDetails(parseEmailApprovalDetails(actionType, resourceDetails));
}
