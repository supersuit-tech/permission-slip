import { render, screen } from "@testing-library/react";
import { describe, it, expect } from "vitest";
import { MemoryRouter } from "react-router-dom";
import { UpgradePrompt } from "../UpgradePrompt";

describe("UpgradePrompt", () => {
  it("renders the feature text", () => {
    render(
      <MemoryRouter>
        <UpgradePrompt feature="Upgrade to add more agents." />
      </MemoryRouter>,
    );
    expect(
      screen.getByText(/Upgrade to add more agents/),
    ).toBeInTheDocument();
  });

  it("renders upgrade link to billing page", () => {
    render(
      <MemoryRouter>
        <UpgradePrompt feature="Upgrade to add more agents." />
      </MemoryRouter>,
    );
    const link = screen.getByRole("link", { name: /Upgrade/ });
    expect(link).toHaveAttribute("href", "/billing");
  });

  it("has alert role for screen reader announcement", () => {
    render(
      <MemoryRouter>
        <UpgradePrompt feature="Upgrade to add more agents." />
      </MemoryRouter>,
    );
    expect(screen.getByRole("alert")).toBeInTheDocument();
  });
});
