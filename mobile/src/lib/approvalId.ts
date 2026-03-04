/**
 * Validation for approval IDs used in deep links and notification payloads.
 *
 * Shared across useApproval (deep link handling) and useNotificationNavigation
 * (push notification tap handling) to ensure consistent validation at the
 * boundary between external input and API calls.
 */

/**
 * Matches the approval_id format from the OpenAPI spec (e.g. "appr_xyz789").
 * Defense-in-depth: prevents API calls with obviously invalid IDs from
 * crafted deep link URLs or malformed notification payloads.
 */
export const APPROVAL_ID_PATTERN = /^appr_[a-zA-Z0-9]{6,64}$/;

/** Returns true if the given string matches the expected approval ID format. */
export function isValidApprovalId(id: string): boolean {
  return APPROVAL_ID_PATTERN.test(id);
}
