import { useCookieConsent } from "./CookieConsentContext";
import { Button } from "./ui/button";

/**
 * GDPR-compliant cookie consent banner. Shows at the bottom of the viewport
 * until the user accepts or rejects non-essential cookies. Both options are
 * equally prominent (no dark patterns).
 */
export function CookieConsentBanner() {
  const { consent, accept, reject } = useCookieConsent();

  // Already made a choice — don't render.
  if (consent !== null) return null;

  return (
    <div
      role="region"
      aria-label="Cookie consent"
      className="fixed inset-x-0 bottom-0 z-50 border-t bg-card p-4 shadow-lg md:p-6"
    >
      <div className="mx-auto flex max-w-[1200px] flex-col gap-4 md:flex-row md:items-center md:justify-between">
        <p className="text-sm text-card-foreground">
          We use essential cookies for authentication. Accepting enables
          privacy-focused analytics (PostHog) to help us improve the product
          — no advertising or tracking. See our{" "}
          <a href="/policy/cookies" className="underline underline-offset-4 hover:text-primary">
            Cookie Policy
          </a>{" "}
          for details.
        </p>
        <div className="flex shrink-0 gap-3">
          <Button variant="outline" size="sm" onClick={reject}>
            Reject All
          </Button>
          <Button size="sm" onClick={accept}>
            Accept All
          </Button>
        </div>
      </div>
    </div>
  );
}
