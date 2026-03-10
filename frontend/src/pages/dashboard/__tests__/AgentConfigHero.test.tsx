import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { describe, it, expect } from "vitest";
import { AgentConfigHero } from "../AgentConfigHero";

function renderHero(agentId: number, agentName?: string) {
  return render(
    <MemoryRouter>
      <AgentConfigHero agentId={agentId} agentName={agentName} />
    </MemoryRouter>,
  );
}

describe("AgentConfigHero", () => {
  it("renders headline with agent name when provided", () => {
    renderHero(42, "Claude Bot");

    expect(
      screen.getByText(/Claude Bot is ready.*now give it superpowers/),
    ).toBeInTheDocument();
  });

  it("renders fallback headline when no agent name is provided", () => {
    renderHero(42);

    expect(
      screen.getByText(/Your agent is ready.*now give it superpowers/),
    ).toBeInTheDocument();
  });

  it("shows completed register step and pending config steps", () => {
    renderHero(42, "My Agent");

    expect(screen.getByText("Register agent")).toBeInTheDocument();
    expect(screen.getByText("Add a connector")).toBeInTheDocument();
    expect(screen.getByText("Set permissions")).toBeInTheDocument();
  });

  it("links CTA to the agent config page with agent name", () => {
    renderHero(42, "Claude Bot");

    const link = screen.getByRole("link", { name: "Configure Claude Bot" });
    expect(link).toHaveAttribute("href", "/agents/42");
  });

  it("links CTA with fallback name when no agent name", () => {
    renderHero(42);

    const link = screen.getByRole("link", {
      name: "Configure Your agent",
    });
    expect(link).toHaveAttribute("href", "/agents/42");
  });
});
