import { Link } from "react-router-dom";
import { cn } from "@/lib/utils";
import { useCookieConsent } from "./CookieConsentContext";

interface FooterProps {
  className?: string;
}

const linkClass = "hover:text-foreground transition-colors";

/**
 * Shared site footer used across all layouts (app, auth, policy pages).
 * Renders policy links, support mailto, and a "Manage Cookies" button
 * that re-opens the consent banner.
 */
export function Footer({ className }: FooterProps) {
  const { reset: resetConsent } = useCookieConsent();

  return (
    <footer className={cn("text-xs text-muted-foreground", className)}>
      <div className="flex flex-wrap gap-x-4 gap-y-2">
        <Link to="/policy/privacy" className={linkClass}>Privacy Policy</Link>
        <Link to="/policy/terms" className={linkClass}>Terms of Service</Link>
        <Link to="/policy/cookies" className={linkClass}>Cookie Policy</Link>
        <a href="mailto:support@supersuit.tech" className={linkClass}>Support</a>
        <button type="button" onClick={resetConsent} className={linkClass}>Manage Cookies</button>
      </div>
    </footer>
  );
}
