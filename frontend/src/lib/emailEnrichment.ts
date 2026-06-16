/**
 * Email enrichment detection for approval prompts.
 *
 * Proton Mail message actions are enriched at approval-creation time with
 * human-readable email metadata (subject/from/to/date) stored in
 * resource_details (#1304). Enrichment is best-effort — when it fails the
 * approval falls back to raw IMAP UIDs. This helper detects that fallback so
 * the UI can tell the approver the details are unavailable rather than
 * silently showing bare ids.
 */

/** Action types whose approval prompts are expected to carry email metadata. */
const EMAIL_DETAIL_ACTION_TYPES = new Set([
  "protonmail.read_email",
  "protonmail.archive_email",
  "protonmail.reply_email",
  "protonmail.mark_read",
  "protonmail.mark_unread",
  "protonmail.flag",
  "protonmail.unflag",
  "protonmail.move_to_folder",
  "protonmail.delete",
  "protonmail.apply_label",
  "protonmail.remove_label",
]);

/** Keys enrichment writes: flat single-message fields, batch map, or reply ref. */
const ENRICHMENT_KEYS = ["subject", "messages", "in_reply_to"];

/**
 * True when an action's approval prompt should show email metadata but the
 * enrichment is missing (resolution failed or returned nothing).
 */
export function emailDetailsUnavailable(
  actionType: string,
  resourceDetails?: Record<string, unknown> | null,
): boolean {
  if (!EMAIL_DETAIL_ACTION_TYPES.has(actionType)) return false;
  if (!resourceDetails) return true;
  return !ENRICHMENT_KEYS.some((key) => resourceDetails[key] != null);
}
