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
  it("shows retention limit for free plan", () => {
    renderBanner({ days: 7 });
    expect(screen.getByText(/Showing events from the last 7 days/)).toBeInTheDocument();
    expect(screen.getByText("Upgrade")).toBeInTheDocument();
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
});
