import { screen } from "@testing-library/react";
import { render } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { describe, it, expect } from "vitest";
import { BetaBanner } from "../BetaBanner";
import { GITHUB_REPO_URL } from "@/lib/links";

function renderBanner() {
  return render(
    <MemoryRouter>
      <BetaBanner />
    </MemoryRouter>,
  );
}

describe("BetaBanner", () => {
  it("renders the beta notice banner", () => {
    renderBanner();
    expect(screen.getByRole("banner", { name: /beta notice/i })).toBeInTheDocument();
    expect(screen.getByText(/early access/i)).toBeInTheDocument();
  });

  it("links to the open source GitHub repository", () => {
    renderBanner();
    const link = screen.getByRole("link", { name: /open source repository/i });
    expect(link).toHaveAttribute("href", GITHUB_REPO_URL);
    expect(link).toHaveAttribute("target", "_blank");
    expect(link).toHaveAttribute("rel", "noopener noreferrer");
  });

  it("links to the Terms of Service page", () => {
    renderBanner();
    const link = screen.getByRole("link", { name: /terms of service/i });
    expect(link).toHaveAttribute("href", "/policy/terms");
  });
});
