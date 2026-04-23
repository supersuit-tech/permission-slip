import { AlertTriangle, Clock, Mail, ShieldAlert } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";

/**
 * Shown before the user is redirected to Google's OAuth consent screen while
 * Permission Slip is in closed beta. Explains three things users need to know:
 *
 *  1. The app is not yet Google-verified, so Google shows a scary "unverified"
 *     warning — users must click "Advanced" → "Go to Permission Slip (unsafe)"
 *     to proceed.
 *  2. During beta, Google treats them as a test user. We can only add test
 *     users by email address, so they must have emailed support to be added.
 *  3. Google refresh tokens expire every ~7 days while the app is unverified,
 *     so they'll periodically need to reconnect.
 */

export const BETA_SUPPORT_EMAIL = "support@supersuit.tech";

interface GoogleBetaNoticeDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onContinue: () => void;
  mode?: "connect" | "reconnect";
}

export function GoogleBetaNoticeDialog({
  open,
  onOpenChange,
  onContinue,
  mode = "connect",
}: GoogleBetaNoticeDialogProps) {
  const actionLabel = mode === "reconnect" ? "Continue to reconnect" : "Continue to Google";

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>Before you connect Google</DialogTitle>
          <DialogDescription>
            Permission Slip is in closed beta. Please read this before continuing —
            Google&apos;s flow will look a little different than a fully-verified app.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-2 text-sm">
          <NoticeItem
            icon={<Mail className="size-5 text-amber-600 dark:text-amber-400" />}
            title="Your Google account must be added to the beta"
          >
            <p>
              Google only lets unverified apps connect accounts that have been
              individually added to the allowlist. If you haven&apos;t already
              done so, please email{" "}
              <a
                href={`mailto:${BETA_SUPPORT_EMAIL}?subject=${encodeURIComponent(
                  "Permission Slip beta — Google account access",
                )}`}
                className="font-medium underline underline-offset-2"
              >
                {BETA_SUPPORT_EMAIL}
              </a>{" "}
              with the Google account you plan to use. Without this, the
              connection will fail with an &quot;access blocked&quot; error.
            </p>
          </NoticeItem>

          <NoticeItem
            icon={
              <ShieldAlert className="size-5 text-amber-600 dark:text-amber-400" />
            }
            title={'Google will say "Google hasn\'t verified this app"'}
          >
            <p>
              This is expected during beta. To proceed, click{" "}
              <span className="font-medium">Advanced</span> on the warning
              screen, then{" "}
              <span className="font-medium">
                &quot;Go to Permission Slip (unsafe)&quot;
              </span>
              . Your data is not at additional risk — Google simply hasn&apos;t
              finished reviewing our OAuth scopes yet.
            </p>
          </NoticeItem>

          <NoticeItem
            icon={<Clock className="size-5 text-amber-600 dark:text-amber-400" />}
            title="You'll need to reconnect every ~7 days"
          >
            <p>
              Google expires refresh tokens for unverified apps after 7 days.
              Permission Slip will let you know when your Google connection
              needs to be re-authorized — a banner will appear at the top of
              the app. This goes away once we complete Google verification.
            </p>
          </NoticeItem>
        </div>

        <DialogFooter>
          <Button variant="ghost" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button onClick={onContinue}>{actionLabel}</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function NoticeItem({
  icon,
  title,
  children,
}: {
  icon: React.ReactNode;
  title: string;
  children: React.ReactNode;
}) {
  return (
    <div className="flex gap-3 rounded-lg border border-amber-200 bg-amber-50/60 p-3 dark:border-amber-900/40 dark:bg-amber-950/30">
      <div aria-hidden="true" className="mt-0.5 shrink-0">
        {icon}
      </div>
      <div className="space-y-1">
        <p className="font-medium text-foreground">{title}</p>
        <div className="text-muted-foreground leading-relaxed">{children}</div>
      </div>
    </div>
  );
}

/** Re-export for call sites that want to display the warning icon alongside
 *  a "Beta" inline note without opening the full dialog. */
export function GoogleBetaInlineNote({ className = "" }: { className?: string }) {
  return (
    <p
      className={`text-muted-foreground flex items-start gap-1.5 text-xs ${className}`.trim()}
    >
      <AlertTriangle
        aria-hidden="true"
        className="mt-0.5 size-3.5 shrink-0 text-amber-600 dark:text-amber-400"
      />
      <span>
        Google is in closed beta. Your account must be allowlisted — email{" "}
        <a
          href={`mailto:${BETA_SUPPORT_EMAIL}`}
          className="underline underline-offset-2"
        >
          {BETA_SUPPORT_EMAIL}
        </a>{" "}
        to join. You&apos;ll see an &quot;unverified app&quot; warning from
        Google and will need to reconnect every ~7 days.
      </span>
    </p>
  );
}
