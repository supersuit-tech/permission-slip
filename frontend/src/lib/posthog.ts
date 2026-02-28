/**
 * PostHog product analytics — consent-gated initialization.
 *
 * PostHog is only active when:
 *  1. VITE_POSTHOG_KEY is set (build-time env var)
 *  2. The user has accepted cookies via the consent banner
 *
 * The SDK starts in opted-out state with memory-only persistence (no
 * cookies or localStorage). When consent is granted, capturing is
 * enabled. This ensures zero data is sent before explicit consent.
 */
import posthog from "posthog-js";
import type { PostHogEventName } from "./posthog-events";

const POSTHOG_KEY = import.meta.env.VITE_POSTHOG_KEY as string | undefined;
const POSTHOG_HOST =
  (import.meta.env.VITE_POSTHOG_HOST as string | undefined) ||
  "https://us.i.posthog.com";

/** Whether PostHog is configured (API key is present). */
export const isPostHogConfigured = Boolean(POSTHOG_KEY);

const isDev = import.meta.env.DEV;

/** Dev-mode logging prefixed with [PostHog]. Stripped in production builds. */
function devLog(...args: unknown[]): void {
  if (isDev) {
    console.log("[PostHog]", ...args);
  }
}

/**
 * Initialize the PostHog client. Must be called once at app startup.
 * Starts with capturing disabled — consent must be granted first.
 */
export function initPostHog(): void {
  if (!POSTHOG_KEY) {
    devLog("disabled (VITE_POSTHOG_KEY not set)");
    return;
  }

  devLog("initializing →", POSTHOG_HOST);
  posthog.init(POSTHOG_KEY, {
    api_host: POSTHOG_HOST,
    // Start opted out — no events are sent until the user consents.
    opt_out_capturing_by_default: true,
    // Memory-only persistence avoids writing cookies/localStorage
    // before the user has made a consent choice.
    persistence: "memory",
    // Disable automatic pageview on init — we capture manually on
    // route changes inside PostHogProvider.
    capture_pageview: false,
    // Capture page-leave events (duration, scroll depth) when opted in.
    capture_pageleave: true,
    // Don't send the real IP to PostHog for privacy.
    ip: false,
  });
}

/** Enable PostHog capturing after consent is granted. */
export function optInPostHog(): void {
  if (!isPostHogConfigured) return;
  devLog("consent granted → capturing enabled");
  posthog.opt_in_capturing();
}

/** Disable PostHog capturing when consent is rejected or reset. */
export function optOutPostHog(): void {
  if (!isPostHogConfigured) return;
  devLog("consent revoked → capturing disabled");
  posthog.opt_out_capturing();
}

/**
 * Identify an authenticated user. Uses the opaque Supabase user ID —
 * no PII (email, name, etc.) is sent.
 */
export function identifyUser(userId: string): void {
  if (!isPostHogConfigured) return;
  posthog.identify(userId);
}

/** Reset the PostHog identity on logout. */
export function resetPostHogIdentity(): void {
  if (!isPostHogConfigured) return;
  posthog.reset();
}

/** Capture a page view for the current URL. */
export function capturePageView(): void {
  if (!isPostHogConfigured) return;
  posthog.capture("$pageview");
}

/**
 * Track a product analytics event. No-ops if PostHog is not configured
 * or if the user has not consented (PostHog's internal opt-out flag
 * prevents capture calls from sending data).
 *
 * Event names are typed — use constants from `posthog-events.ts` to
 * get autocomplete and prevent typos.
 */
export function trackEvent(
  eventName: PostHogEventName,
  properties?: Record<string, string | number | boolean>,
): void {
  if (!isPostHogConfigured) return;
  devLog("event:", eventName, properties ?? "");
  posthog.capture(eventName, properties);
}
