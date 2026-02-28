import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { ErrorBoundary } from "../ErrorBoundary";

function ThrowingChild(): React.ReactNode {
  throw new Error("Test error");
}

describe("ErrorBoundary", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    vi.spyOn(console, "error").mockImplementation(() => {});
  });

  it("renders children when no error occurs", () => {
    render(
      <ErrorBoundary>
        <div>Working content</div>
      </ErrorBoundary>
    );
    expect(screen.getByText("Working content")).toBeInTheDocument();
  });

  it("renders default fallback when a child throws", () => {
    render(
      <ErrorBoundary>
        <ThrowingChild />
      </ErrorBoundary>
    );
    expect(screen.getByText("Something went wrong")).toBeInTheDocument();
    expect(
      screen.getByText(/An unexpected error occurred/)
    ).toBeInTheDocument();
    expect(screen.getByText("Refresh Page")).toBeInTheDocument();
  });

  it("renders custom fallback when provided", () => {
    render(
      <ErrorBoundary fallback={<div>Custom error UI</div>}>
        <ThrowingChild />
      </ErrorBoundary>
    );
    expect(screen.getByText("Custom error UI")).toBeInTheDocument();
  });

  it("logs the error to console", () => {
    const consoleSpy = vi.spyOn(console, "error");
    render(
      <ErrorBoundary>
        <ThrowingChild />
      </ErrorBoundary>
    );
    expect(consoleSpy).toHaveBeenCalledWith(
      "ErrorBoundary caught:",
      expect.any(Error),
      expect.any(String)
    );
  });

  it("calls window.location.reload when Refresh Page is clicked", async () => {
    const reloadMock = vi.fn();
    Object.defineProperty(window, "location", {
      value: { ...window.location, reload: reloadMock },
      writable: true,
    });

    render(
      <ErrorBoundary>
        <ThrowingChild />
      </ErrorBoundary>
    );
    await userEvent.click(screen.getByText("Refresh Page"));
    expect(reloadMock).toHaveBeenCalled();
  });
});
