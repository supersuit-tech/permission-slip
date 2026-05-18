/**
 * PostHog product analytics — enabled only when `VITE_POSTHOG_KEY` is set at
 * build time (self-hosted operators opt in explicitly).
 */
import posthog from "posthog-js";
import type { PostHogEventName } from "./posthog-events";

export { PostHogEvents } from "./posthog-events";
export type { PostHogEventName } from "./posthog-events";

const POSTHOG_KEY = import.meta.env.VITE_POSTHOG_KEY as string | undefined;
const POSTHOG_HOST =
  (import.meta.env.VITE_POSTHOG_HOST as string | undefined) ||
  "https://us.i.posthog.com";

/** Whether PostHog is configured (API key is present). */
export const isPostHogConfigured = Boolean(POSTHOG_KEY);

const isDev = import.meta.env.DEV;

function devLog(...args: unknown[]): void {
  if (isDev) {
    console.log("[PostHog]", ...args);
  }
}

/**
 * Initialize the PostHog client. Must be called once at app startup.
 * Capturing is on when a key is configured; otherwise this is a no-op.
 */
export function initPostHog(): void {
  if (!POSTHOG_KEY) {
    devLog("disabled (VITE_POSTHOG_KEY not set)");
    return;
  }

  devLog("initializing →", POSTHOG_HOST);
  posthog.init(POSTHOG_KEY, {
    api_host: POSTHOG_HOST,
    ui_host: POSTHOG_HOST,
    opt_out_capturing_by_default: false,
    persistence: "localStorage",
    capture_pageview: false,
    capture_pageleave: true,
    ip: false,
  });
}

export function optInPostHog(): void {
  if (!isPostHogConfigured) return;
  devLog("capturing enabled");
  posthog.opt_in_capturing();
}

export function optOutPostHog(): void {
  if (!isPostHogConfigured) return;
  devLog("capturing disabled");
  posthog.opt_out_capturing();
}

export function identifyUser(userId: string): void {
  if (!isPostHogConfigured) return;
  posthog.identify(userId);
}

export function resetPostHogIdentity(): void {
  if (!isPostHogConfigured) return;
  posthog.reset();
}

export function capturePageView(): void {
  if (!isPostHogConfigured) return;
  posthog.capture("$pageview");
}

export function trackEvent(
  eventName: PostHogEventName,
  properties?: Record<string, string | number | boolean>,
): void {
  if (!isPostHogConfigured) return;
  devLog("event:", eventName, properties ?? "");
  posthog.capture(eventName, properties);
}
