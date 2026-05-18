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
  it("shows retention limit with link to account settings", () => {
    renderBanner({ days: 7 });
    expect(screen.getByText(/Showing last 7 days/)).toBeInTheDocument();
    const link = screen.getByRole("link", { name: /Account/i });
    expect(link).toHaveAttribute("href", "/settings/account");
  });

  it("hides banner for 90-day retention without grace period", () => {
    const { container } = renderBanner({ days: 90 });
    expect(container.firstChild).toBeNull();
  });

  it("shows grace period warning with date and account link", () => {
    renderBanner({
      days: 90,
      grace_period_ends_at: "2026-03-08T14:30:00Z",
    });
    expect(screen.getByText(/90-day audit history is preserved until/)).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /Account/i })).toHaveAttribute(
      "href",
      "/settings/account",
    );
  });

  it("mentions 7-day retention after grace period in copy", () => {
    renderBanner({
      days: 7,
      grace_period_ends_at: "2026-03-08T14:30:00Z",
    });
    expect(screen.getByText(/retention will drop to 7 days/)).toBeInTheDocument();
  });
});
