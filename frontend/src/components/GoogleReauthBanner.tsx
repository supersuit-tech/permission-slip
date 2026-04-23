import { useState } from "react";
import { Link } from "react-router-dom";
import { AlertTriangle, LogIn, X } from "lucide-react";
import { useAuth } from "@/auth/AuthContext";
import { useOAuthConnections } from "@/hooks/useOAuthConnections";
import { getOAuthAuthorizeUrl } from "@/lib/oauth";
import { Button } from "@/components/ui/button";
import { GoogleBetaNoticeDialog } from "./GoogleBetaNoticeDialog";

const DISMISS_KEY_PREFIX = "google-reauth-banner-dismissed:";

/**
 * App-wide banner that appears when any of the user's Google OAuth connections
 * are in the `needs_reauth` state. Shown prominently in AppLayout because
 * Google refresh tokens for unverified apps expire after 7 days during beta,
 * and a silently-broken Google connection is a common support complaint.
 *
 * Dismissal is per-connection-ID and stored in sessionStorage, so the banner
 * reappears if the connection still needs reauth in a new session but doesn't
 * pester the user who's deliberately ignoring it right now.
 */
export function GoogleReauthBanner() {
  const { session } = useAuth();
  const { connections } = useOAuthConnections();
  const [betaNoticeOpen, setBetaNoticeOpen] = useState(false);
  const [pendingReconnect, setPendingReconnect] = useState<string | null>(null);
  const [dismissedTick, setDismissedTick] = useState(0);

  const needsReauthGoogleConnections = connections.filter(
    (c) => c.provider === "google" && c.status === "needs_reauth",
  );

  // Re-read dismissal state each render (cheap). `dismissedTick` forces a
  // rerender after the user dismisses.
  void dismissedTick;
  const visibleConnections = needsReauthGoogleConnections.filter(
    (c) =>
      typeof sessionStorage === "undefined" ||
      !sessionStorage.getItem(`${DISMISS_KEY_PREFIX}${c.id}`),
  );

  if (visibleConnections.length === 0) {
    return null;
  }

  function handleReconnectClick(connectionId: string) {
    setPendingReconnect(connectionId);
    setBetaNoticeOpen(true);
  }

  function handleContinueToGoogle() {
    if (!session?.access_token || !pendingReconnect) return;
    window.location.href = getOAuthAuthorizeUrl(
      "google",
      session.access_token,
      { replaceId: pendingReconnect },
    );
  }

  function handleDismiss(connectionId: string) {
    if (typeof sessionStorage !== "undefined") {
      sessionStorage.setItem(`${DISMISS_KEY_PREFIX}${connectionId}`, "1");
    }
    setDismissedTick((n) => n + 1);
  }

  const hasMultiple = visibleConnections.length > 1;
  const first = visibleConnections[0]!;

  return (
    <div className="mb-4 space-y-2" role="status">
      <div className="flex flex-col gap-3 rounded-lg border border-amber-300 bg-amber-50 px-4 py-3 text-sm text-amber-900 shadow-sm sm:flex-row sm:items-center sm:justify-between dark:border-amber-700 dark:bg-amber-950/50 dark:text-amber-200">
        <div className="flex min-w-0 items-start gap-3">
          <AlertTriangle className="mt-0.5 size-5 shrink-0" aria-hidden="true" />
          <div className="min-w-0 space-y-0.5">
            <p className="font-medium">
              {hasMultiple
                ? `${visibleConnections.length} Google connections need to be reconnected`
                : "Your Google connection needs to be reconnected"}
            </p>
            <p className="text-xs opacity-90">
              While Permission Slip is in closed beta, Google expires refresh
              tokens every ~7 days. Agents that depend on Google will fail until
              you reconnect.{" "}
              <Link to="/settings" className="underline underline-offset-2">
                Manage all connections
              </Link>
              .
            </p>
          </div>
        </div>
        <div className="flex shrink-0 items-center gap-2">
          {!hasMultiple && (
            <Button
              size="sm"
              variant="outline"
              className="border-amber-400 bg-white/70 hover:bg-white dark:border-amber-700 dark:bg-amber-950/40"
              onClick={() => handleReconnectClick(first.id)}
            >
              <LogIn className="size-4" />
              Reconnect Google
            </Button>
          )}
          {hasMultiple && (
            <Button
              size="sm"
              variant="outline"
              className="border-amber-400 bg-white/70 hover:bg-white dark:border-amber-700 dark:bg-amber-950/40"
              asChild
            >
              <Link to="/settings">
                <LogIn className="size-4" />
                Go to Settings
              </Link>
            </Button>
          )}
          <Button
            size="icon"
            variant="ghost"
            aria-label="Dismiss until next session"
            onClick={() => {
              // Dismiss all currently-visible banners with one click.
              visibleConnections.forEach((c) => handleDismiss(c.id));
            }}
          >
            <X className="size-4" />
          </Button>
        </div>
      </div>

      <GoogleBetaNoticeDialog
        open={betaNoticeOpen}
        onOpenChange={(nextOpen) => {
          setBetaNoticeOpen(nextOpen);
          if (!nextOpen) setPendingReconnect(null);
        }}
        onContinue={handleContinueToGoogle}
        mode="reconnect"
      />
    </div>
  );
}
