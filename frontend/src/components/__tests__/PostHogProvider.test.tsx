import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { MemoryRouter, Route, Routes, Link } from "react-router-dom";
import { PostHogProvider } from "../PostHogProvider";

const mockInitPostHog = vi.fn();
const mockCapturePageView = vi.fn();

vi.mock("../../lib/posthog", () => ({
  isPostHogConfigured: true,
  initPostHog: (...args: unknown[]) => mockInitPostHog(...args),
  capturePageView: (...args: unknown[]) => mockCapturePageView(...args),
}));

function NavHarness() {
  return (
    <div>
      <Link to="/other">Go other</Link>
      <Routes>
        <Route path="/" element={<span>home</span>} />
        <Route path="/other" element={<span>other</span>} />
      </Routes>
    </div>
  );
}

describe("PostHogProvider", () => {
  beforeEach(() => {
    mockInitPostHog.mockClear();
    mockCapturePageView.mockClear();
  });

  it("initializes PostHog once when configured", () => {
    render(
      <MemoryRouter initialEntries={["/"]}>
        <PostHogProvider>
          <span>child</span>
        </PostHogProvider>
      </MemoryRouter>,
    );
    expect(mockInitPostHog).toHaveBeenCalledTimes(1);
  });

  it("captures a page view on mount and on navigation", async () => {
    const user = userEvent.setup();
    render(
      <MemoryRouter initialEntries={["/"]}>
        <PostHogProvider>
          <NavHarness />
        </PostHogProvider>
      </MemoryRouter>,
    );

    expect(mockCapturePageView).toHaveBeenCalled();

    mockCapturePageView.mockClear();
    await user.click(screen.getByRole("link", { name: /go other/i }));
    expect(mockCapturePageView).toHaveBeenCalled();
  });

  it("does not initialize PostHog more than once across re-renders", () => {
    const { rerender } = render(
      <MemoryRouter>
        <PostHogProvider>
          <span>a</span>
        </PostHogProvider>
      </MemoryRouter>,
    );

    rerender(
      <MemoryRouter>
        <PostHogProvider>
          <span>b</span>
        </PostHogProvider>
      </MemoryRouter>,
    );

    expect(mockInitPostHog).toHaveBeenCalledTimes(1);
  });
});
