import { useCallback } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { useAuth } from "@/auth/AuthContext";

/**
 * Wraps the auth signOut call with consistent error handling:
 * logs to console and shows a toast on failure.
 *
 * Used by UserMenu so the pattern isn't duplicated.
 */
export function useSignOut() {
  const { signOut } = useAuth();
  const queryClient = useQueryClient();

  const handleSignOut = useCallback(async () => {
    queryClient.clear();

    const { error } = await signOut();
    if (error) {
      console.error("Sign out failed:", error);
      toast.error("Sign out failed. Please try again.");
    }
  }, [signOut, queryClient]);

  return handleSignOut;
}
