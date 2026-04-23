import { useEffect, useRef } from "react";
import { useSearchParams } from "react-router-dom";
import { useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { providerLabel } from "@/lib/labels";

/**
 * Global handler for OAuth callback query parameters. After the backend
 * completes the OAuth flow it redirects the user back to whatever page
 * they started on with `oauth_status`, `oauth_provider`, and optionally
 * `oauth_error` as query params. This hook shows an appropriate toast
 * and strips the params from the URL so they don't persist on refresh.
 *
 * Mount this once near the top of the authenticated app tree (e.g.
 * inside AppLayout).
 */
export function useOAuthCallbackToast() {
  const [searchParams, setSearchParams] = useSearchParams();
  const queryClient = useQueryClient();
  const firedRef = useRef(false);

  useEffect(() => {
    if (firedRef.current) return;

    const oauthStatus = searchParams.get("oauth_status");
    if (!oauthStatus) return;

    firedRef.current = true;
    const oauthProvider = searchParams.get("oauth_provider");

    if (oauthStatus === "success") {
      toast.success(
        `Successfully connected ${oauthProvider ? providerLabel(oauthProvider) : "account"}.`,
      );
      queryClient.invalidateQueries({ queryKey: ["oauth-connections"] });
    } else {
      const oauthError = searchParams.get("oauth_error");
      const label = oauthProvider ? providerLabel(oauthProvider) : "account";
      const detail = oauthError
        ? `Failed to connect ${label}: ${oauthError}`
        : `Failed to connect ${label}. Please try again.`;
      // While Permission Slip is in closed beta, the most common Google
      // failure is "access_denied" from an account that isn't on the test
      // user list. Surface the beta waitlist email so users know what to
      // do next.
      const isGoogleAccessDenied =
        oauthProvider === "google" &&
        !!oauthError &&
        /access.?denied|admin.?policy|verification/i.test(oauthError);
      toast.error(detail, {
        description: isGoogleAccessDenied
          ? "Google blocked the sign-in. If you haven't already, email support@supersuit.tech to be added to the Permission Slip beta."
          : undefined,
      });
    }

    // Remove OAuth params without a full navigation.
    // Note: oauth_connection_id is left for useAutoAssignOAuthCredential
    // which reads it on the agent connector page. On non-agent pages the
    // param is harmless but we don't strip it here to avoid a race where
    // the auto-assign hook hasn't read it yet.
    searchParams.delete("oauth_status");
    searchParams.delete("oauth_provider");
    searchParams.delete("oauth_error");
    setSearchParams(searchParams, { replace: true });
  }, []); // eslint-disable-line react-hooks/exhaustive-deps -- run once on mount
}
