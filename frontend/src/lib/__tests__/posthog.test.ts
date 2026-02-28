import { describe, it, expect, beforeEach, vi } from "vitest";
import { PostHogEvents } from "../posthog-events";

// Mock posthog-js before importing our module.
const mockInit = vi.fn();
const mockCapture = vi.fn();
const mockIdentify = vi.fn();
const mockReset = vi.fn();
const mockOptIn = vi.fn();
const mockOptOut = vi.fn();

vi.mock("posthog-js", () => ({
  default: {
    init: mockInit,
    capture: mockCapture,
    identify: mockIdentify,
    reset: mockReset,
    opt_in_capturing: mockOptIn,
    opt_out_capturing: mockOptOut,
  },
}));

describe("posthog utilities", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    mockInit.mockClear();
    mockCapture.mockClear();
    mockIdentify.mockClear();
    mockReset.mockClear();
    mockOptIn.mockClear();
    mockOptOut.mockClear();
  });

  describe("when VITE_POSTHOG_KEY is not set", () => {
    it("isPostHogConfigured is false", async () => {
      const mod = await import("../posthog");
      expect(mod.isPostHogConfigured).toBe(false);
    });

    it("initPostHog does not call posthog.init", async () => {
      const mod = await import("../posthog");
      mod.initPostHog();
      expect(mockInit).not.toHaveBeenCalled();
    });

    it("trackEvent does not call posthog.capture", async () => {
      const mod = await import("../posthog");
      mod.trackEvent(PostHogEvents.APPROVAL_APPROVED, { key: "value" });
      expect(mockCapture).not.toHaveBeenCalled();
    });

    it("identifyUser does not call posthog.identify", async () => {
      const mod = await import("../posthog");
      mod.identifyUser("user-123");
      expect(mockIdentify).not.toHaveBeenCalled();
    });

    it("resetPostHogIdentity does not call posthog.reset", async () => {
      const mod = await import("../posthog");
      mod.resetPostHogIdentity();
      expect(mockReset).not.toHaveBeenCalled();
    });

    it("optInPostHog does not call posthog.opt_in_capturing", async () => {
      const mod = await import("../posthog");
      mod.optInPostHog();
      expect(mockOptIn).not.toHaveBeenCalled();
    });

    it("optOutPostHog does not call posthog.opt_out_capturing", async () => {
      const mod = await import("../posthog");
      mod.optOutPostHog();
      expect(mockOptOut).not.toHaveBeenCalled();
    });

    it("capturePageView does not call posthog.capture", async () => {
      const mod = await import("../posthog");
      mod.capturePageView();
      expect(mockCapture).not.toHaveBeenCalled();
    });
  });
});
