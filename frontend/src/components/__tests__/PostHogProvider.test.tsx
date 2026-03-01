import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { MemoryRouter } from "react-router-dom";
import {
  CookieConsentProvider,
  useCookieConsent,
} from "../CookieConsentContext";
import { PostHogProvider } from "../PostHogProvider";
import { clearConsentCookie } from "../../test-cookie-helpers";

// Mock the posthog module so we can verify opt-in/opt-out and page view calls.
const mockInitPostHog = vi.fn();
const mockOptInPostHog = vi.fn();
const mockOptOutPostHog = vi.fn();
const mockCapturePageView = vi.fn();

vi.mock("../../lib/posthog", () => ({
  isPostHogConfigured: true,
  initPostHog: (...args: unknown[]) => mockInitPostHog(...args),
  optInPostHog: (...args: unknown[]) => mockOptInPostHog(...args),
  optOutPostHog: (...args: unknown[]) => mockOptOutPostHog(...args),
  capturePageView: (...args: unknown[]) => mockCapturePageView(...args),
}));

/** Test harness that renders PostHogProvider with consent controls. */
function TestHarness() {
  const { consent, accept, reject, reset } = useCookieConsent();
  return (
    <div>
      <span data-testid="consent">{consent ?? "null"}</span>
      <button onClick={accept}>Accept</button>
      <button onClick={reject}>Reject</button>
      <button onClick={reset}>Reset</button>
    </div>
  );
}

function renderWithProviders(initialRoute = "/") {
  return render(
    <MemoryRouter initialEntries={[initialRoute]}>
      <CookieConsentProvider>
        <PostHogProvider>
          <TestHarness />
        </PostHogProvider>
      </CookieConsentProvider>
    </MemoryRouter>,
  );
}

describe("PostHogProvider", () => {
  beforeEach(() => {
    clearConsentCookie();
    mockInitPostHog.mockClear();
    mockOptInPostHog.mockClear();
    mockOptOutPostHog.mockClear();
    mockCapturePageView.mockClear();
  });

  it("initializes PostHog on mount", () => {
    renderWithProviders();
    expect(mockInitPostHog).toHaveBeenCalledTimes(1);
  });

  it("opts out when consent is null (undecided)", () => {
    renderWithProviders();
    // Initial consent is null → should opt out
    expect(mockOptOutPostHog).toHaveBeenCalled();
    expect(mockOptInPostHog).not.toHaveBeenCalled();
  });

  it("opts in when user accepts cookies", async () => {
    renderWithProviders();
    mockOptInPostHog.mockClear();

    await userEvent.click(screen.getByText("Accept"));

    expect(mockOptInPostHog).toHaveBeenCalledTimes(1);
  });

  it("opts out when user rejects cookies", async () => {
    renderWithProviders();
    mockOptOutPostHog.mockClear();

    await userEvent.click(screen.getByText("Reject"));

    expect(mockOptOutPostHog).toHaveBeenCalledTimes(1);
  });

  it("opts out when consent is reset", async () => {
    renderWithProviders();

    // First accept
    await userEvent.click(screen.getByText("Accept"));
    mockOptOutPostHog.mockClear();

    // Then reset
    await userEvent.click(screen.getByText("Reset"));
    expect(mockOptOutPostHog).toHaveBeenCalledTimes(1);
  });

  it("does not capture page views without consent", () => {
    renderWithProviders("/dashboard");
    expect(mockCapturePageView).not.toHaveBeenCalled();
  });

  it("captures a page view after consent is granted", async () => {
    renderWithProviders("/dashboard");
    mockCapturePageView.mockClear();

    await userEvent.click(screen.getByText("Accept"));

    expect(mockCapturePageView).toHaveBeenCalled();
  });

  it("does not initialize PostHog more than once", () => {
    const { rerender } = render(
      <MemoryRouter>
        <CookieConsentProvider>
          <PostHogProvider>
            <TestHarness />
          </PostHogProvider>
        </CookieConsentProvider>
      </MemoryRouter>,
    );

    // Force a re-render
    rerender(
      <MemoryRouter>
        <CookieConsentProvider>
          <PostHogProvider>
            <TestHarness />
          </PostHogProvider>
        </CookieConsentProvider>
      </MemoryRouter>,
    );

    expect(mockInitPostHog).toHaveBeenCalledTimes(1);
  });
});
