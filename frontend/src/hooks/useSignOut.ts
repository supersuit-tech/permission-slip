import { useCallback } from "react";
import { toast } from "sonner";
import { useAuth } from "@/auth/AuthContext";
import { MFA_PENDING_ENROLLMENT_KEY } from "@/auth/mfaPendingEnrollment";

/**
 * Wraps the auth signOut call with consistent error handling:
 * logs to console and shows a toast on failure.
 *
 * Used by UserMenu and MfaChallengePage so the pattern isn't duplicated.
 */
export function useSignOut() {
  const { signOut } = useAuth();

  const handleSignOut = useCallback(async () => {
    // Clear MFA enrollment state before sign-out so it doesn't leak
    // to the next user who signs in on the same tab.
    try {
      sessionStorage.removeItem(MFA_PENDING_ENROLLMENT_KEY);
    } catch {
      // sessionStorage unavailable
    }

    const { error } = await signOut();
    if (error) {
      console.error("Sign out failed:", error);
      toast.error("Sign out failed. Please try again.");
    }
  }, [signOut]);

  return handleSignOut;
}
