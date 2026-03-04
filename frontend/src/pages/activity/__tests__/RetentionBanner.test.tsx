import { render, screen } from "@testing-library/react";
import { describe, it, expect } from "vitest";
import { MemoryRouter } from "react-router-dom";
import { RetentionBanner } from "../RetentionBanner";

function renderBanner(retention: { days: number; grace_period_ends_at?: string | null }) {
  return render(
    <MemoryRouter>
      <RetentionBanner retention={retention} />
    </MemoryRouter>,
  );
}

describe("RetentionBanner", () => {
  it("shows retention limit for free plan with plan name", () => {
    renderBanner({ days: 7 });
    expect(screen.getByText(/Showing last 7 days \(Free plan\)/)).toBeInTheDocument();
    expect(screen.getByText("Upgrade")).toBeInTheDocument();
  });

  it("links upgrade to billing page", () => {
    renderBanner({ days: 7 });
    const link = screen.getByRole("link", { name: "Upgrade" });
    expect(link).toHaveAttribute("href", "/billing");
  });

  it("hides banner for paid plan (90-day retention)", () => {
    const { container } = renderBanner({ days: 90 });
    expect(container.firstChild).toBeNull();
  });

  it("shows grace period warning with date", () => {
    renderBanner({
      days: 90,
      grace_period_ends_at: "2026-03-08T14:30:00Z",
    });
    expect(screen.getByText(/90-day audit history is preserved until/)).toBeInTheDocument();
    expect(screen.getByText("Upgrade")).toBeInTheDocument();
  });

  it("always shows 7-day retention after grace period ends", () => {
    renderBanner({
      days: 7,
      grace_period_ends_at: "2026-03-08T14:30:00Z",
    });
    expect(screen.getByText(/retention will drop to 7 days/)).toBeInTheDocument();
  });
});
