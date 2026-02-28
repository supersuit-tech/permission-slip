/**
 * SessionStorage helpers for the MFA pending-enrollment marker.
 *
 * Extracted into its own module so that consumers like useSignOut don't
 * need to import from the MfaEnrollmentFlow page component (which would
 * pull React UI code into the bundle unnecessarily).
 */

export const MFA_PENDING_ENROLLMENT_KEY = "mfa_pending_enrollment";

interface PendingEnrollment {
  userId: string;
}

export function savePendingEnrollment(userId: string) {
  try {
    const enrollment: PendingEnrollment = { userId };
    sessionStorage.setItem(MFA_PENDING_ENROLLMENT_KEY, JSON.stringify(enrollment));
  } catch {
    // sessionStorage unavailable (e.g. private browsing quota exceeded)
  }
}

/**
 * Check whether there is a pending enrollment marker for the given user,
 * returning true if the entry exists and belongs to this user.
 */
export function hasPendingEnrollment(userId: string): boolean {
  try {
    const raw = sessionStorage.getItem(MFA_PENDING_ENROLLMENT_KEY);
    if (!raw) return false;
    const parsed: unknown = JSON.parse(raw);
    if (typeof parsed !== "object" || parsed === null) return false;

    const storedUserId =
      "userId" in parsed &&
      typeof (parsed as PendingEnrollment).userId === "string"
        ? (parsed as PendingEnrollment).userId
        : "";

    // Reject entries that belong to a different user.
    return storedUserId === userId;
  } catch {
    return false;
  }
}

export function clearPendingEnrollment() {
  try {
    sessionStorage.removeItem(MFA_PENDING_ENROLLMENT_KEY);
  } catch {
    // ignore
  }
}
