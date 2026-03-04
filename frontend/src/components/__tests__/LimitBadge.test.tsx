import { render, screen } from "@testing-library/react";
import { describe, it, expect } from "vitest";
import { LimitBadge } from "../LimitBadge";

describe("LimitBadge", () => {
  it("renders count with limit for free tier", () => {
    render(<LimitBadge current={2} max={3} resource="agents" />);
    expect(screen.getByText("2 / 3 agents")).toBeInTheDocument();
  });

  it("renders count without limit for paid tier", () => {
    render(<LimitBadge current={5} max={null} resource="agents" />);
    expect(screen.getByText("5 agents")).toBeInTheDocument();
  });

  it("applies red styling at limit", () => {
    const { container } = render(
      <LimitBadge current={3} max={3} resource="agents" />,
    );
    const badge = container.firstChild as HTMLElement;
    expect(badge.className).toContain("border-red-300");
  });

  it("applies amber styling near limit (80%+)", () => {
    const { container } = render(
      <LimitBadge current={4} max={5} resource="agents" />,
    );
    const badge = container.firstChild as HTMLElement;
    expect(badge.className).toContain("border-amber-300");
  });

  it("applies no special styling when well below limit", () => {
    const { container } = render(
      <LimitBadge current={1} max={5} resource="agents" />,
    );
    const badge = container.firstChild as HTMLElement;
    expect(badge.className).not.toContain("border-red-300");
    expect(badge.className).not.toContain("border-amber-300");
  });

  it("provides accessible aria-label with remaining count", () => {
    render(<LimitBadge current={2} max={5} resource="agents" />);
    expect(
      screen.getByLabelText("2 of 5 agents used, 3 remaining"),
    ).toBeInTheDocument();
  });

  it("provides aria-label indicating limit reached", () => {
    render(<LimitBadge current={3} max={3} resource="agents" />);
    expect(
      screen.getByLabelText("agents limit reached (3 of 3)"),
    ).toBeInTheDocument();
  });

  it("provides aria-label for unlimited plans", () => {
    render(<LimitBadge current={5} max={null} resource="agents" />);
    expect(
      screen.getByLabelText("5 agents used"),
    ).toBeInTheDocument();
  });
});
