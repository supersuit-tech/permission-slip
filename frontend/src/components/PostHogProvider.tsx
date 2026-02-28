import { useEffect, useRef, type ReactNode } from "react";
import { useLocation } from "react-router-dom";
import { useCookieConsent } from "./CookieConsentContext";
import {
  initPostHog,
  isPostHogConfigured,
  optInPostHog,
  optOutPostHog,
  capturePageView,
} from "../lib/posthog";

/**
 * Bridges the cookie consent state with PostHog's opt-in/opt-out API.
 *
 * - Initializes PostHog once on mount (opted out by default).
 * - Watches the consent state and enables/disables capturing accordingly.
 * - Captures page views on React Router navigation when opted in.
 *
 * Must be placed inside both <BrowserRouter> and <CookieConsentProvider>.
 */
export function PostHogProvider({ children }: { children: ReactNode }) {
  const { consent } = useCookieConsent();
  const location = useLocation();
  const initializedRef = useRef(false);

  // Initialize PostHog once on mount.
  useEffect(() => {
    if (!isPostHogConfigured || initializedRef.current) return;
    initPostHog();
    initializedRef.current = true;
  }, []);

  // React to consent changes.
  useEffect(() => {
    if (!initializedRef.current) return;

    if (consent === "accepted") {
      optInPostHog();
    } else {
      optOutPostHog();
    }
  }, [consent]);

  // Capture page views on route changes.
  useEffect(() => {
    if (!initializedRef.current) return;
    capturePageView();
  }, [location.pathname]);

  return <>{children}</>;
}
