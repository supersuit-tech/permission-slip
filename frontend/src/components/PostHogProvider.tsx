import { useEffect, useRef, type ReactNode } from "react";
import { useLocation } from "react-router-dom";
import {
  initPostHog,
  isPostHogConfigured,
  capturePageView,
} from "../lib/posthog";

/**
 * Initializes PostHog when configured and captures SPA navigations as page views.
 * Must be placed inside <BrowserRouter>.
 */
export function PostHogProvider({ children }: { children: ReactNode }) {
  const location = useLocation();
  const initializedRef = useRef(false);

  useEffect(() => {
    if (!isPostHogConfigured || initializedRef.current) return;
    initPostHog();
    initializedRef.current = true;
  }, []);

  useEffect(() => {
    if (!initializedRef.current) return;
    capturePageView();
  }, [location.pathname]);

  return <>{children}</>;
}
